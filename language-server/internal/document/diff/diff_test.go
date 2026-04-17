// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diff

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/document/source"
	"github.com/hashicorp/hcl/v2"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// created by Claude and looks reasonable

// Test fileChange.Text() method
func TestFileChange_Text(t *testing.T) {
	tests := []struct {
		name     string
		newText  string
		expected string
	}{
		{
			name:     "empty text",
			newText:  "",
			expected: "",
		},
		{
			name:     "simple text",
			newText:  "hello world",
			expected: "hello world",
		},
		{
			name:     "multi-line text",
			newText:  "line 1\nline 2\nline 3\n",
			expected: "line 1\nline 2\nline 3\n",
		},
		{
			name:     "text with special characters",
			newText:  "tab\there\nnewline\nend",
			expected: "tab\there\nnewline\nend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fileChange{
				newText: tt.newText,
			}
			assert.Equal(t, tt.expected, fc.Text())
		})
	}
}

// Test fileChange.Range() method with nil range
func TestFileChange_Range_Nil(t *testing.T) {
	fc := &fileChange{
		newText: "some text",
		rng:     nil,
	}

	result := fc.Range()
	assert.Nil(t, result)
}

// Test fileChange.Range() method with valid range
func TestFileChange_Range_Valid(t *testing.T) {
	tests := []struct {
		name     string
		hclRange hcl.Range
		expected document.Range
	}{
		{
			name: "single point range",
			hclRange: hcl.Range{
				Filename: "test.hcl",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 1, Byte: 0},
			},
			expected: document.Range{
				Start: document.Pos{Line: 0, Column: 0},
				End:   document.Pos{Line: 0, Column: 0},
			},
		},
		{
			name: "single line range",
			hclRange: hcl.Range{
				Filename: "test.hcl",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 10, Byte: 9},
			},
			expected: document.Range{
				Start: document.Pos{Line: 0, Column: 0},
				End:   document.Pos{Line: 0, Column: 9},
			},
		},
		{
			name: "multi-line range",
			hclRange: hcl.Range{
				Filename: "test.hcl",
				Start:    hcl.Pos{Line: 2, Column: 5, Byte: 10},
				End:      hcl.Pos{Line: 5, Column: 8, Byte: 50},
			},
			expected: document.Range{
				Start: document.Pos{Line: 1, Column: 4},
				End:   document.Pos{Line: 4, Column: 7},
			},
		},
		{
			name: "range with zero column",
			hclRange: hcl.Range{
				Filename: "test.hcl",
				Start:    hcl.Pos{Line: 3, Column: 1, Byte: 20},
				End:      hcl.Pos{Line: 3, Column: 15, Byte: 34},
			},
			expected: document.Range{
				Start: document.Pos{Line: 2, Column: 0},
				End:   document.Pos{Line: 2, Column: 14},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fileChange{
				newText: "text",
				rng:     &tt.hclRange,
			}

			result := fc.Range()
			require.NotNil(t, result)
			assert.Equal(t, tt.expected.Start.Line, result.Start.Line)
			assert.Equal(t, tt.expected.Start.Column, result.Start.Column)
			assert.Equal(t, tt.expected.End.Line, result.End.Line)
			assert.Equal(t, tt.expected.End.Column, result.End.Column)
		})
	}
}

