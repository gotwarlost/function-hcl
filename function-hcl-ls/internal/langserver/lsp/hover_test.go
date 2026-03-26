// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestHoverData(t *testing.T) {
	tests := []struct {
		name     string
		data     *lang.HoverData
		caps     lsp.TextDocumentClientCapabilities
		expected *lsp.Hover
	}{
		{
			name:     "nil hover data",
			data:     nil,
			caps:     lsp.TextDocumentClientCapabilities{},
			expected: nil,
		},
		{
			name: "hover with markdown support",
			data: &lang.HoverData{
				Content: lang.Markdown("# Heading\n\nSome **bold** text"),
				Range: hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
				},
			},
			caps: lsp.TextDocumentClientCapabilities{
				Hover: lsp.HoverClientCapabilities{
					ContentFormat: []lsp.MarkupKind{lsp.Markdown},
				},
			},
			expected: &lsp.Hover{
				Contents: lsp.MarkupContent{
					Kind:  lsp.Markdown,
					Value: "# Heading\n\nSome **bold** text",
				},
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 9},
				},
			},
		},
		{
			name: "hover without markdown support",
			data: &lang.HoverData{
				Content: lang.Markdown("# Heading\n\nSome **bold** text"),
				Range: hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
				},
			},
			caps: lsp.TextDocumentClientCapabilities{
				Hover: lsp.HoverClientCapabilities{
					ContentFormat: []lsp.MarkupKind{},
				},
			},
			expected: &lsp.Hover{
				Contents: lsp.MarkupContent{
					Kind:  lsp.PlainText,
					Value: "Heading\n\nSome bold text",
				},
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 9},
				},
			},
		},
		{
			name: "hover with plaintext content",
			data: &lang.HoverData{
				Content: lang.PlainText("Simple text"),
				Range: hcl.Range{
					Start: hcl.Pos{Line: 2, Column: 5, Byte: 20},
					End:   hcl.Pos{Line: 2, Column: 15, Byte: 30},
				},
			},
			caps: lsp.TextDocumentClientCapabilities{
				Hover: lsp.HoverClientCapabilities{
					ContentFormat: []lsp.MarkupKind{lsp.Markdown},
				},
			},
			expected: &lsp.Hover{
				Contents: lsp.MarkupContent{
					Kind:  lsp.PlainText,
					Value: "Simple text",
				},
				Range: lsp.Range{
					Start: lsp.Position{Line: 1, Character: 4},
					End:   lsp.Position{Line: 1, Character: 14},
				},
			},
		},
		{
			name: "hover with empty content",
			data: &lang.HoverData{
				Content: lang.Markdown(""),
				Range: hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:   hcl.Pos{Line: 1, Column: 5, Byte: 4},
				},
			},
			caps: lsp.TextDocumentClientCapabilities{
				Hover: lsp.HoverClientCapabilities{
					ContentFormat: []lsp.MarkupKind{lsp.Markdown},
				},
			},
			expected: &lsp.Hover{
				Contents: lsp.MarkupContent{
					Kind:  lsp.Markdown,
					Value: "",
				},
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 4},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HoverData(tt.data, tt.caps)
			assert.Equal(t, tt.expected, result)
		})
	}
}
