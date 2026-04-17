// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package source

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// created by Claude and looks reasonable

func TestMakeSourceLines_Empty(t *testing.T) {
	lines := MakeSourceLines("test.hcl", []byte{})

	require.Len(t, lines, 1, "empty source should produce one virtual line")
	assert.Empty(t, lines[0].Bytes)
	assert.Equal(t, "test.hcl", lines[0].Range.Filename)
	assert.Equal(t, hcl.InitialPos, lines[0].Range.Start)
	assert.Equal(t, hcl.InitialPos, lines[0].Range.End)
}

func TestMakeSourceLines_SingleLineNoNewline(t *testing.T) {
	source := []byte("hello world")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 2, "should have content line plus virtual line")

	// First line
	assert.Equal(t, []byte("hello world"), lines[0].Bytes)
	assert.Equal(t, "test.hcl", lines[0].Range.Filename)
	assert.Equal(t, 1, lines[0].Range.Start.Line)
	assert.Equal(t, 1, lines[0].Range.Start.Column)

	// Virtual line
	assert.Empty(t, lines[1].Bytes)
	assert.Equal(t, lines[0].Range.End, lines[1].Range.Start)
	assert.Equal(t, lines[0].Range.End, lines[1].Range.End)
}

func TestMakeSourceLines_SingleLineWithNewline(t *testing.T) {
	source := []byte("hello world\n")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 2, "should have content line plus virtual line")

	// First line (includes newline)
	assert.Equal(t, []byte("hello world\n"), lines[0].Bytes)
	assert.Equal(t, "test.hcl", lines[0].Range.Filename)
	assert.Equal(t, 1, lines[0].Range.Start.Line)
	assert.Equal(t, 1, lines[0].Range.Start.Column)
	assert.Equal(t, 2, lines[0].Range.End.Line)
	assert.Equal(t, 1, lines[0].Range.End.Column)

	// Virtual line
	assert.Empty(t, lines[1].Bytes)
	assert.Equal(t, 2, lines[1].Range.Start.Line)
}

func TestMakeSourceLines_MultipleLinesWithNewlines(t *testing.T) {
	source := []byte("line 1\nline 2\nline 3\n")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 4, "should have 3 content lines plus virtual line")

	// First line
	assert.Equal(t, []byte("line 1\n"), lines[0].Bytes)
	assert.Equal(t, 1, lines[0].Range.Start.Line)
	assert.Equal(t, 2, lines[0].Range.End.Line)

	// Second line
	assert.Equal(t, []byte("line 2\n"), lines[1].Bytes)
	assert.Equal(t, 2, lines[1].Range.Start.Line)
	assert.Equal(t, 3, lines[1].Range.End.Line)

	// Third line
	assert.Equal(t, []byte("line 3\n"), lines[2].Bytes)
	assert.Equal(t, 3, lines[2].Range.Start.Line)
	assert.Equal(t, 4, lines[2].Range.End.Line)

	// Virtual line
	assert.Empty(t, lines[3].Bytes)
	assert.Equal(t, 4, lines[3].Range.Start.Line)
}

func TestMakeSourceLines_MultipleLinesLastNoNewline(t *testing.T) {
	source := []byte("line 1\nline 2\nline 3")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 4, "should have 3 content lines plus virtual line")

	// First line
	assert.Equal(t, []byte("line 1\n"), lines[0].Bytes)

	// Second line
	assert.Equal(t, []byte("line 2\n"), lines[1].Bytes)

	// Third line (no newline)
	assert.Equal(t, []byte("line 3"), lines[2].Bytes)
	assert.Equal(t, 3, lines[2].Range.Start.Line)

	// Virtual line
	assert.Empty(t, lines[3].Bytes)
}

func TestMakeSourceLines_EmptyLines(t *testing.T) {
	source := []byte("\n\n\n")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 4, "should have 3 newline lines plus virtual line")

	assert.Equal(t, []byte("\n"), lines[0].Bytes)
	assert.Equal(t, []byte("\n"), lines[1].Bytes)
	assert.Equal(t, []byte("\n"), lines[2].Bytes)
	assert.Empty(t, lines[3].Bytes)
}

func TestMakeSourceLines_MixedContent(t *testing.T) {
	source := []byte("resource \"test\" {\n  name = \"value\"\n}\n")
	lines := MakeSourceLines("main.tf", source)

	require.Len(t, lines, 4)

	assert.Equal(t, []byte("resource \"test\" {\n"), lines[0].Bytes)
	assert.Equal(t, []byte("  name = \"value\"\n"), lines[1].Bytes)
	assert.Equal(t, []byte("}\n"), lines[2].Bytes)
	assert.Empty(t, lines[3].Bytes)

	// Check filenames are preserved
	for i := range lines {
		assert.Equal(t, "main.tf", lines[i].Range.Filename)
	}
}

