// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package filesystem provides an FS abstraction over operating system files overlaid
// with unsaved editor content.
package filesystem

import (
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/logging"
)

// Filesystem provides io/fs.FS compatible two-layer read-only filesystem
// with preferred source being DocumentStore and native OS FS acting as fallback.
//
// This allows for reading files in a directory while reflecting unsaved changes.
type Filesystem struct {
	osFs     osFs
	docStore DocumentStore

	logger *log.Logger
}

// DocumentStore proves list and get facilities for unsaved documents.
type DocumentStore interface {
	Get(document.Handle) (*document.Document, error)
	List(document.DirHandle) []*document.Document
}

// New creates an OS filesystem overlaid with the supplied document store.
func New(docStore DocumentStore) *Filesystem {
	return &Filesystem{
		osFs:     osFs{},
		docStore: docStore,
		logger:   logging.LoggerFor(logging.ModuleFilesystem),
	}
}

// ReadFile provides the content at the supplied path.
func (f *Filesystem) ReadFile(name string) ([]byte, error) {
	doc, err := f.docStore.Get(document.HandleFromPath(name))
	if err != nil {
		if document.IsNotFound(err) {
			return f.osFs.ReadFile(name)
		}
		return nil, err
	}
	return doc.Text, err
}

// ReadDir provides entries under the supplied directory path.
func (f *Filesystem) ReadDir(dir string) ([]fs.DirEntry, error) {
	dirHandle := document.DirHandleFromPath(dir)
	docList := f.docStore.List(dirHandle)

	osList, err := f.osFs.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("OS FS: %w", err)
	}

	list := documentsAsDirEntries(docList)
	for _, osEntry := range osList {
		if entryIsInList(list, osEntry) {
			continue
		}
		list = append(list, osEntry)
	}

	return list, nil
}

func entryIsInList(list []fs.DirEntry, entry fs.DirEntry) bool {
	for _, di := range list {
		if di.Name() == entry.Name() {
			return true
		}
	}
	return false
}

// Open implements fs.FS.
func (f *Filesystem) Open(name string) (fs.File, error) {
	doc, err := f.docStore.Get(document.HandleFromPath(name))
	if err != nil {
		if document.IsNotFound(err) {
			return f.osFs.Open(name)
		}
		return nil, err
	}
	return documentAsFile(doc), err
}

// Stat provides file information at the supplied path.
func (f *Filesystem) Stat(name string) (os.FileInfo, error) {
	doc, err := f.docStore.Get(document.HandleFromPath(name))
	if err != nil {
		if document.IsNotFound(err) {
			return f.osFs.Stat(name)
		}
		return nil, err
	}

	return documentAsFileInfo(doc), err
}
