// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"bytes"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document/source"
)

// Change represents an edit where a specific range in a document is substituted with specific text.
// When no range is present, a full document change is assumed.
type Change interface {
	Text() string  // the text to use
	Range() *Range // the range to replace
}

// Changes is a list of change commands.
type Changes []Change

// ApplyChanges applies the supplied changes to the text supplied.
func ApplyChanges(original []byte, changes Changes) ([]byte, error) {
	if len(changes) == 0 {
		return original, nil
	}

	var buf bytes.Buffer
	_, err := buf.Write(original)
	if err != nil {
		return nil, err
	}

	for _, ch := range changes {
		err := applyDocumentChange(&buf, ch)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func applyDocumentChange(buf *bytes.Buffer, change Change) error {
	// if the range is nil, we assume it is full content change
	if change.Range() == nil {
		buf.Reset()
		_, err := buf.WriteString(change.Text())
		return err
	}

	lines := source.MakeSourceLines("", buf.Bytes())

	startByte, err := ByteOffsetForPos(lines, change.Range().Start)
	if err != nil {
		return err
	}
	endByte, err := ByteOffsetForPos(lines, change.Range().End)
	if err != nil {
		return err
	}

	diff := endByte - startByte
	if diff > 0 {
		buf.Grow(diff)
	}

	beforeChange := make([]byte, startByte)
	copy(beforeChange, buf.Bytes())
	afterBytes := buf.Bytes()[endByte:]
	afterChange := make([]byte, len(afterBytes))
	copy(afterChange, afterBytes)

	buf.Reset()

	_, err = buf.Write(beforeChange)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(change.Text())
	if err != nil {
		return err
	}
	_, err = buf.Write(afterChange)
	if err != nil {
		return err
	}

	return nil
}
