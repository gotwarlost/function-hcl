package loader

import (
	"io"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// ExtractObjects returns runtime objects from the supplied reader that is assumed
// to have one or more YAML documents.
func ExtractObjects(reader io.Reader) ([]runtime.Object, error) {
	return extractObjects(io.NopCloser(reader))
}

// LoadReader returns schemas found in the supplied reader.
func LoadReader(reader io.Reader) (*resource.Schemas, error) {
	objs, err := ExtractObjects(reader)
	if err != nil {
		return nil, err
	}
	return resource.ToSchemas(objs...), nil
}
