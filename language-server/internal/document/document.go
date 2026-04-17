// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package document encapsulates processing for HCL text documents.
package document

import (
	"path/filepath"
	"time"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document/source"
)

// Document represents a text document of interest.
type Document struct {
	Dir        DirHandle     // the directory where the doc lives.
	Filename   string        // the file name.
	ModTime    time.Time     // last modified time.
	LanguageID string        // language ID as supplied by the language client.
	Version    int           // document version used to ensure edits are in sequence.
	Text       []byte        // the text of the document as a byte slice.
	Lines      []source.Line // text separated into lines to enable byte offset computation for position conversions.
}

// FullPath returns the full filesystem path of the document.
func (d *Document) FullPath() string {
	return filepath.Join(d.Dir.Path(), d.Filename)
}
