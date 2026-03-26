// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"fmt"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/zclconf/go-cty/cty"
)

// List represents a list, equivalent of a TupleConsExpr
// interpreted as list, i.e. ordering of item (which are all
// the same type) matters.
type List struct {
	// Elem defines constraint to apply to each item
	Elem Constraint
	// Description defines description of the whole list (affects hover)
	Description lang.MarkupContent
	// MinItems defines minimum number of items (affects completion)
	MinItems uint64
	// MaxItems defines maximum number of items (affects completion)
	MaxItems uint64
}

func (List) isConstraintImpl() constraintSigil {
	return constraintSigil{}
}

func (l List) FriendlyName() string {
	if l.Elem != nil && l.Elem.FriendlyName() != "" {
		return fmt.Sprintf("list of %s", l.Elem.FriendlyName())
	}
	return "list"
}

func (l List) Copy() Constraint {
	var elem Constraint
	if l.Elem != nil {
		elem = l.Elem.Copy()
	}
	return List{
		Elem:        elem,
		Description: l.Description,
		MinItems:    l.MinItems,
		MaxItems:    l.MaxItems,
	}
}

func (l List) EmptyCompletionData(nextPlaceholder int, nestingLevel int) CompletionData {
	if l.Elem == nil {
		return CompletionData{
			NewText:         "[ ]",
			Snippet:         fmt.Sprintf("[ ${%d} ]", nextPlaceholder),
			NextPlaceholder: nextPlaceholder + 1,
		}
	}

	elemData := l.Elem.EmptyCompletionData(nextPlaceholder, nestingLevel)
	if elemData.NewText == "" || elemData.Snippet == "" {
		return CompletionData{
			NewText:         "[ ]",
			Snippet:         fmt.Sprintf("[ ${%d} ]", nextPlaceholder),
			TriggerSuggest:  elemData.TriggerSuggest,
			NextPlaceholder: nextPlaceholder + 1,
		}
	}

	return CompletionData{
		NewText:         fmt.Sprintf("[ %s ]", elemData.NewText),
		Snippet:         fmt.Sprintf("[ %s ]", elemData.Snippet),
		NextPlaceholder: elemData.NextPlaceholder,
	}
}

func (l List) ConstraintType() (cty.Type, bool) {
	if l.Elem == nil {
		return cty.NilType, false
	}
	elemCons, ok := l.Elem.(TypeAwareConstraint)
	if !ok {
		return cty.NilType, false
	}
	elemType, ok := elemCons.ConstraintType()
	if !ok {
		return cty.NilType, false
	}
	return cty.List(elemType), true
}
