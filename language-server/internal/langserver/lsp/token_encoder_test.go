// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document/source"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang/semtok"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenEncoder(t *testing.T) {
	text := "hello world\n"
	lines := source.MakeSourceLines("test.hcl", []byte(text))
	tokens := []semtok.SemanticToken{
		{
			Type:      semtok.TokenTypeKeyword,
			Modifiers: semtok.TokenModifiers{},
			Range: hcl.Range{
				Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:   hcl.Pos{Line: 1, Column: 6, Byte: 5},
			},
		},
	}

	encoder := NewTokenEncoder(tokens, lines)

	require.NotNil(t, encoder)
	assert.Equal(t, tokens, encoder.tokens)
	assert.Equal(t, lines, encoder.lines)
	assert.NotNil(t, encoder.types)
	assert.NotNil(t, encoder.mods)
	assert.Equal(t, 0, encoder.lastEncodedTokenIdx)
}

func TestTokenEncoder_Encode_SingleLineToken(t *testing.T) {
	text := "hello world\n"
	lines := source.MakeSourceLines("test.hcl", []byte(text))
	tokens := []semtok.SemanticToken{
		{
			Type:      semtok.TokenTypeKeyword,
			Modifiers: semtok.TokenModifiers{},
			Range: hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 6, Byte: 5},
				Filename: "test.hcl",
			},
		},
	}

	encoder := NewTokenEncoder(tokens, lines)
	data := encoder.Encode()

	// Should have 5 values per token: deltaLine, deltaStartChar, length, tokenTypeIdx, modifierBitMask
	require.Len(t, data, 5)

	// First token should be at line 0 (delta from start), column 0, length 5
	assert.Equal(t, uint32(0), data[0]) // deltaLine
	assert.Equal(t, uint32(0), data[1]) // deltaStartChar
	assert.Equal(t, uint32(5), data[2]) // length
	// data[3] is tokenTypeIdx - depends on legend order
	assert.Equal(t, uint32(0), data[4]) // modifierBitMask (no modifiers)
}

func TestTokenEncoder_Encode_MultipleTokensSameLine(t *testing.T) {
	text := "foo bar baz\n"
	lines := source.MakeSourceLines("test.hcl", []byte(text))
	tokens := []semtok.SemanticToken{
		{
			Type:      semtok.TokenTypeKeyword,
			Modifiers: semtok.TokenModifiers{},
			Range: hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 4, Byte: 3},
				Filename: "test.hcl",
			},
		},
		{
			Type:      semtok.TokenTypeVariable,
			Modifiers: semtok.TokenModifiers{},
			Range: hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 5, Byte: 4},
				End:      hcl.Pos{Line: 1, Column: 8, Byte: 7},
				Filename: "test.hcl",
			},
		},
	}

	encoder := NewTokenEncoder(tokens, lines)
	data := encoder.Encode()

	// Should have 10 values (5 per token)
	require.Len(t, data, 10)

	// First token
	assert.Equal(t, uint32(0), data[0]) // deltaLine from start
	assert.Equal(t, uint32(0), data[1]) // deltaStartChar from start
	assert.Equal(t, uint32(3), data[2]) // length

	// Second token
	assert.Equal(t, uint32(0), data[5]) // deltaLine (same line)
	assert.Equal(t, uint32(4), data[6]) // deltaStartChar (4 chars from previous token start)
	assert.Equal(t, uint32(3), data[7]) // length
}

func TestTokenEncoder_Encode_MultipleLines(t *testing.T) {
	text := "foo\nbar\n"
	lines := source.MakeSourceLines("test.hcl", []byte(text))
	tokens := []semtok.SemanticToken{
		{
			Type:      semtok.TokenTypeKeyword,
			Modifiers: semtok.TokenModifiers{},
			Range: hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 4, Byte: 3},
				Filename: "test.hcl",
			},
		},
		{
			Type:      semtok.TokenTypeVariable,
			Modifiers: semtok.TokenModifiers{},
			Range: hcl.Range{
				Start:    hcl.Pos{Line: 2, Column: 1, Byte: 4},
				End:      hcl.Pos{Line: 2, Column: 4, Byte: 7},
				Filename: "test.hcl",
			},
		},
	}

	encoder := NewTokenEncoder(tokens, lines)
	data := encoder.Encode()

	// Should have 10 values (5 per token)
	require.Len(t, data, 10)

	// First token
	assert.Equal(t, uint32(0), data[0]) // deltaLine from start
	assert.Equal(t, uint32(0), data[1]) // deltaStartChar from start
	assert.Equal(t, uint32(3), data[2]) // length

	// Second token
	assert.Equal(t, uint32(1), data[5]) // deltaLine (1 line down)
	assert.Equal(t, uint32(0), data[6]) // deltaStartChar (from start of new line)
	assert.Equal(t, uint32(3), data[7]) // length
}