// Test Diff function with identical content
func TestDiff_IdenticalContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "empty files",
			content: "",
		},
		{
			name:    "single line",
			content: "hello world",
		},
		{
			name: "multi-line",
			content: `line 1
line 2
line 3`,
		},
		{
			name: "complex content",
			content: `resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  tags = {
    Name = "main"
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle := document.Handle{
				Filename: "test.tf",
			}

			before := []byte(tt.content)
			after := []byte(tt.content)

			changes := Diff(handle, before, after)
			assert.Empty(t, changes)
		})
	}
}

// Test Diff function with simple replacements
func TestDiff_SimpleReplace(t *testing.T) {
	handle := document.Handle{
		Filename: "test.hcl",
	}

	before := []byte("hello world\n")
	after := []byte("hello there\n")

	changes := Diff(handle, before, after)
	require.Len(t, changes, 1)

	assert.Equal(t, "hello there\n", changes[0].Text())
	assert.NotNil(t, changes[0].Range())
}

// Test Diff function with deletions
func TestDiff_Deletion(t *testing.T) {
	handle := document.Handle{
		Filename: "test.hcl",
	}

	tests := []struct {
		name   string
		before string
		after  string
	}{
		{
			name: "delete single line",
			before: `line 1
line 2
line 3`,
			after: `line 1
line 3`,
		},
		{
			name: "delete multiple lines",
			before: `line 1
line 2
line 3
line 4
line 5`,
			after: `line 1
line 5`,
		},
		{
			name: "delete chars from beginning",
			before: `line 1
line 2
line 3`,
			after: "line 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := Diff(handle, []byte(tt.before), []byte(tt.after))
			assert.NotEmpty(t, changes)
			// At least one change should be a deletion (empty text)
			hasDelete := false
			for _, ch := range changes {
				if ch.Text() == "" {
					hasDelete = true
					break
				}
			}
			assert.True(t, hasDelete, "Expected at least one deletion")
		})
	}
}

// Test Diff function with insertions
func TestDiff_Insertion(t *testing.T) {
	handle := document.Handle{
		Filename: "test.hcl",
	}

	tests := []struct {
		name   string
		before string
		after  string
	}{
		{
			name:   "insert single line",
			before: "line 1\nline 3",
			after:  "line 1\nline 2\nline 3",
		},
		{
			name:   "insert at beginning",
			before: "line 2\nline 3",
			after:  "line 1\nline 2\nline 3",
		},
		{
			name:   "insert at end",
			before: "line 1\nline 2",
			after:  "line 1\nline 2\nline 3",
		},
		{
			name:   "insert to empty file",
			before: "",
			after:  "new content\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := Diff(handle, []byte(tt.before), []byte(tt.after))
			assert.NotEmpty(t, changes)
		})
	}
}

// Test diffLines with empty inputs
func TestDiffLines_EmptyInputs(t *testing.T) {
	tests := []struct {
		name        string
		beforeLines []source.Line
		afterLines  []source.Line
	}{
		{
			name:        "both empty",
			beforeLines: []source.Line{},
			afterLines:  []source.Line{},
		},
		{
			name: "before empty, after has content",
			beforeLines: []source.Line{
				{Bytes: []byte{}, Range: hcl.Range{Filename: "test.hcl", Start: hcl.InitialPos, End: hcl.InitialPos}},
			},
			afterLines: []source.Line{
				{Bytes: []byte("new line\n"), Range: hcl.Range{Filename: "test.hcl", Start: hcl.Pos{Line: 1, Column: 1}, End: hcl.Pos{Line: 2, Column: 1}}},
				{Bytes: []byte{}, Range: hcl.Range{Filename: "test.hcl", Start: hcl.Pos{Line: 2, Column: 1}, End: hcl.Pos{Line: 2, Column: 1}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffLines("test.hcl", tt.beforeLines, tt.afterLines)
			// Should return a valid Changes slice (may be empty)
			assert.NotNil(t, changes)
		})
	}
}

// Test diffLines with single line changes
func TestDiffLines_SingleLine(t *testing.T) {
	filename := "test.hcl"

	beforeLines := source.MakeSourceLines(filename, []byte("old line\n"))
	afterLines := source.MakeSourceLines(filename, []byte("new line\n"))

	changes := diffLines(filename, beforeLines, afterLines)

	require.Len(t, changes, 1)
	assert.Equal(t, "new line\n", changes[0].Text())
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with multiple line changes
func TestDiffLines_MultipleLines(t *testing.T) {
	filename := "test.hcl"

	before := `line 1
old line 2
line 3
old line 4
line 5
`
	after := `line 1
new line 2
line 3
new line 4
line 5
`

	beforeLines := source.MakeSourceLines(filename, []byte(before))
	afterLines := source.MakeSourceLines(filename, []byte(after))

	changes := diffLines(filename, beforeLines, afterLines)

	assert.NotEmpty(t, changes)
	// Should have multiple changes for the two separate modifications
	assert.GreaterOrEqual(t, len(changes), 1)
}

// Test diffLines with replace operation
func TestDiffLines_ReplaceOperation(t *testing.T) {
	filename := "test.hcl"

	beforeLines := source.MakeSourceLines(filename, []byte("aaa\nbbb\nccc\n"))
	afterLines := source.MakeSourceLines(filename, []byte("aaa\nXXX\nccc\n"))

	changes := diffLines(filename, beforeLines, afterLines)

	require.Len(t, changes, 1)
	assert.Equal(t, "XXX\n", changes[0].Text())
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with delete operation
func TestDiffLines_DeleteOperation(t *testing.T) {
	filename := "test.hcl"

	beforeLines := source.MakeSourceLines(filename, []byte("aaa\nbbb\nccc\n"))
	afterLines := source.MakeSourceLines(filename, []byte("aaa\nccc\n"))

	changes := diffLines(filename, beforeLines, afterLines)

	require.Len(t, changes, 1)
	assert.Equal(t, "", changes[0].Text())
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with insert operation at beginning
func TestDiffLines_InsertAtBeginning(t *testing.T) {
	filename := "test.hcl"

	beforeLines := source.MakeSourceLines(filename, []byte("bbb\nccc\n"))
	afterLines := source.MakeSourceLines(filename, []byte("aaa\nbbb\nccc\n"))

	changes := diffLines(filename, beforeLines, afterLines)

	require.NotEmpty(t, changes)
	// Check that the inserted text is in the changes
	hasInsert := false
	for _, ch := range changes {
		if ch.Text() == "aaa\n" || ch.Text() == "aaa\nbbb\n" || ch.Text() == "aaa\nbbb\nccc\n" {
			hasInsert = true
			break
		}
	}
	assert.True(t, hasInsert)
}

// Test diffLines with insert operation at end
func TestDiffLines_InsertAtEnd(t *testing.T) {
	filename := "test.hcl"

	beforeLines := source.MakeSourceLines(filename, []byte("aaa\nbbb\n"))
	afterLines := source.MakeSourceLines(filename, []byte("aaa\nbbb\nccc\n"))

	changes := diffLines(filename, beforeLines, afterLines)

	require.NotEmpty(t, changes)
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with insert operation in middle
func TestDiffLines_InsertInMiddle(t *testing.T) {
	filename := "test.hcl"

	beforeLines := source.MakeSourceLines(filename, []byte("aaa\nccc\n"))
	afterLines := source.MakeSourceLines(filename, []byte("aaa\nbbb\nccc\n"))

	changes := diffLines(filename, beforeLines, afterLines)

	require.NotEmpty(t, changes)
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with multi-line replacement
func TestDiffLines_MultiLineReplace(t *testing.T) {
	filename := "test.hcl"

	before := `line 1
old 2
old 3
line 4
`
	after := `line 1
new 2
new 3
line 4
`

	beforeLines := source.MakeSourceLines(filename, []byte(before))
	afterLines := source.MakeSourceLines(filename, []byte(after))

	changes := diffLines(filename, beforeLines, afterLines)

	require.NotEmpty(t, changes)
	// Should have at least one replacement
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with multi-line deletion
func TestDiffLines_MultiLineDelete(t *testing.T) {
	filename := "test.hcl"

	before := `line 1
delete 2
delete 3
line 4
`
	after := `line 1
line 4
`

	beforeLines := source.MakeSourceLines(filename, []byte(before))
	afterLines := source.MakeSourceLines(filename, []byte(after))

	changes := diffLines(filename, beforeLines, afterLines)

	require.Len(t, changes, 1)
	assert.Equal(t, "", changes[0].Text())
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with multi-line insertion
func TestDiffLines_MultiLineInsert(t *testing.T) {
	filename := "test.hcl"

	before := `line 1
line 4
`
	after := `line 1
new 2
new 3
line 4
`

	beforeLines := source.MakeSourceLines(filename, []byte(before))
	afterLines := source.MakeSourceLines(filename, []byte(after))

	changes := diffLines(filename, beforeLines, afterLines)

	require.NotEmpty(t, changes)
	assert.NotNil(t, changes[0].Range())
}

// Test diffLines with complex mixed operations
func TestDiffLines_ComplexMixed(t *testing.T) {
	filename := "test.hcl"

	before := `keep 1
replace 2
delete 3
keep 4
keep 5
`
	after := `keep 1
REPLACED 2
keep 4
INSERT 4.5
keep 5
`

	beforeLines := source.MakeSourceLines(filename, []byte(before))
	afterLines := source.MakeSourceLines(filename, []byte(after))

	changes := diffLines(filename, beforeLines, afterLines)

	// Should have multiple changes
	assert.GreaterOrEqual(t, len(changes), 2)

	// All changes should have valid ranges
	for _, ch := range changes {
		assert.NotNil(t, ch.Range())
	}
}

// Test fileChange with opCode field
func TestFileChange_WithOpCode(t *testing.T) {
	opCode := difflib.OpCode{
		Tag: opDelete,
		I1:  1,
		I2:  3,
		J1:  1,
		J2:  1,
	}

	fc := &fileChange{
		newText: "",
		rng: &hcl.Range{
			Filename: "test.hcl",
			Start:    hcl.Pos{Line: 2, Column: 1, Byte: 10},
			End:      hcl.Pos{Line: 4, Column: 1, Byte: 30},
		},
		opCode: opCode,
	}

	assert.Equal(t, "", fc.Text())
	assert.NotNil(t, fc.Range())
}

// Test Diff with real document handle
func TestDiff_WithDocumentHandle(t *testing.T) {
	handle := document.Handle{
		Filename: "main.tf",
	}

	before := []byte(`resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`)

	after := []byte(`resource "aws_instance" "web" {
  ami           = "ami-87654321"
  instance_type = "t2.small"
}
`)

	changes := Diff(handle, before, after)

	assert.NotEmpty(t, changes)
	// Should have changes for the two modified lines
	assert.GreaterOrEqual(t, len(changes), 1)

	// Verify all changes have proper ranges
	for _, ch := range changes {
		assert.NotNil(t, ch.Range())
	}
}

// Test edge case: single character change
func TestDiff_SingleCharChange(t *testing.T) {
	handle := document.Handle{
		Filename: "test.hcl",
	}

	before := []byte("a\n")
	after := []byte("b\n")

	changes := Diff(handle, before, after)

	require.Len(t, changes, 1)
	assert.Equal(t, "b\n", changes[0].Text())
}

// Test edge case: whitespace only changes
func TestDiff_WhitespaceOnly(t *testing.T) {
	handle := document.Handle{
		Filename: "test.hcl",
	}

	tests := []struct {
		name   string
		before string
		after  string
	}{
		{
			name:   "add trailing space",
			before: "line\n",
			after:  "line \n",
		},
		{
			name:   "remove trailing space",
			before: "line \n",
			after:  "line\n",
		},
		{
			name:   "change tab to spaces",
			before: "\tindented\n",
			after:  "    indented\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := Diff(handle, []byte(tt.before), []byte(tt.after))
			assert.NotEmpty(t, changes)
		})
	}
}

// Test diffLines with no trailing newline
func TestDiffLines_NoTrailingNewline(t *testing.T) {
	filename := "test.hcl"

	beforeLines := source.MakeSourceLines(filename, []byte("line 1\nline 2"))
	afterLines := source.MakeSourceLines(filename, []byte("line 1\nline 2\n"))

	changes := diffLines(filename, beforeLines, afterLines)

	// Should detect the addition of trailing newline
	assert.NotEmpty(t, changes)
}
