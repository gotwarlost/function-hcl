package crds

import (
	"archive/tar"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	packageFile                = "package.yaml"
	errBadReference            = "package tag is not a valid reference"
	errFetchPackage            = "failed to fetch package from remote"
	errGetManifest             = "failed to get package image manifest from remote"
	errFetchLayer              = "failed to fetch annotated base layer from remote"
	errGetUncompressed         = "failed to get uncompressed contents from layer"
	errMultipleAnnotatedLayers = "package is invalid due to multiple annotated base layers"
	errFmtNoPackageFileFound   = "couldn't find \"" + packageFile + "\" file after checking %d files in the archive (annotated layer: %v)"
	errFmtMaxManifestLayers    = "package has %d layers, but only %d are allowed"
	errValidateLayer           = "invalid package layer"
	errValidateImage           = "invalid package image"
)

const (
	layerAnnotation     = "io.crossplane.xpkg"
	baseAnnotationValue = "base"
	// maxLayers is the maximum number of layers an image can have.
	maxLayers = 256
)

// metadata is the subset of k8s attributes we need for filtering and processing objects.
type metadata struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	} `json:"metadata"`
}

func (m metadata) ObjectMetadata() ObjectMetadata {
	slashParts := strings.SplitN(m.APIVersion, "/", 2)
	var group, version string
	if len(slashParts) == 1 {
		group = ""
		version = slashParts[0]
	} else {
		group = slashParts[0]
		version = slashParts[1]
	}
	return ObjectMetadata{
		GVK: schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    m.Kind,
		},
		Namespace: m.Metadata.Namespace,
		Name:      m.Metadata.Name,
	}
}

type configPackageReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Package    string `json:"package"`
	Version    string `json:"version"`
}

// processInputs processes all inputs for the supplied image source.
func (d *Extractor) processInputs(inputs ...io.Reader) (images []string, finalErr error) {
	seenImages := map[string]bool{}

	for _, multiYamlDoc := range inputs {
		reader := yamlutil.NewYAMLReader(bufio.NewReader(multiYamlDoc))
		for {
			doc, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("read yaml document: %w", err)
			}
			var meta metadata
			jsonDoc, err := yamlutil.ToJSON(doc)
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(jsonDoc, &meta)
			if err != nil {
				return nil, err
			}
			switch {
			case meta.Kind == "CustomResourceDefinition" && strings.HasPrefix(meta.APIVersion, "apiextensions.k8s.io/"):
				if err = d.writer.Write(meta.ObjectMetadata(), doc); err != nil {
					return nil, err
				}
			case meta.Kind == "CompositeResourceDefinition" && strings.HasPrefix(meta.APIVersion, "apiextensions.crossplane.io/"):
				if err = d.writer.Write(meta.ObjectMetadata(), doc); err != nil {
					return nil, err
				}
			case meta.Kind == "Provider" && strings.HasPrefix(meta.APIVersion, "pkg.crossplane.io/"):
				var p map[string]any
				err = json.Unmarshal(jsonDoc, &p)
				if err != nil {
					return nil, err
				}
				pkg, err := fieldpath.Pave(p).GetString("spec.package")
				if err != nil {
					return nil, err
				}
				if pkg != "" {
					seenImages[pkg] = true
				}
			case meta.Kind == "Configuration" && strings.HasPrefix(meta.APIVersion, "pkg.crossplane.io/"):
				var p map[string]any
				err = json.Unmarshal(jsonDoc, &p)
				if err != nil {
					return nil, err
				}
				paved := fieldpath.Pave(p)
				var deps []configPackageReference
				err := paved.GetValueInto("spec.dependsOn", &deps)
				if err != nil {
					return nil, err
				}
				for _, dep := range deps {
					if dep.Kind == "Provider" && strings.HasPrefix(dep.APIVersion, "pkg.crossplane.io/") {
						seenImages[fmt.Sprintf("%s:%s", dep.Package, dep.Version)] = true
					}
				}
			}
		}
	}
	var ret []string
	for k := range seenImages {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret, nil
}

func (d *Extractor) processImage(img string) (finalErr error) {
	ref, err := name.ParseReference(img, name.WithDefaultTag("latest"))
	if err != nil {
		return fmt.Errorf("%s: %w", errBadReference, err)
	}
	im := Image{
		Name:    ref.Context().String(),
		Version: ref.Identifier(),
	}
	if err = d.writer.StartImage(im); err != nil {
		return err
	}
	defer func() {
		err = d.writer.EndImage()
		if err != nil && finalErr == nil {
			finalErr = err
		}
	}()

	image, err := crane.Pull(ref.String())
	if err != nil {
		return fmt.Errorf("%s: %w", errFetchPackage, err)
	}
	rc, err := d.getPackageYamlStream(image)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	_, err = d.processInputs(rc)
	if err != nil {
		return err
	}
	return nil
}

// getPackageYamlStream extracts the package YAML stream from the downloaded image.
// Code copied from the crossplane source.
func (d *Extractor) getPackageYamlStream(img v1.Image) (io.ReadCloser, error) {
	// Get image manifest.
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errGetManifest, err)
	}

	// Check that the image has less than the maximum allowed number of layers.
	if nLayers := len(manifest.Layers); nLayers > maxLayers {
		return nil, fmt.Errorf(errFmtMaxManifestLayers, nLayers, maxLayers)
	}

	var tarc io.ReadCloser
	// determine if the image is using annotated layers.
	foundAnnotated := false
	for _, l := range manifest.Layers {
		if a, ok := l.Annotations[layerAnnotation]; !ok || a != baseAnnotationValue {
			continue
		}
		if foundAnnotated {
			return nil, errors.New(errMultipleAnnotatedLayers)
		}
		foundAnnotated = true
		layer, err := img.LayerByDigest(l.Digest)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errFetchLayer, err)
		}
		if err := validate.Layer(layer); err != nil {
			return nil, fmt.Errorf("%s: %w", errValidateLayer, err)
		}
		tarc, err = layer.Uncompressed()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errGetUncompressed, err)
		}
	}
	// If we still don't have content then we need to flatten image filesystem.
	if !foundAnnotated {
		if err := validate.Image(img); err != nil {
			return nil, fmt.Errorf("%s: %w", errValidateImage, err)
		}
		tarc = mutate.Extract(img)
	}
	// the ReadCloser is an uncompressed tarball, either consisting of annotated
	// layer contents or flattened filesystem content. Either way, we only want
	// the package YAML stream.
	t := tar.NewReader(tarc)
	var read int
	for {
		h, err := t.Next()
		if err != nil {
			return nil, fmt.Errorf(errFmtNoPackageFileFound+": %w", read, foundAnnotated, err)
		}
		if filepath.Base(h.Name) == packageFile {
			break
		}
		read++
	}
	return &joinedReadCloser{r: t, c: tarc}, nil
}

// joinedReadCloser joins a reader and a closer. It is typically used in the
// context of a ReadCloser being wrapped by a Reader.
type joinedReadCloser struct {
	r io.Reader
	c io.Closer
}

// Read calls the underlying reader Read method.
func (r *joinedReadCloser) Read(b []byte) (int, error) {
	return r.r.Read(b)
}

// Close closes the closer for the JoinedReadCloser.
func (r *joinedReadCloser) Close() error {
	return r.c.Close()
}
