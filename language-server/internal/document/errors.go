// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"fmt"
)

type invalidPosErr struct {
	Pos Pos
}

func (e *invalidPosErr) Error() string {
	return fmt.Sprintf("invalid position: %s", e.Pos)
}

// NotFound returns an error that represents a missing document.
func NotFound(uri string) error {
	return &notFound{URI: uri}
}

// IsNotFound returns true if the supplied error refers to a missing document.
func IsNotFound(err error) bool {
	_, ok := err.(*notFound)
	return ok
}

type notFound struct {
	URI string
}

func (e *notFound) Error() string {
	msg := "document not found"
	if e.URI != "" {
		return fmt.Sprintf("%s: %s", e.URI, msg)
	}
	return msg
}

func (e *notFound) Is(err error) bool {
	_, ok := err.(*notFound)
	return ok
}
