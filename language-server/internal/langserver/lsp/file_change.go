// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
)

type contentChange struct {
	text string
	rng  *document.Range
}

func getContentChange(chEvent lsp.TextDocumentContentChangeEvent) document.Change {
	return &contentChange{
		text: chEvent.Text,
		rng:  lspRangeToDocRange(chEvent.Range),
	}
}

func DocumentChanges(events []lsp.TextDocumentContentChangeEvent) document.Changes {
	changes := make(document.Changes, len(events))
	for i, event := range events {
		ch := getContentChange(event)
		changes[i] = ch
	}
	return changes
}

func (fc *contentChange) Text() string {
	return fc.text
}

func (fc *contentChange) Range() *document.Range {
	return fc.rng
}
