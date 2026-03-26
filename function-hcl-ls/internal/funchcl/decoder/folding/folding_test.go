package folding

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var innerBraceBehavior = decoder.LangServerBehavior{InnerBraceRangesForFolding: true}

func TestCollect(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Range
	}{
		{
			name:     "single line block not folded",
			input:    "resource foo { body = {} }\n",
			expected: nil,
		},
		{
			name: "multiline block folded",
			input: `resource foo {
  body = {}
}
`,
			expected: []Range{
				{
					StartLine:   1,
					StartColumn: 15,
					EndLine:     3,
					EndColumn:   1,
					Kind:        "region",
				},
			},
		},
		{
			name: "nested blocks",
			input: `resource foo {
  locals {
    x = 1
  }
  body = {}
}
`,
			expected: []Range{
				{
					StartLine:   1,
					StartColumn: 15,
					EndLine:     6,
					EndColumn:   1,
					Kind:        "region",
				},
				{
					StartLine:   2,
					StartColumn: 11,
					EndLine:     4,
					EndColumn:   3,
					Kind:        "region",
				},
			},
		},
		{
			name: "multiline object expression",
			input: `locals {
  obj = {
    foo = "bar"
  }
}
`,
			expected: []Range{
				{
					StartLine:   1,
					StartColumn: 9,
					EndLine:     5,
					EndColumn:   1,
					Kind:        "region",
				},
				{
					StartLine:   2,
					StartColumn: 10,
					EndLine:     4,
					EndColumn:   3,
					Kind:        "region",
				},
			},
		},
		{
			name: "block with multiple labels",
			input: `composite status {
  body = {}
}
`,
			expected: []Range{
				{
					StartLine:   1,
					StartColumn: 19,
					EndLine:     3,
					EndColumn:   1,
					Kind:        "region",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.input), "test.hcl", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "parse error: %s", diags.Error())

			got := Collect(file, innerBraceBehavior)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCollect_NilFile(t *testing.T) {
	got := Collect(nil, innerBraceBehavior)
	assert.Nil(t, got)
}
