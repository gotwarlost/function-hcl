package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/resource"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/resource/loader"
)

var errNoChanges = fmt.Errorf("no changes")

func fileInfoHash(fi fs.FileInfo) string {
	h := sha256.New()
	_, _ = fmt.Fprint(h, fi.ModTime().UnixNano())
	_, _ = fmt.Fprint(h, fi.Size())
	return hex.EncodeToString(h.Sum(nil))
}

// fileSchema tracks the schemas found in a single loaded file.
type fileSchema struct {
	path     string
	checksum string
	schema   *resource.Schemas
}

func (f *fileSchema) Path() string {
	return f.path
}

func (f *fileSchema) Checksum() string {
	return f.checksum
}

func (f *fileSchema) Schema() *resource.Schemas {
	return f.schema
}

// fileStore tracks all files that contain CRD information independent of which
// source it is used in.
type fileStore struct {
	l     sync.RWMutex
	files map[string]*fileSchema
}

func newFileStore() *fileStore {
	return &fileStore{
		files: map[string]*fileSchema{},
	}
}

func (f *fileStore) get(path string) *fileSchema {
	f.l.RLock()
	defer f.l.RUnlock()
	return f.files[path]
}

func (f *fileStore) put(fs *fileSchema) {
	f.l.Lock()
	defer f.l.Unlock()
	f.files[fs.path] = fs
}

//nolint:unused
func (f *fileStore) remove(filePath string) {
	f.l.Lock()
	defer f.l.Unlock()
	delete(f.files, filePath)
}

//nolint:unused
func (f *fileStore) list() map[string]*fileSchema {
	f.l.RLock()
	defer f.l.RUnlock()
	list := map[string]*fileSchema{}
	for k, v := range f.files {
		list[k] = v
	}
	return list
}

func (f *fileStore) add(filePath string) error {
	have := f.get(filePath)

	st, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	hashString := fileInfoHash(st)
	if have != nil && have.checksum == hashString {
		return errNoChanges
	}

	schema, err := loader.NewFile(filePath).Load()
	if err != nil {
		return err
	}
	f.put(&fileSchema{
		path:     filePath,
		checksum: hashString,
		schema:   schema,
	})
	return nil
}

func (f *fileStore) getSchema(path string) *resource.Schemas {
	f.l.RLock()
	defer f.l.RUnlock()
	fi := f.files[path]
	if fi == nil {
		return emptySchema
	}
	return fi.Schema()
}
