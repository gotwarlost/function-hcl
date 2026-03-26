// Package store implements a store for HCL documents open in the editor and tracks unsaved changes for each of them.
package store

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document/source"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/utils/logging"
)

// Store is the document store.
type Store struct {
	l      sync.RWMutex
	docs   map[document.Handle]*document.Document
	logger *log.Logger
}

// New creates an empty store.
func New() *Store {
	return &Store{
		docs:   map[document.Handle]*document.Document{},
		logger: logging.LoggerFor(logging.ModuleDocStore),
	}
}

// Open creates an in-memory representation of the supplied document. The document should not
// have been already opened.
func (s *Store) Open(dh document.Handle, langId string, version int, text []byte) error {
	s.l.Lock()
	defer s.l.Unlock()
	doc := s.docs[dh]
	if doc != nil {
		return fmt.Errorf("document %s already exists", dh.FullPath())
	}
	s.docs[dh] = &document.Document{
		Dir:        dh.Dir,
		Filename:   dh.Filename,
		ModTime:    time.Now().UTC(),
		LanguageID: langId,
		Version:    version,
		Text:       text,
		Lines:      source.MakeSourceLines(dh.Filename, text),
	}
	return nil
}

// Update modifies the store to update document text.
func (s *Store) Update(dh document.Handle, newText []byte, newVersion int) error {
	s.l.Lock()
	defer s.l.Unlock()
	doc := s.docs[dh]
	if doc == nil {
		return document.NotFound(dh.FullPath())
	}
	if newVersion <= doc.Version {
		return fmt.Errorf("version not ascending: %d => %d", doc.Version, newVersion)
	}
	s.docs[dh] = &document.Document{
		Dir:        dh.Dir,
		Filename:   dh.Filename,
		ModTime:    time.Now().UTC(),
		LanguageID: doc.LanguageID,
		Version:    newVersion,
		Text:       newText,
		Lines:      source.MakeSourceLines(dh.Filename, newText),
	}
	return nil
}

// Close removes the supplied document from the store.
func (s *Store) Close(dh document.Handle) error {
	s.l.Lock()
	defer s.l.Unlock()
	doc := s.docs[dh]
	if doc == nil {
		return document.NotFound(dh.FullPath())
	}
	delete(s.docs, dh)
	return nil
}

// HasOpenDocuments returns true if the supplied directory has any open documents.
func (s *Store) HasOpenDocuments(dirHandle document.DirHandle) bool {
	s.l.RLock()
	defer s.l.RUnlock()
	for key := range s.docs {
		if key.Dir == dirHandle {
			return true
		}
	}
	return false
}

// IsDocumentOpen returns true if it has been opened in the store.
func (s *Store) IsDocumentOpen(dh document.Handle) bool {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.docs[dh] != nil
}

// Get returns the document for the supplied handle or an error if the document
// could not be found.
func (s *Store) Get(dh document.Handle) (*document.Document, error) {
	s.l.RLock()
	defer s.l.RUnlock()
	d := s.docs[dh]
	if d == nil {
		return nil, document.NotFound(dh.FullPath())
	}
	return d, nil
}

// List returns all documents under the specified directory.
func (s *Store) List(dirHandle document.DirHandle) []*document.Document {
	s.l.RLock()
	defer s.l.RUnlock()
	var ret []*document.Document
	for key, doc := range s.docs {
		if key.Dir == dirHandle {
			ret = append(ret, doc)
		}
	}
	return ret
}
