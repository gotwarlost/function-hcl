// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder/symbols"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestSupportedSymbolKind(t *testing.T) {
	tests := []struct {
		name       string
		supported  []lsp.SymbolKind
		kind       lsp.SymbolKind
		expected   lsp.SymbolKind
		expectedOk bool
	}{
		{
			name:       "empty supported list",
			supported:  []lsp.SymbolKind{},
			kind:       lsp.Class,
			expected:   lsp.SymbolKind(0),
			expectedOk: false,
		},
		{
			name:       "kind supported",
			supported:  []lsp.SymbolKind{lsp.Class, lsp.Function, lsp.Variable},
			kind:       lsp.Class,
			expected:   lsp.Class,
			expectedOk: true,
		},
		{
			name:       "kind not supported",
			supported:  []lsp.SymbolKind{lsp.Class, lsp.Function},
			kind:       lsp.Variable,
			expected:   lsp.SymbolKind(0),
			expectedOk: false,
		},
		{
			name:       "single supported kind matches",
			supported:  []lsp.SymbolKind{lsp.String},
			kind:       lsp.String,
			expected:   lsp.String,
			expectedOk: true,
		},
		{
			name:       "single supported kind does not match",
			supported:  []lsp.SymbolKind{lsp.String},
			kind:       lsp.Number,
			expected:   lsp.SymbolKind(0),
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := supportedSymbolKind(tt.supported, tt.kind)
			assert.Equal(t, tt.expectedOk, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprSymbolKind(t *testing.T) {
	tests := []struct {
		name       string
		symbolKind lang.SymbolExprKind
		supported  []lsp.SymbolKind
		expected   lsp.SymbolKind
		expectedOk bool
	}{
		{
			name: "bool literal type",
			symbolKind: lang.LiteralTypeKind{
				Type: cty.Bool,
			},
			supported:  []lsp.SymbolKind{lsp.Boolean, lsp.String, lsp.Number},
			expected:   lsp.Boolean,
			expectedOk: true,
		},
		{
			name: "string literal type",
			symbolKind: lang.LiteralTypeKind{
				Type: cty.String,
			},
			supported:  []lsp.SymbolKind{lsp.Boolean, lsp.String, lsp.Number},
			expected:   lsp.String,
			expectedOk: true,
		},
		{
			name: "number literal type",
			symbolKind: lang.LiteralTypeKind{
				Type: cty.Number,
			},
			supported:  []lsp.SymbolKind{lsp.Boolean, lsp.String, lsp.Number},
			expected:   lsp.Number,
			expectedOk: true,
		},
		{
			name:       "reference expression",
			symbolKind: lang.ReferenceExprKind{},
			supported:  []lsp.SymbolKind{lsp.Constant, lsp.Variable},
			expected:   lsp.Constant,
			expectedOk: true,
		},
		{
			name:       "tuple cons expression",
			symbolKind: lang.TupleConsExprKind{},
			supported:  []lsp.SymbolKind{lsp.Array, lsp.Variable},
			expected:   lsp.Array,
			expectedOk: true,
		},
		{
			name:       "object cons expression",
			symbolKind: lang.ObjectConsExprKind{},
			supported:  []lsp.SymbolKind{lsp.Struct, lsp.Variable},
			expected:   lsp.Struct,
			expectedOk: true,
		},
		{
			name: "bool literal type not supported",
			symbolKind: lang.LiteralTypeKind{
				Type: cty.Bool,
			},
			supported:  []lsp.SymbolKind{lsp.String, lsp.Number},
			expected:   lsp.SymbolKind(0),
			expectedOk: false,
		},
		{
			name:       "reference expression not supported when constant not in list",
			symbolKind: lang.ReferenceExprKind{},
			supported:  []lsp.SymbolKind{lsp.Variable},
			expected:   lsp.SymbolKind(0),
			expectedOk: false,
		},
		{
			name: "unknown expression kind falls back to variable",
			symbolKind: lang.LiteralTypeKind{
				Type: cty.DynamicPseudoType,
			},
			supported:  []lsp.SymbolKind{lsp.Variable},
			expected:   lsp.Variable,
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := exprSymbolKind(tt.symbolKind, tt.supported)
			assert.Equal(t, tt.expectedOk, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultSymbols(t *testing.T) {
	assert.NotEmpty(t, defaultSymbols)
	assert.Contains(t, defaultSymbols, lsp.File)
	assert.Contains(t, defaultSymbols, lsp.Class)
	assert.Contains(t, defaultSymbols, lsp.Function)
	assert.Contains(t, defaultSymbols, lsp.Variable)
}

func TestWorkspaceSymbols_Empty(t *testing.T) {
	result := WorkspaceSymbols([]symbols.Symbol{}, nil)
	assert.Empty(t, result)
}

func TestDocumentSymbols_Empty(t *testing.T) {
	result := DocumentSymbols([]symbols.Symbol{}, lsp.DocumentSymbolClientCapabilities{})
	assert.Empty(t, result)
}
