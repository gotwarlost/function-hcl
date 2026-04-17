// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/stretchr/testify/assert"
)

func TestCodeActions_AsSlice(t *testing.T) {
	tests := []struct {
		name     string
		actions  CodeActions
		expected []lsp.CodeActionKind
	}{
		{
			name: "single code action",
			actions: CodeActions{
				SourceFormatAllTerraform: true,
			},
			expected: []lsp.CodeActionKind{SourceFormatAllTerraform},
		},
		{
			name: "multiple code actions sorted",
			actions: CodeActions{
				"source.z":                true,
				"source.a":                true,
				SourceFormatAllTerraform:  true,
				lsp.SourceOrganizeImports: true,
			},
			expected: []lsp.CodeActionKind{
				"source.a",
				SourceFormatAllTerraform,
				lsp.SourceOrganizeImports,
				"source.z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.actions.AsSlice()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCodeActions_Only(t *testing.T) {
	tests := []struct {
		name     string
		actions  CodeActions
		only     []lsp.CodeActionKind
		expected CodeActions
	}{
		{
			name: "filter to empty",
			actions: CodeActions{
				SourceFormatAllTerraform: true,
			},
			only:     []lsp.CodeActionKind{},
			expected: CodeActions{},
		},
		{
			name: "filter to matching action",
			actions: CodeActions{
				SourceFormatAllTerraform:  true,
				lsp.SourceOrganizeImports: true,
			},
			only: []lsp.CodeActionKind{SourceFormatAllTerraform},
			expected: CodeActions{
				SourceFormatAllTerraform: true,
			},
		},
		{
			name: "filter to non-existent action",
			actions: CodeActions{
				SourceFormatAllTerraform: true,
			},
			only:     []lsp.CodeActionKind{"source.nonexistent"},
			expected: CodeActions{},
		},
		{
			name: "filter to multiple matching actions",
			actions: CodeActions{
				SourceFormatAllTerraform:  true,
				lsp.SourceOrganizeImports: true,
				lsp.RefactorExtract:       true,
			},
			only: []lsp.CodeActionKind{
				SourceFormatAllTerraform,
				lsp.SourceOrganizeImports,
			},
			expected: CodeActions{
				SourceFormatAllTerraform:  true,
				lsp.SourceOrganizeImports: true,
			},
		},
		{
			name: "filter with some matching and some non-matching",
			actions: CodeActions{
				SourceFormatAllTerraform: true,
			},
			only: []lsp.CodeActionKind{
				SourceFormatAllTerraform,
				"source.nonexistent",
			},
			expected: CodeActions{
				SourceFormatAllTerraform: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.actions.Only(tt.only)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportedCodeActions(t *testing.T) {
	assert.True(t, SupportedCodeActions[SourceFormatAllTerraform])
	assert.Len(t, SupportedCodeActions, 1)
}
