// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentChanges(t *testing.T) {
	tests := []struct {
		name     string
		events   []lsp.TextDocumentContentChangeEvent
		expected document.Changes
	}{
		{
			name:     "empty events",
			events:   []lsp.TextDocumentContentChangeEvent{},
			expected: document.Changes{},
		},
		{
			name: "single change with range",
			events: []lsp.TextDocumentContentChangeEvent{
				{
					Range: &lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 5},
					},
					Text: "hello",
				},
			},
			expected: document.Changes{
				&contentChange{
					text: "hello",
					rng: &document.Range{
						Start: document.Pos{Line: 0, Column: 0},
						End:   document.Pos{Line: 0, Column: 5},
					},
				},
			},
		},
		{
			name: "single change without range (full document)",
			events: []lsp.TextDocumentContentChangeEvent{
				{
					Range: nil,
					Text:  "full document text",
				},
			},
			expected: document.Changes{
				&contentChange{
					text: "full document text",
					rng:  nil,
				},
			},
		},
		{
			name: "multiple changes",
			events: []lsp.TextDocumentContentChangeEvent{
				{
					Range: &lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 5},
					},
					Text: "first",
				},
				{
					Range: &lsp.Range{
						Start: lsp.Position{Line: 1, Character: 10},
						End:   lsp.Position{Line: 1, Character: 20},
					},
					Text: "second",
				},
			},
			expected: document.Changes{
				&contentChange{
					text: "first",
					rng: &document.Range{
						Start: document.Pos{Line: 0, Column: 0},
						End:   document.Pos{Line: 0, Column: 5},
					},
				},
				&contentChange{
					text: "second",
					rng: &document.Range{
						Start: document.Pos{Line: 1, Column: 10},
						End:   document.Pos{Line: 1, Column: 20},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DocumentChanges(tt.events)
			require.Len(t, result, len(tt.expected))

			for i := range result {
				assert.Equal(t, tt.expected[i].Text(), result[i].Text())
				assert.Equal(t, tt.expected[i].Range(), result[i].Range())
			}
		})
	}
}

func TestContentChange_Text(t *testing.T) {
	change := &contentChange{
		text: "test text",
		rng:  nil,
	}
	assert.Equal(t, "test text", change.Text())
}

func TestContentChange_Range(t *testing.T) {
	tests := []struct {
		name     string
		change   *contentChange
		expected *document.Range
	}{
		{
			name: "nil range",
			change: &contentChange{
				text: "test",
				rng:  nil,
			},
			expected: nil,
		},
		{
			name: "with range",
			change: &contentChange{
				text: "test",
				rng: &document.Range{
					Start: document.Pos{Line: 0, Column: 0},
					End:   document.Pos{Line: 0, Column: 5},
				},
			},
			expected: &document.Range{
				Start: document.Pos{Line: 0, Column: 0},
				End:   document.Pos{Line: 0, Column: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.change.Range()
			assert.Equal(t, tt.expected, result)
		})
	}
}
