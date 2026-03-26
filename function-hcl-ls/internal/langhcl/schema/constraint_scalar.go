package schema

import "github.com/zclconf/go-cty/cty"

// String represents a string type.
type String struct{}

func (String) isConstraintImpl() constraintSigil  { return constraintSigil{} }
func (s String) FriendlyName() string             { return cty.String.FriendlyNameForConstraint() }
func (s String) Copy() Constraint                 { return String{} }
func (s String) ConstraintType() (cty.Type, bool) { return cty.String, true }

func (s String) EmptyCompletionData(nextPlaceholder int, nestingLevel int) CompletionData {
	return noCompletion
}

// Number represents a number type.
type Number struct{}

func (Number) isConstraintImpl() constraintSigil  { return constraintSigil{} }
func (n Number) FriendlyName() string             { return cty.Number.FriendlyNameForConstraint() }
func (n Number) Copy() Constraint                 { return Number{} }
func (n Number) ConstraintType() (cty.Type, bool) { return cty.Number, true }

func (n Number) EmptyCompletionData(nextPlaceholder int, nestingLevel int) CompletionData {
	return noCompletion
}

// Bool represents a boolean type.
type Bool struct{}

func (Bool) isConstraintImpl() constraintSigil  { return constraintSigil{} }
func (b Bool) FriendlyName() string             { return cty.Bool.FriendlyNameForConstraint() }
func (b Bool) Copy() Constraint                 { return Bool{} }
func (b Bool) ConstraintType() (cty.Type, bool) { return cty.Bool, true }

func (b Bool) EmptyCompletionData(nextPlaceholder int, nestingLevel int) CompletionData {
	return noCompletion
}
