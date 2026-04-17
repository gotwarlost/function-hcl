// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"github.com/zclconf/go-cty/cty"
)

// Any represents an unknown type that can be satisfied by any value.
type Any struct{}

func (Any) isConstraintImpl() constraintSigil   { return constraintSigil{} }
func (ae Any) FriendlyName() string             { return cty.DynamicPseudoType.FriendlyNameForConstraint() }
func (ae Any) Copy() Constraint                 { return Any{} }
func (ae Any) ConstraintType() (cty.Type, bool) { return cty.DynamicPseudoType, true }

func (ae Any) EmptyCompletionData(nextPlaceholder int, nestingLevel int) CompletionData {
	return noCompletion
}
