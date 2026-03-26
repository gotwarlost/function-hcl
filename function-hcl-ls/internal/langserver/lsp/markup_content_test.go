// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/stretchr/testify/assert"
)

func TestMarkupContent(t *testing.T) {
	tests := []struct {
		name        string
		input       lang.MarkupContent
		mdSupported bool
		expected    lsp.MarkupContent
	}{
		{
			name:        "plain text content with markdown supported",
			input:       lang.PlainText("Hello, World!"),
			mdSupported: true,
			expected: lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: "Hello, World!",
			},
		},
		{
			name:        "plain text content without markdown supported",
			input:       lang.PlainText("Hello, World!"),
			mdSupported: false,
			expected: lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: "Hello, World!",
			},
		},
		{
			name:        "markdown content with markdown supported",
			input:       lang.Markdown("# Heading\n\nSome **bold** text"),
			mdSupported: true,
			expected: lsp.MarkupContent{
				Kind:  lsp.Markdown,
				Value: "# Heading\n\nSome **bold** text",
			},
		},
		{
			name:        "markdown content without markdown supported",
			input:       lang.Markdown("# Heading\n\nSome **bold** text"),
			mdSupported: false,
			expected: lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: "Heading\n\nSome bold text",
			},
		},
		{
			name:        "markdown with code blocks without markdown support",
			input:       lang.Markdown("Use `code` for inline code"),
			mdSupported: false,
			expected: lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: "Use code for inline code",
			},
		},
		{
			name:        "markdown with links without markdown support",
			input:       lang.Markdown("Visit [example](https://example.com)"),
			mdSupported: false,
			expected: lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: "Visit example",
			},
		},
		{
			name:        "empty content",
			input:       lang.PlainText(""),
			mdSupported: true,
			expected: lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: "",
			},
		},
		{
			name:        "markdown with multiple elements without markdown support",
			input:       lang.Markdown("## Section\n\n- Item 1\n- Item 2\n\n**Important**: Use `config`"),
			mdSupported: false,
			expected: lsp.MarkupContent{
				Kind:  lsp.PlainText,
				Value: "Section\n\n- Item 1\n- Item 2\n\nImportant: Use config",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := markupContent(tt.input, tt.mdSupported)
			assert.Equal(t, tt.expected.Kind, result.Kind)
			assert.Equal(t, tt.expected.Value, result.Value)
		})
	}
}
