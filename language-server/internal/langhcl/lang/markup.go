// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lang

import "unique"

const (
	NilKind MarkupKind = iota
	PlainTextKind
	MarkdownKind
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=MarkupKind -output=markup_kind_string.go
type MarkupKind uint

// MarkupContent represents human-readable content
// which can be represented as Markdown or plaintext
// for backwards-compatible reasons.
type MarkupContent struct {
	value *unique.Handle[string]
	kind  MarkupKind
}

func (m MarkupContent) Kind() MarkupKind {
	return m.kind
}

func (m MarkupContent) Value() string {
	if m.value == nil {
		return ""
	}
	return m.value.Value()
}

func (m MarkupContent) AsDetail() string {
	v := m.Value()
	if v == "" {
		return ""
	}
	return "\n\n" + v
}

func NewMarkup(kind MarkupKind, value string) MarkupContent {
	h := unique.Make(value)
	return MarkupContent{
		kind:  kind,
		value: &h,
	}
}

func PlainText(value string) MarkupContent {
	return NewMarkup(PlainTextKind, value)
}

func Markdown(value string) MarkupContent {
	return NewMarkup(MarkdownKind, value)
}
