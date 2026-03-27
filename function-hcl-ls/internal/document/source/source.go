// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package source provides facilities to represent text documents as a sequence
// of lines.
package source

import (
	"bytes"

	"github.com/hashicorp/hcl/v2"
)

// Line represents a line of source code. Each line is associated with an HCL range that
// includes the filename, start and end positions.
type Line struct {
	// Bytes returns the line byte inc. any trailing end-of-line markers
	Bytes []byte
	// Range returns range of the line bytes inc. any trailing end-of-line markers
	// The range will span across two lines in most cases
	// (other than last line without trailing new line)
	Range hcl.Range
}

// MakeSourceLines returns the lines in the supplied text. The returned lines will
// have one more extra line than the source for insertion use.
func MakeSourceLines(filename string, s []byte) []Line {
	var ret []Line

	lastRng := hcl.Range{
		Filename: filename,
		Start:    hcl.InitialPos,
		End:      hcl.InitialPos,
	}
	sc := hcl.NewRangeScanner(s, filename, scanLines)
	for sc.Scan() {
		ret = append(ret, Line{
			Bytes: sc.Bytes(),
			Range: sc.Range(),
		})
		lastRng = sc.Range()
	}

	// Account for the last (virtual) user-perceived line
	ret = append(ret, Line{
		Bytes: []byte{},
		Range: hcl.Range{
			Filename: lastRng.Filename,
			Start:    lastRng.End,
			End:      lastRng.End,
		},
	})
	return ret
}

// scanLines is a split function for a Scanner that returns each line of
// text (separated by \n), INCLUDING any trailing end-of-line marker.
// The last non-empty line of input will be returned even if it has no
// newline.
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[0 : i+1], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// StringLines returns the supplied lines as a sequence of strings.
func StringLines(lines []Line) []string {
	strLines := make([]string, len(lines))
	for i, l := range lines {
		strLines[i] = string(l.Bytes)
	}
	return strLines
}
