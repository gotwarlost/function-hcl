// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestToLocationLink(t *testing.T) {
	tests := []struct {
		name     string
		path     lang.Path
		rng      hcl.Range
		expected lsp.LocationLink
	}{
		{
			name: "simple location link",
			path: lang.Path{
				Path:       "/test/path",
				LanguageID: "hcl",
			},
			rng: hcl.Range{
				Filename: "test.hcl",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
			},
			expected: lsp.LocationLink{
				OriginSelectionRange: &lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 4},
				},
				TargetURI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "test.hcl"))),
				TargetRange: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 4},
				},
				TargetSelectionRange: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 4},
				},
			},
		},
		{
			name: "multiline location link",
			path: lang.Path{
				Path:       "/another/path",
				LanguageID: "hcl",
			},
			rng: hcl.Range{
				Filename: "config.hcl",
				Start:    hcl.Pos{Line: 2, Column: 3, Byte: 10},
				End:      hcl.Pos{Line: 4, Column: 6, Byte: 50},
			},
			expected: lsp.LocationLink{
				OriginSelectionRange: &lsp.Range{
					Start: lsp.Position{Line: 1, Character: 2},
					End:   lsp.Position{Line: 3, Character: 5},
				},
				TargetURI: lsp.DocumentURI(getFileURI(filepath.Join("/another/path", "config.hcl"))),
				TargetRange: lsp.Range{
					Start: lsp.Position{Line: 1, Character: 2},
					End:   lsp.Position{Line: 3, Character: 5},
				},
				TargetSelectionRange: lsp.Range{
					Start: lsp.Position{Line: 1, Character: 2},
					End:   lsp.Position{Line: 3, Character: 5},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLocationLink(tt.path, tt.rng)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLocationLinks(t *testing.T) {
	path := lang.Path{
		Path:       "/test/path",
		LanguageID: "hcl",
	}

	tests := []struct {
		name     string
		ranges   []hcl.Range
		expected []lsp.LocationLink
	}{
		{
			name: "single range",
			ranges: []hcl.Range{
				{
					Filename: "test.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
				},
			},
			expected: []lsp.LocationLink{
				{
					OriginSelectionRange: &lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
					TargetURI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "test.hcl"))),
					TargetRange: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
					TargetSelectionRange: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
				},
			},
		},
		{
			name: "multiple ranges",
			ranges: []hcl.Range{
				{
					Filename: "first.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
				},
				{
					Filename: "second.hcl",
					Start:    hcl.Pos{Line: 2, Column: 2, Byte: 10},
					End:      hcl.Pos{Line: 2, Column: 8, Byte: 16},
				},
			},
			expected: []lsp.LocationLink{
				{
					OriginSelectionRange: &lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
					TargetURI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "first.hcl"))),
					TargetRange: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
					TargetSelectionRange: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
				},
				{
					OriginSelectionRange: &lsp.Range{
						Start: lsp.Position{Line: 1, Character: 1},
						End:   lsp.Position{Line: 1, Character: 7},
					},
					TargetURI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "second.hcl"))),
					TargetRange: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 1},
						End:   lsp.Position{Line: 1, Character: 7},
					},
					TargetSelectionRange: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 1},
						End:   lsp.Position{Line: 1, Character: 7},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLocationLinks(path, tt.ranges)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLocation(t *testing.T) {
	tests := []struct {
		name     string
		path     lang.Path
		rng      hcl.Range
		expected lsp.Location
	}{
		{
			name: "simple location",
			path: lang.Path{
				Path:       "/test/path",
				LanguageID: "hcl",
			},
			rng: hcl.Range{
				Filename: "test.hcl",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
			},
			expected: lsp.Location{
				URI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "test.hcl"))),
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 4},
				},
			},
		},
		{
			name: "multiline location",
			path: lang.Path{
				Path:       "/another/path",
				LanguageID: "hcl",
			},
			rng: hcl.Range{
				Filename: "config.hcl",
				Start:    hcl.Pos{Line: 2, Column: 3, Byte: 10},
				End:      hcl.Pos{Line: 4, Column: 6, Byte: 50},
			},
			expected: lsp.Location{
				URI: lsp.DocumentURI(getFileURI(filepath.Join("/another/path", "config.hcl"))),
				Range: lsp.Range{
					Start: lsp.Position{Line: 1, Character: 2},
					End:   lsp.Position{Line: 3, Character: 5},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLocation(tt.path, tt.rng)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLocations(t *testing.T) {
	path := lang.Path{
		Path:       "/test/path",
		LanguageID: "hcl",
	}

	tests := []struct {
		name     string
		ranges   []hcl.Range
		expected []lsp.Location
	}{
		{
			name: "single range",
			ranges: []hcl.Range{
				{
					Filename: "test.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
				},
			},
			expected: []lsp.Location{
				{
					URI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "test.hcl"))),
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
				},
			},
		},
		{
			name: "multiple ranges",
			ranges: []hcl.Range{
				{
					Filename: "first.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 5, Byte: 4},
				},
				{
					Filename: "second.hcl",
					Start:    hcl.Pos{Line: 2, Column: 2, Byte: 10},
					End:      hcl.Pos{Line: 2, Column: 8, Byte: 16},
				},
			},
			expected: []lsp.Location{
				{
					URI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "first.hcl"))),
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 4},
					},
				},
				{
					URI: lsp.DocumentURI(getFileURI(filepath.Join("/test/path", "second.hcl"))),
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 1},
						End:   lsp.Position{Line: 1, Character: 7},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLocations(path, tt.ranges)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// getFileURI returns a file URI for the given path, handling platform-specific differences
func getFileURI(path string) string {
	if runtime.GOOS == "windows" {
		// On Windows, paths should be like file:///C:/path/to/file
		return "file:///" + filepath.ToSlash(path)
	}
	// On Unix-like systems, paths should be like file:///path/to/file
	return "file://" + path
}
