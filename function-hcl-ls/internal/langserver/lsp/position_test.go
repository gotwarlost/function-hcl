// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document/source"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHCLPositionFromLspPosition(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		lspPos   lsp.Position
		expected hcl.Pos
		wantErr  bool
	}{
		{
			name:   "simple position at start",
			text:   "hello world\n",
			lspPos: lsp.Position{Line: 0, Character: 0},
			expected: hcl.Pos{
				Line:   1,
				Column: 1,
				Byte:   0,
			},
			wantErr: false,
		},
		{
			name:   "position in middle of line",
			text:   "hello world\n",
			lspPos: lsp.Position{Line: 0, Character: 6},
			expected: hcl.Pos{
				Line:   1,
				Column: 7,
				Byte:   6,
			},
			wantErr: false,
		},
		{
			name:   "position on second line",
			text:   "hello world\nfoo bar\n",
			lspPos: lsp.Position{Line: 1, Character: 0},
			expected: hcl.Pos{
				Line:   2,
				Column: 1,
				Byte:   12,
			},
			wantErr: false,
		},
		{
			name:   "position in middle of second line",
			text:   "hello world\nfoo bar\n",
			lspPos: lsp.Position{Line: 1, Character: 4},
			expected: hcl.Pos{
				Line:   2,
				Column: 5,
				Byte:   16,
			},
			wantErr: false,
		},
		{
			name:    "invalid line number",
			text:    "hello world\n",
			lspPos:  lsp.Position{Line: 10, Character: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := source.MakeSourceLines("test.hcl", []byte(tt.text))
			doc := &document.Document{
				Lines: lines,
			}

			result, err := HCLPositionFromLspPosition(tt.lspPos, doc)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestLspPosToDocumentPos(t *testing.T) {
	tests := []struct {
		name     string
		input    lsp.Position
		expected document.Pos
	}{
		{
			name:     "zero position",
			input:    lsp.Position{Line: 0, Character: 0},
			expected: document.Pos{Line: 0, Column: 0},
		},
		{
			name:     "position with values",
			input:    lsp.Position{Line: 5, Character: 10},
			expected: document.Pos{Line: 5, Column: 10},
		},
		{
			name:     "large position values",
			input:    lsp.Position{Line: 100, Character: 200},
			expected: document.Pos{Line: 100, Column: 200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lspPosToDocumentPos(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
