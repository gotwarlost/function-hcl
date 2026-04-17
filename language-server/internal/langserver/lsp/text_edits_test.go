// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

// mockChange is a test implementation of document.Change
type mockChange struct {
	text string
	rng  *document.Range
}

func (m *mockChange) Text() string {
	return m.text
}

func (m *mockChange) Range() *document.Range {
	return m.rng
}

func TestTextEditsFromDocumentChanges(t *testing.T) {
	tests := []struct {
		name     string
		changes  document.Changes
		expected []lsp.TextEdit
	}{
		{
			name:     "empty changes",
			changes:  document.Changes{},
			expected: []lsp.TextEdit{},
		},
		{
			name: "single change",
			changes: document.Changes{
				&mockChange{
					text: "new text",
					rng: &document.Range{
						Start: document.Pos{Line: 0, Column: 0},
						End:   document.Pos{Line: 0, Column: 5},
					},
				},
			},
			expected: []lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 5},
					},
					NewText: "new text",
				},
			},
		},
		{
			name: "multiple changes",
			changes: document.Changes{
				&mockChange{
					text: "first",
					rng: &document.Range{
						Start: document.Pos{Line: 0, Column: 0},
						End:   document.Pos{Line: 0, Column: 3},
					},
				},
				&mockChange{
					text: "second",
					rng: &document.Range{
						Start: document.Pos{Line: 1, Column: 5},
						End:   document.Pos{Line: 1, Column: 10},
					},
				},
			},
			expected: []lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 3},
					},
					NewText: "first",
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 5},
						End:   lsp.Position{Line: 1, Character: 10},
					},
					NewText: "second",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextEditsFromDocumentChanges(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTextEdits(t *testing.T) {
	tests := []struct {
		name           string
		textEdits      []lang.TextEdit
		snippetSupport bool
		expected       []lsp.TextEdit
	}{
		{
			name:           "empty edits",
			textEdits:      []lang.TextEdit{},
			snippetSupport: false,
			expected:       []lsp.TextEdit{},
		},
		{
			name: "single edit without snippet support",
			textEdits: []lang.TextEdit{
				{
					Range:   hcl.Range{Start: hcl.Pos{Line: 1, Column: 1}, End: hcl.Pos{Line: 1, Column: 5}},
					NewText: "plain text",
					Snippet: "snippet ${1:text}",
				},
			},
			snippetSupport: false,
			expected: []lsp.TextEdit{
				{
					Range:   lsp.Range{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 4}},
					NewText: "plain text",
				},
			},
		},
		{
			name: "single edit with snippet support",
			textEdits: []lang.TextEdit{
				{
					Range:   hcl.Range{Start: hcl.Pos{Line: 1, Column: 1}, End: hcl.Pos{Line: 1, Column: 5}},
					NewText: "plain text",
					Snippet: "snippet ${1:text}",
				},
			},
			snippetSupport: true,
			expected: []lsp.TextEdit{
				{
					Range:   lsp.Range{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 4}},
					NewText: "snippet ${1:text}",
				},
			},
		},
		{
			name: "multiple edits with snippet support",
			textEdits: []lang.TextEdit{
				{
					Range:   hcl.Range{Start: hcl.Pos{Line: 1, Column: 1}, End: hcl.Pos{Line: 1, Column: 5}},
					NewText: "first",
					Snippet: "first ${1:value}",
				},
				{
					Range:   hcl.Range{Start: hcl.Pos{Line: 2, Column: 1}, End: hcl.Pos{Line: 2, Column: 3}},
					NewText: "second",
					Snippet: "second ${2:value}",
				},
			},
			snippetSupport: true,
			expected: []lsp.TextEdit{
				{
					Range:   lsp.Range{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 4}},
					NewText: "first ${1:value}",
				},
				{
					Range:   lsp.Range{Start: lsp.Position{Line: 1, Character: 0}, End: lsp.Position{Line: 1, Character: 2}},
					NewText: "second ${2:value}",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextEdits(tt.textEdits, tt.snippetSupport)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInsertTextFormat(t *testing.T) {
	tests := []struct {
		name           string
		snippetSupport bool
		expected       lsp.InsertTextFormat
	}{
		{
			name:           "snippet support enabled",
			snippetSupport: true,
			expected:       lsp.SnippetTextFormat,
		},
		{
			name:           "snippet support disabled",
			snippetSupport: false,
			expected:       lsp.PlainTextTextFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := insertTextFormat(tt.snippetSupport)
			assert.Equal(t, tt.expected, result)
		})
	}
}
