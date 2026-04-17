package crds

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func sanitize(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

// Image is the image being processed for CRDs. When processing local files, the image is given a name
// but its version is empty.
type Image struct {
	Name    string
	Version string
}

func (im Image) Filename() string {
	return fmt.Sprintf("%s.yaml", sanitize(im.Name))
}

func (im Image) String() string {
	if im.Version == "" {
		return im.Name
	}
	return im.Name + ":" + im.Version
}

// ObjectMetadata is the metadata of the object that needs to be written.
type ObjectMetadata struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

func (o ObjectMetadata) Filename() string {
	gvk := o.GVK
	name := o.Name
	if o.Namespace != "" {
		name += "." + o.Namespace
	}
	g := sanitize(gvk.Group)
	if g == "" {
		g = "core"
	}
	k := sanitize(gvk.Kind)
	n := sanitize(name)
	return fmt.Sprintf("%s.%s.%s.yaml", n, k, g)
}

func (o ObjectMetadata) String() string {
	name := o.Name
	if o.Namespace != "" {
		name += o.Namespace + "." + name
	}
	return fmt.Sprintf("%s: %s", o.GVK.String(), name)
}

// Writer writes CRD content.
type Writer interface {
	// StartImage is called when a new image is processed.
	StartImage(image Image) error
	// Write allows the writer to write an object of the supplied GVK (one of CRD or XRD), its name and content.
	Write(meta ObjectMetadata, content []byte) error
	// EndImage is called when the current image has been fully processed. It will always be called once
	// for every StartImage call.
	EndImage() error
}

// Extractor extracts CRDs and XRDs from YAML files, and images found as part of configuration
// and provider definitions.
type Extractor struct {
	localImageName string
	writer         Writer
}

// NewExtractor creates an Extractor.
func NewExtractor(w Writer, localImageName string) *Extractor {
	if localImageName == "" {
		localImageName = "local-objects"
	}
	return &Extractor{writer: w, localImageName: localImageName}
}

// ExtractCRDs extracts CRDs and XRDs from the supplied readers, including those in images
// referenced in Configuration and Provider data.
func (d *Extractor) ExtractCRDs(inputs ...io.Reader) error {
	im := Image{Name: d.localImageName}
	if err := d.writer.StartImage(im); err != nil {
		return err
	}
	images, err := d.processInputs(inputs...)
	if err != nil {
		_ = d.writer.EndImage()
		return err
	}
	if err = d.writer.EndImage(); err != nil {
		return err
	}

	for _, img := range images {
		if err := d.processImage(img); err != nil {
			return err
		}
	}
	return nil
}
