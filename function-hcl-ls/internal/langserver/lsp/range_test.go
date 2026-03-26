// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestDocumentRangeToLSP(t *testing.T) {
	tests := []struct {
		name     string
		input    *document.Range
		expected lsp.Range
	}{
		{
			name:  "nil range",
			input: nil,
			expected: lsp.Range{
				Start: lsp.Position{Character: 0, Line: 0},
				End:   lsp.Position{Character: 0, Line: 0},
			},
		},
		{
			name: "valid range",
			input: &document.Range{
				Start: document.Pos{Line: 5, Column: 10},
				End:   document.Pos{Line: 5, Column: 20},
			},
			expected: lsp.Range{
				Start: lsp.Position{Character: 10, Line: 5},
				End:   lsp.Position{Character: 20, Line: 5},
			},
		},
		{
			name: "multiline range",
			input: &document.Range{
				Start: document.Pos{Line: 1, Column: 0},
				End:   document.Pos{Line: 3, Column: 15},
			},
			expected: lsp.Range{
				Start: lsp.Position{Character: 0, Line: 1},
				End:   lsp.Position{Character: 15, Line: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := documentRangeToLSP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLspRangeToDocRange(t *testing.T) {
	tests := []struct {
		name     string
		input    *lsp.Range
		expected *document.Range
	}{
		{
			name:     "nil range",
			input:    nil,
			expected: nil,
		},
		{
			name: "valid range",
			input: &lsp.Range{
				Start: lsp.Position{Character: 10, Line: 5},
				End:   lsp.Position{Character: 20, Line: 5},
			},
			expected: &document.Range{
				Start: document.Pos{Line: 5, Column: 10},
				End:   document.Pos{Line: 5, Column: 20},
			},
		},
		{
			name: "multiline range",
			input: &lsp.Range{
				Start: lsp.Position{Character: 0, Line: 1},
				End:   lsp.Position{Character: 15, Line: 3},
			},
			expected: &document.Range{
				Start: document.Pos{Line: 1, Column: 0},
				End:   document.Pos{Line: 3, Column: 15},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lspRangeToDocRange(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHCLRangeToLSP(t *testing.T) {
	tests := []struct {
		name     string
		input    hcl.Range
		expected lsp.Range
	}{
		{
			name: "simple range",
			input: hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
				Filename: "test.hcl",
			},
			expected: lsp.Range{
				Start: lsp.Position{Line: 0, Character: 0},
				End:   lsp.Position{Line: 0, Character: 4},
			},
		},
		{
			name: "multiline range",
			input: hcl.Range{
				Start:    hcl.Pos{Line: 2, Column: 3, Byte: 10},
				End:      hcl.Pos{Line: 4, Column: 6, Byte: 50},
				Filename: "test.hcl",
			},
			expected: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 2},
				End:   lsp.Position{Line: 3, Character: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HCLRangeToLSP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHCLPosToLSP(t *testing.T) {
	tests := []struct {
		name     string
		input    hcl.Pos
		expected lsp.Position
	}{
		{
			name:     "simple position",
			input:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			expected: lsp.Position{Line: 0, Character: 0},
		},
		{
			name:     "position with offset",
			input:    hcl.Pos{Line: 10, Column: 20, Byte: 100},
			expected: lsp.Position{Line: 9, Character: 19},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HCLPosToLSP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
