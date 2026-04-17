package schema

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

var anyBody = &schema.BodySchema{}

var anyLabelSchema = make([]*schema.LabelSchema, 0)

var anyAttribute = &schema.AttributeSchema{
	Description: lang.PlainText("any value"),
	IsOptional:  true,
	Constraint:  schema.Any{},
}

var anyRequiredObjAttribute = &schema.AttributeSchema{
	Description: lang.PlainText("object value"),
	IsRequired:  true,
	Constraint: schema.Object{
		AnyAttribute: schema.Any{},
	},
}

var requiredMapStringAttribute = &schema.AttributeSchema{
	Description: lang.PlainText("map of strings"),
	IsRequired:  true,
	Constraint:  schema.Map{Elem: schema.String{}},
}

type lookup struct {
	dyn DynamicLookup
}

func (l *lookup) LabelSchema(bs schema.BlockStack) []*schema.LabelSchema {
	innermost, parent := bs.Peek(0).Type, bs.Peek(1).Type
	sch, ok := std[parent]
	if !ok {
		return anyLabelSchema
	}
	for t, nls := range sch.NestedBlocks {
		if t == innermost {
			return nls.Labels
		}
	}
	return anyLabelSchema
}

func (l *lookup) BodySchema(bs schema.BlockStack) *schema.BodySchema {
	innermostBlockType := bs.Peek(0).Type
	if sch, ok := std[innermostBlockType]; ok {
		return sch
	}
	return anyBody
}

func (l *lookup) compositeStatusSchema() *schema.AttributeSchema {
	cs, ok := l.dyn.(CompositeSchemaLookup)
	if !ok {
		return nil
	}
	s := cs.CompositeSchema()
	if s == nil {
		return nil
	}
	cons, ok := s.Constraint.(schema.Object)
	if !ok {
		return nil
	}
	return cons.Attributes["status"]
}

func (l *lookup) AttributeSchema(bs schema.BlockStack, attrName string) *schema.AttributeSchema {
	block := bs.Peek(0)
	blockName := block.Type
	switch {
	case blockName == "locals":
		return anyAttribute
	case attrName == "body" && blockName == "composite":
		if len(block.Labels) == 0 {
			return anyAttribute
		}
		switch block.Labels[0] {
		case "status":
			cs := l.compositeStatusSchema()
			if cs != nil {
				return cs
			}
			return anyRequiredObjAttribute
		case "connection":
			return requiredMapStringAttribute
		default:
			return anyAttribute
		}
	case attrName == "body" && (blockName == "resource" || blockName == "template"):
		dep, ok := dependentSchema(l.dyn, bs.Peek(0))
		if ok {
			return dep
		}
		fallthrough
	default:
		sch, ok := std[blockName]
		if !ok {
			return anyAttribute
		}
		as, ok := sch.Attributes[attrName]
		if !ok {
			return anyAttribute
		}
		return as
	}
}

// ImpliedAttributeSchema returns a computed schema implied by the attribute expression.
// This is called when hovering over the name of an attribute.
func (l *lookup) ImpliedAttributeSchema(bs schema.BlockStack, attrName string) *schema.AttributeSchema {
	block := bs.Peek(0)
	if block.Type != "locals" {
		return nil
	}
	al, ok := l.dyn.(LocalsAttributeLookup)
	if !ok {
		return nil
	}
	return al.LocalSchema(attrName)
}

func (l *lookup) Functions() map[string]schema.FunctionSignature {
	return stdFunctions() // TODO: take "invoke" and user defined functions into account
}

func dependentSchema(dyn DynamicLookup, bodyBlock *hclsyntax.Block) (*schema.AttributeSchema, bool) {
	attr, ok := bodyBlock.Body.Attributes["body"]
	if !ok {
		return nil, false
	}
	// ignore diags since value can be incomplete.
	val, _ := attr.Expr.Value(nil)
	if !val.Type().IsObjectType() && !val.Type().IsMapType() {
		return nil, false
	}
	if val.IsNull() || !val.IsKnown() {
		return nil, false
	}
	obj := val.AsValueMap()
	apiVersion, apiOK := obj["apiVersion"]
	kind, kindOK := obj["kind"]

	//nolint:staticcheck
	if !(apiOK && kindOK) {
		return nil, false
	}
	//nolint:staticcheck
	if !(apiVersion.Type() == cty.String && kind.Type() == cty.String) {
		return nil, false
	}
	dynamicSchema := dyn.Schema(apiVersion.AsString(), kind.AsString())
	if dynamicSchema == nil {
		return nil, false
	}
	return dynamicSchema, true
}
