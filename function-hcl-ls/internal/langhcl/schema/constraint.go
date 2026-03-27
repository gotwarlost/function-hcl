// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/zclconf/go-cty/cty"
)

var noCompletion = CompletionData{
	NewText:        "",
	Snippet:        "",
	TriggerSuggest: true,
}

type constraintSigil struct{}

type Constraint interface {
	isConstraintImpl() constraintSigil
	FriendlyName() string
	Copy() Constraint
	// EmptyCompletionData provides completion data in context where
	// there is no corresponding configuration, such as when the Constraint
	// is part of another, and it is desirable to complete
	// the parent constraint as whole.
	EmptyCompletionData(nextPlaceholder int, nestingLevel int) CompletionData
}

type Validatable interface {
	Validate() error
}

// TypeAwareConstraint represents a constraint which may be type-aware.
// Most constraints which implement this are always type-aware,
// but for some this is runtime concern depending on the configuration.
//
// This makes it comparable to another type for conformity during completion,
// and enables collection of type-aware reference target, if the attribute
// itself is targetable as type-aware.
type TypeAwareConstraint interface {
	ConstraintType() (cty.Type, bool)
}

type CompletionData struct {
	NewText string
	// Snippet represents text to be inserted via text edits,
	// with snippet placeholder identifiers such as ${1} (if any) starting
	// from given nextPlaceholder (provided as arg to EmptyCompletionData).
	Snippet         string
	TriggerSuggest  bool
	NextPlaceholder int
}

type HoverData struct {
	Content lang.MarkupContent
}
