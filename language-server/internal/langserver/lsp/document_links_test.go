// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestLinks(t *testing.T) {
	tests := []struct {
		name     string
		links    []lang.Link
		caps     *lsp.DocumentLinkClientCapabilities
		expected []lsp.DocumentLink
	}{
		{
			name:     "empty links",
			links:    []lang.Link{},
			caps:     nil,
			expected: []lsp.DocumentLink{},
		},
		{
			name: "single link without tooltip support",
			links: []lang.Link{
				{
					URI:     "https://example.com",
					Tooltip: "Example Link",
					Range: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
					},
				},
			},
			caps: nil,
			expected: []lsp.DocumentLink{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 9},
					},
					Target:  "https://example.com",
					Tooltip: "",
				},
			},
		},
		{
			name: "single link with tooltip support disabled",
			links: []lang.Link{
				{
					URI:     "https://example.com",
					Tooltip: "Example Link",
					Range: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
					},
				},
			},
			caps: &lsp.DocumentLinkClientCapabilities{
				TooltipSupport: false,
			},
			expected: []lsp.DocumentLink{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 9},
					},
					Target:  "https://example.com",
					Tooltip: "",
				},
			},
		},
		{
			name: "single link with tooltip support enabled",
			links: []lang.Link{
				{
					URI:     "https://example.com",
					Tooltip: "Example Link",
					Range: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
					},
				},
			},
			caps: &lsp.DocumentLinkClientCapabilities{
				TooltipSupport: true,
			},
			expected: []lsp.DocumentLink{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 9},
					},
					Target:  "https://example.com",
					Tooltip: "Example Link",
				},
			},
		},
		{
			name: "multiple links with tooltip support",
			links: []lang.Link{
				{
					URI:     "https://example.com",
					Tooltip: "First Link",
					Range: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
					},
				},
				{
					URI:     "https://another.com",
					Tooltip: "Second Link",
					Range: hcl.Range{
						Start: hcl.Pos{Line: 2, Column: 5, Byte: 20},
						End:   hcl.Pos{Line: 2, Column: 15, Byte: 30},
					},
				},
			},
			caps: &lsp.DocumentLinkClientCapabilities{
				TooltipSupport: true,
			},
			expected: []lsp.DocumentLink{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 9},
					},
					Target:  "https://example.com",
					Tooltip: "First Link",
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 4},
						End:   lsp.Position{Line: 1, Character: 14},
					},
					Target:  "https://another.com",
					Tooltip: "Second Link",
				},
			},
		},
		{
			name: "link with empty tooltip",
			links: []lang.Link{
				{
					URI:     "https://example.com",
					Tooltip: "",
					Range: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
					},
				},
			},
			caps: &lsp.DocumentLinkClientCapabilities{
				TooltipSupport: true,
			},
			expected: []lsp.DocumentLink{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 9},
					},
					Target:  "https://example.com",
					Tooltip: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Links(tt.links, tt.caps)
			assert.Equal(t, tt.expected, result)
		})
	}
}