func TestMakeSourceLines_PreservesRangeInfo(t *testing.T) {
	source := []byte("abc\ndef\n")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 3)

	// Line 1: "abc\n"
	assert.Equal(t, 1, lines[0].Range.Start.Line)
	assert.Equal(t, 1, lines[0].Range.Start.Column)
	assert.Equal(t, 2, lines[0].Range.End.Line)
	assert.Equal(t, 1, lines[0].Range.End.Column)
	assert.Equal(t, 0, lines[0].Range.Start.Byte)
	assert.Equal(t, 4, lines[0].Range.End.Byte)

	// Line 2: "def\n"
	assert.Equal(t, 2, lines[1].Range.Start.Line)
	assert.Equal(t, 1, lines[1].Range.Start.Column)
	assert.Equal(t, 3, lines[1].Range.End.Line)
	assert.Equal(t, 1, lines[1].Range.End.Column)
	assert.Equal(t, 4, lines[1].Range.Start.Byte)
	assert.Equal(t, 8, lines[1].Range.End.Byte)
}

func TestScanLines_Empty(t *testing.T) {
	advance, token, err := scanLines([]byte{}, true)

	assert.NoError(t, err)
	assert.Equal(t, 0, advance)
	assert.Nil(t, token)
}

func TestScanLines_SingleLineWithNewline(t *testing.T) {
	data := []byte("hello\n")
	advance, token, err := scanLines(data, false)

	assert.NoError(t, err)
	assert.Equal(t, 6, advance)
	assert.Equal(t, []byte("hello\n"), token)
}

func TestScanLines_SingleLineNoNewline_NotEOF(t *testing.T) {
	data := []byte("hello")
	advance, token, err := scanLines(data, false)

	assert.NoError(t, err)
	assert.Equal(t, 0, advance, "should request more data")
	assert.Nil(t, token)
}

func TestScanLines_SingleLineNoNewline_EOF(t *testing.T) {
	data := []byte("hello")
	advance, token, err := scanLines(data, true)

	assert.NoError(t, err)
	assert.Equal(t, 5, advance)
	assert.Equal(t, []byte("hello"), token)
}

func TestScanLines_MultipleNewlines(t *testing.T) {
	data := []byte("line1\nline2\n")

	// First scan
	advance, token, err := scanLines(data, false)
	assert.NoError(t, err)
	assert.Equal(t, 6, advance)
	assert.Equal(t, []byte("line1\n"), token)

	// Second scan
	advance, token, err = scanLines(data[6:], false)
	assert.NoError(t, err)
	assert.Equal(t, 6, advance)
	assert.Equal(t, []byte("line2\n"), token)
}

func TestScanLines_JustNewline(t *testing.T) {
	data := []byte("\n")
	advance, token, err := scanLines(data, false)

	assert.NoError(t, err)
	assert.Equal(t, 1, advance)
	assert.Equal(t, []byte("\n"), token)
}

func TestStringLines_Empty(t *testing.T) {
	var lines []Line
	result := StringLines(lines)
	assert.Empty(t, result)
}

func TestStringLines_SingleLine(t *testing.T) {
	lines := []Line{
		{
			Bytes: []byte("hello world"),
			Range: hcl.Range{},
		},
	}
	result := StringLines(lines)

	require.Len(t, result, 1)
	assert.Equal(t, "hello world", result[0])
}

func TestStringLines_MultipleLines(t *testing.T) {
	lines := []Line{
		{Bytes: []byte("line 1\n"), Range: hcl.Range{}},
		{Bytes: []byte("line 2\n"), Range: hcl.Range{}},
		{Bytes: []byte("line 3"), Range: hcl.Range{}},
	}
	result := StringLines(lines)

	require.Len(t, result, 3)
	assert.Equal(t, "line 1\n", result[0])
	assert.Equal(t, "line 2\n", result[1])
	assert.Equal(t, "line 3", result[2])
}

func TestStringLines_EmptyBytes(t *testing.T) {
	lines := []Line{
		{Bytes: []byte("content"), Range: hcl.Range{}},
		{Bytes: []byte{}, Range: hcl.Range{}},
		{Bytes: []byte("more"), Range: hcl.Range{}},
	}
	result := StringLines(lines)

	require.Len(t, result, 3)
	assert.Equal(t, "content", result[0])
	assert.Equal(t, "", result[1])
	assert.Equal(t, "more", result[2])
}

func TestStringLines_PreservesNewlines(t *testing.T) {
	source := []byte("a\nb\nc\n")
	lines := MakeSourceLines("test.hcl", source)
	result := StringLines(lines)

	require.Len(t, result, 4)
	assert.Equal(t, "a\n", result[0])
	assert.Equal(t, "b\n", result[1])
	assert.Equal(t, "c\n", result[2])
	assert.Equal(t, "", result[3]) // virtual line
}

func TestMakeSourceLines_UnicodeContent(t *testing.T) {
	source := []byte("hello 世界\n你好\n")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 3)
	assert.Equal(t, []byte("hello 世界\n"), lines[0].Bytes)
	assert.Equal(t, []byte("你好\n"), lines[1].Bytes)
	assert.Empty(t, lines[2].Bytes)
}

func TestMakeSourceLines_SpecialCharacters(t *testing.T) {
	source := []byte("tab\there\nnull\x00byte\nend")
	lines := MakeSourceLines("test.hcl", source)

	require.Len(t, lines, 4)
	assert.Equal(t, []byte("tab\there\n"), lines[0].Bytes)
	assert.Equal(t, []byte("null\x00byte\n"), lines[1].Bytes)
	assert.Equal(t, []byte("end"), lines[2].Bytes)
}
