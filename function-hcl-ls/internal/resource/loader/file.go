package loader

import (
	"os"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource"
)

// File loads schemas from YAML files.
type File struct {
	file string
}

// NewFile returns a loader that loads schemas from a YAML file that
// can contains multiple YAML documents each representing a CRD.
func NewFile(file string) *File {
	return &File{file: file}
}

// Load returns schemas found in the YAML file.
func (f *File) Load() (*resource.Schemas, error) {
	fh, err := os.Open(f.file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	return LoadReader(fh)
}
