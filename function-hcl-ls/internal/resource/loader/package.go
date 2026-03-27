package loader

import (
	"archive/tar"
	"io"
	"path/filepath"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

// CrossplanePackage loads schemas from crossplane packages gotten from an OCI registry.
type CrossplanePackage struct {
	imageRef string
}

// NewCrossplanePackage creates a CrossplanePackage
func NewCrossplanePackage(imageRef string) *CrossplanePackage {
	return &CrossplanePackage{imageRef: imageRef}
}

func (p *CrossplanePackage) Load() (*resource.Schemas, error) {
	objs, err := p.ExtractObjects()
	if err != nil {
		return nil, err
	}
	return resource.ToSchemas(objs...), nil
}

func (p *CrossplanePackage) ExtractObjects() ([]runtime.Object, error) {
	ref, err := name.ParseReference(p.imageRef)
	if err != nil {
		return nil, errors.Wrap(err, errBadReference)
	}
	img, err := crane.Pull(ref.String())
	if err != nil {
		return nil, errors.Wrap(err, errFetchPackage)
	}
	rc, err := p.getPackageYamlStream(img)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return ExtractObjects(rc)
}

// getPackageYamlStream extracts the package YAML stream from the downloaded image.
// Code copied from the crossplane source.
func (p *CrossplanePackage) getPackageYamlStream(img v1.Image) (io.ReadCloser, error) {
	// Get image manifest.
	manifest, err := img.Manifest()
	if err != nil {
		return nil, errors.Wrap(err, errGetManifest)
	}

	// Check that the image has less than the maximum allowed number of layers.
	if nLayers := len(manifest.Layers); nLayers > maxLayers {
		return nil, errors.Errorf(errFmtMaxManifestLayers, nLayers, maxLayers)
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
			return nil, errors.Wrap(err, errFetchLayer)
		}
		if err := validate.Layer(layer); err != nil {
			return nil, errors.Wrap(err, errValidateLayer)
		}
		tarc, err = layer.Uncompressed()
		if err != nil {
			return nil, errors.Wrap(err, errGetUncompressed)
		}
	}
	// If we still don't have content then we need to flatten image filesystem.
	if !foundAnnotated {
		if err := validate.Image(img); err != nil {
			return nil, errors.Wrap(err, errValidateImage)
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
			return nil, errors.Wrapf(err, errFmtNoPackageFileFound, read, foundAnnotated)
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