func TestTokenEncoder_Encode_WithModifiers(t *testing.T) {
	text := "hello\n"
	lines := source.MakeSourceLines("test.hcl", []byte(text))
	tokens := []semtok.SemanticToken{
		{
			Type:      semtok.TokenTypeKeyword,
			Modifiers: semtok.TokenModifiers{semtok.TokenModifierDeclaration},
			Range: hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 6, Byte: 5},
				Filename: "test.hcl",
			},
		},
	}

	encoder := NewTokenEncoder(tokens, lines)
	data := encoder.Encode()

	// Should have 5 values per token
	require.Len(t, data, 5)

	// Modifier bit mask should be non-zero (2^0 = 1 for the first modifier)
	assert.Equal(t, uint32(1), data[4]) // modifierBitMask
}

func TestTokenEncoder_Encode_MultilineToken(t *testing.T) {
	text := "hello\nworld\n"
	lines := source.MakeSourceLines("test.hcl", []byte(text))
	tokens := []semtok.SemanticToken{
		{
			Type:      semtok.TokenTypeString,
			Modifiers: semtok.TokenModifiers{},
			Range: hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 2, Column: 4, Byte: 9},
				Filename: "test.hcl",
			},
		},
	}

	encoder := NewTokenEncoder(tokens, lines)
	data := encoder.Encode()

	// Multiline tokens should be split into multiple entries (one per line)
	// Should have 10 values (5 per line)
	require.Len(t, data, 10)

	// First line of the multiline token
	assert.Equal(t, uint32(0), data[0]) // deltaLine from start
	assert.Equal(t, uint32(0), data[1]) // deltaStartChar from start
	// data[2] should be the length of the first line

	// Second line of the multiline token
	assert.Equal(t, uint32(1), data[5]) // deltaLine (1 line down)
	assert.Equal(t, uint32(0), data[6]) // deltaStartChar (from start of new line)
	// data[7] should be the length up to column 4
}

func TestTokenEncoder_Encode_EmptyTokens(t *testing.T) {
	text := "hello\n"
	lines := source.MakeSourceLines("test.hcl", []byte(text))
	tokens := []semtok.SemanticToken{}

	encoder := NewTokenEncoder(tokens, lines)
	data := encoder.Encode()

	// Should have no data for empty tokens
	require.Len(t, data, 0)
}

func TestComputeBitmask(t *testing.T) {
	tests := []struct {
		name     string
		mapping  map[semtok.TokenModifier]int
		values   semtok.TokenModifiers
		expected int
	}{
		{
			name:     "no modifiers",
			mapping:  map[semtok.TokenModifier]int{semtok.TokenModifierDeclaration: 0},
			values:   semtok.TokenModifiers{},
			expected: 0,
		},
		{
			name:     "single modifier at index 0",
			mapping:  map[semtok.TokenModifier]int{semtok.TokenModifierDeclaration: 0},
			values:   semtok.TokenModifiers{semtok.TokenModifierDeclaration},
			expected: 1, // 2^0 = 1
		},
		{
			name:     "single modifier at index 1",
			mapping:  map[semtok.TokenModifier]int{semtok.TokenModifierReadonly: 1},
			values:   semtok.TokenModifiers{semtok.TokenModifierReadonly},
			expected: 2, // 2^1 = 2
		},
		{
			name: "multiple modifiers",
			mapping: map[semtok.TokenModifier]int{
				semtok.TokenModifierDeclaration: 0,
				semtok.TokenModifierReadonly:    1,
				semtok.TokenModifierStatic:      2,
			},
			values:   semtok.TokenModifiers{semtok.TokenModifierDeclaration, semtok.TokenModifierStatic},
			expected: 5, // 2^0 + 2^2 = 1 + 4 = 5
		},
		{
			name: "all modifiers",
			mapping: map[semtok.TokenModifier]int{
				semtok.TokenModifierDeclaration: 0,
				semtok.TokenModifierReadonly:    1,
				semtok.TokenModifierStatic:      2,
			},
			values:   semtok.TokenModifiers{semtok.TokenModifierDeclaration, semtok.TokenModifierReadonly, semtok.TokenModifierStatic},
			expected: 7, // 2^0 + 2^1 + 2^2 = 1 + 2 + 4 = 7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeBitmask(tt.mapping, tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}
