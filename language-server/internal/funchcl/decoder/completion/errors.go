// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package completion

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
)

type fileNotFoundError struct {
	filename string
}

func (e *fileNotFoundError) Error() string {
	return fmt.Sprintf("%s: file not found", e.filename)
}

type posOutOfRangeError struct {
	filename string
	pos      hcl.Pos
	rng      hcl.Range
}

func (e *posOutOfRangeError) Error() string {
	return fmt.Sprintf("%s: position %s is out of range %s", e.filename, posToStr(e.pos), e.rng)
}

type positionalError struct {
	filename string
	pos      hcl.Pos
	msg      string
}

func (e *positionalError) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.filename, posToStr(e.pos), e.msg)
}
