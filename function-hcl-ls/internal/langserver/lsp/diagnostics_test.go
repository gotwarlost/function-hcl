// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHCLSeverityToLSP(t *testing.T) {
	tests := []struct {
		name     string
		severity hcl.DiagnosticSeverity
		expected lsp.DiagnosticSeverity
	}{
		{
			name:     "error severity",
			severity: hcl.DiagError,
			expected: lsp.SeverityError,
		},
		{
			name:     "warning severity",
			severity: hcl.DiagWarning,
			expected: lsp.SeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HCLSeverityToLSP(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHCLSeverityToLSP_InvalidPanics(t *testing.T) {
	assert.Panics(t, func() {
		HCLSeverityToLSP(hcl.DiagInvalid)
	}, "should panic on invalid severity")
}

func TestHCLDiagsToLSP_NeverReturnsNil(t *testing.T) {
	diags := HCLDiagsToLSP(nil, "test")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{}, "test")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
		},
	}, "source")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}
}

func TestHCLDiagsToLSP(t *testing.T) {
	tests := []struct {
		name     string
		hclDiags hcl.Diagnostics
		source   string
		expected []lsp.Diagnostic
	}{
		{
			name:     "empty diagnostics",
			hclDiags: hcl.Diagnostics{},
			source:   "test",
			expected: []lsp.Diagnostic{},
		},
		{
			name: "single error with summary only",
			hclDiags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Test error",
				},
			},
			source: "test-source",
			expected: []lsp.Diagnostic{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 0},
					},
					Severity: lsp.SeverityError,
					Source:   "test-source",
					Message:  "Test error",
				},
			},
		},
		{
			name: "single warning with summary and detail",
			hclDiags: hcl.Diagnostics{
				{
					Severity: hcl.DiagWarning,
					Summary:  "Test warning",
					Detail:   "Additional details",
				},
			},
			source: "test-source",
			expected: []lsp.Diagnostic{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 0},
					},
					Severity: lsp.SeverityWarning,
					Source:   "test-source",
					Message:  "Test warning: Additional details",
				},
			},
		},
		{
			name: "diagnostic with subject range",
			hclDiags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Parse error",
					Subject: &hcl.Range{
						Start:    hcl.Pos{Line: 2, Column: 5, Byte: 20},
						End:      hcl.Pos{Line: 2, Column: 10, Byte: 25},
						Filename: "test.hcl",
					},
				},
			},
			source: "parser",
			expected: []lsp.Diagnostic{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 4},
						End:   lsp.Position{Line: 1, Character: 9},
					},
					Severity: lsp.SeverityError,
					Source:   "parser",
					Message:  "Parse error",
				},
			},
		},
		{
			name: "multiple diagnostics",
			hclDiags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Error 1",
					Detail:   "Details 1",
				},
				{
					Severity: hcl.DiagWarning,
					Summary:  "Warning 1",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "Error 2",
					Subject: &hcl.Range{
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
						Filename: "test.hcl",
					},
				},
			},
			source: "validator",
			expected: []lsp.Diagnostic{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 0},
					},
					Severity: lsp.SeverityError,
					Source:   "validator",
					Message:  "Error 1: Details 1",
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 0},
					},
					Severity: lsp.SeverityWarning,
					Source:   "validator",
					Message:  "Warning 1",
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
					Severity: lsp.SeverityError,
					Source:   "validator",
					Message:  "Error 2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HCLDiagsToLSP(tt.hclDiags, tt.source)
			require.Len(t, result, len(tt.expected))
			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].Severity, result[i].Severity)
				assert.Equal(t, tt.expected[i].Source, result[i].Source)
				assert.Equal(t, tt.expected[i].Message, result[i].Message)
				assert.Equal(t, tt.expected[i].Range, result[i].Range)
			}
		})
	}
}
