// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/zclconf/go-cty/cty"
)

// Object represents an object, equivalent of hclsyntax.ObjectConsExpr
// interpreted as object, i.e. with items of known keys
// and different value types.
type Object struct {
	Name                  string             // overrides friendly name of the constraint
	Attributes            ObjectAttributes   // names and constraints of attributes within the object
	Description           lang.MarkupContent //  description of the whole object (affects hover)
	AllowInterpolatedKeys bool               // determines whether the attribute names can be interpolated
	AnyAttribute          Constraint         // determines if we allow unknown attributes of this type
	PrefillRequiredKeys   bool               // prefill any required keys for the object
}

type ObjectAttributes map[string]*AttributeSchema

func (Object) isConstraintImpl() constraintSigil {
	return constraintSigil{}
}

func (o Object) FriendlyName() string {
	if o.Name == "" {
		return "object"
	}
	return o.Name
}

func (o Object) Copy() Constraint {
	return Object{
		Attributes:            o.Attributes.Copy(),
		Name:                  o.Name,
		Description:           o.Description,
		AllowInterpolatedKeys: o.AllowInterpolatedKeys,
		PrefillRequiredKeys:   o.PrefillRequiredKeys,
	}
}

func (o Object) EmptyCompletionData(placeholder int, nestingLevel int) CompletionData {
	nesting := strings.Repeat("  ", nestingLevel)
	attrNesting := strings.Repeat("  ", nestingLevel+1)
	triggerSuggest := len(o.Attributes) > 0

	emptyObjectData := CompletionData{
		NewText:         fmt.Sprintf("{\n%s\n%s}", attrNesting, nesting),
		Snippet:         fmt.Sprintf("{\n%s${%d}\n%s}", attrNesting, placeholder, nesting),
		NextPlaceholder: placeholder + 1,
		TriggerSuggest:  triggerSuggest,
	}

	if !o.PrefillRequiredKeys {
		return emptyObjectData
	}

	attrData, ok := o.attributesCompletionData(placeholder, nestingLevel)
	if !ok {
		return emptyObjectData
	}

	return CompletionData{
		NewText:         fmt.Sprintf("{\n%s%s}", attrData.NewText, nesting),
		Snippet:         fmt.Sprintf("{\n%s%s}", attrData.Snippet, nesting),
		NextPlaceholder: attrData.NextPlaceholder,
	}
}

func (o Object) attributesCompletionData(placeholder, nestingLevel int) (CompletionData, bool) {
	newText, snippet := "", ""
	anyRequiredFields := false
	attrNesting := strings.Repeat("  ", nestingLevel+1)
	nextPlaceholder := placeholder

	attrNames := sortedObjectExprAttrNames(o.Attributes)

	for _, name := range attrNames {
		attr := o.Attributes[name]
		attrData := attr.Constraint.EmptyCompletionData(nextPlaceholder, nestingLevel+1)
		if attrData.NewText == "" || attrData.Snippet == "" {
			return CompletionData{}, false
		}

		if attr.IsRequired {
			anyRequiredFields = true
		} else {
			continue
		}

		newText += fmt.Sprintf("%s%s = %s\n", attrNesting, name, attrData.NewText)
		snippet += fmt.Sprintf("%s%s = %s\n", attrNesting, name, attrData.Snippet)
		nextPlaceholder = attrData.NextPlaceholder
	}

	if anyRequiredFields {
		return CompletionData{
			NewText:         newText,
			Snippet:         snippet,
			NextPlaceholder: nextPlaceholder,
		}, true
	}

	return CompletionData{}, false
}

func sortedObjectExprAttrNames(attributes ObjectAttributes) []string {
	if len(attributes) == 0 {
		return []string{}
	}

	constraints := attributes
	names := make([]string, len(constraints))
	i := 0
	for name := range constraints {
		names[i] = name
		i++
	}

	sort.Strings(names)
	return names
}

func (o Object) ConstraintType() (cty.Type, bool) {
	objAttributes := make(map[string]cty.Type)

	for name, attr := range o.Attributes {
		cons, ok := attr.Constraint.(TypeAwareConstraint)
		if !ok {
			return cty.NilType, false
		}
		attrType, ok := cons.ConstraintType()
		if !ok {
			return cty.NilType, false
		}

		objAttributes[name] = attrType
	}

	return cty.Object(objAttributes), true
}

func (oa ObjectAttributes) Copy() ObjectAttributes {
	m := ObjectAttributes{}
	for name, aSchema := range oa {
		m[name] = aSchema.Copy()
	}
	return m
}
