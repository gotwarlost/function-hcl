package typeutils

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/zclconf/go-cty/cty"
)

// KVSchema returns a schema for the `each` variable. It is an object with a key and value property.
// If this cannot be derived, the function returns nil.
func KVSchema(base *schema.AttributeSchema) *schema.AttributeSchema {
	if base == nil {
		return nil
	}
	var ret *schema.AttributeSchema
	switch cons := base.Constraint.(type) {
	case schema.List:
		ct, ok := cons.ConstraintType()
		if ok {
			ret = collectionSchemaFromType(ct)
		}
	case schema.Map:
		ct, ok := cons.ConstraintType()
		if ok {
			ret = collectionSchemaFromType(ct)
		}
	case schema.Set:
		ct, ok := cons.ConstraintType()
		if ok {
			ret = collectionSchemaFromType(ct)
		}
	}
	return ret
}

// TypeConstraint returns a constraint for the supplied type.
func TypeConstraint(t cty.Type) schema.Constraint {
	switch {
	case t == cty.String:
		return schema.String{}
	case t == cty.Number:
		return schema.Number{}
	case t == cty.Bool:
		return schema.Bool{}
	case t.IsObjectType():
		types := t.AttributeTypes()
		inner := map[string]*schema.AttributeSchema{}
		for name, typ := range types {
			inner[name] = &schema.AttributeSchema{Constraint: TypeConstraint(typ)}
		}
		return schema.Object{Attributes: inner}
	case t.IsListType() && t.ListElementType() != nil:
		return schema.List{Elem: TypeConstraint(*t.ListElementType())}
	case t.IsMapType() && t.MapElementType() != nil:
		return schema.Map{Elem: TypeConstraint(*t.MapElementType())}
	case t.IsSetType() && t.SetElementType() != nil:
		return schema.Set{Elem: TypeConstraint(*t.SetElementType())}
	default:
		return schema.Any{}
	}
}

func collectionSchemaFromType(cType cty.Type) *schema.AttributeSchema {
	var inferredSchema *schema.AttributeSchema
	switch {
	case cType.IsMapType() && cType.MapElementType() != nil:
		inferredSchema = &schema.AttributeSchema{
			Constraint: schema.Object{
				Attributes: map[string]*schema.AttributeSchema{
					"key":   {Constraint: schema.String{}},
					"value": {Constraint: TypeConstraint(*cType.MapElementType())},
				},
			},
		}
	case cType.IsListType() && cType.ListElementType() != nil:
		inferredSchema = &schema.AttributeSchema{
			Constraint: schema.Object{
				Attributes: map[string]*schema.AttributeSchema{
					"key":   {Constraint: schema.Number{}},
					"value": {Constraint: TypeConstraint(*cType.ListElementType())},
				},
			},
		}
	case cType.IsTupleType() && len(cType.TupleElementTypes()) == 1:
		inferredSchema = &schema.AttributeSchema{
			Constraint: schema.Object{
				Attributes: map[string]*schema.AttributeSchema{
					"key":   {Constraint: schema.Number{}},
					"value": {Constraint: TypeConstraint(cType.TupleElementTypes()[0])},
				},
			},
		}
	case cType.IsSetType() && cType.SetElementType() != nil:
		cons := TypeConstraint(*cType.SetElementType())
		inferredSchema = &schema.AttributeSchema{
			Constraint: schema.Object{
				Attributes: map[string]*schema.AttributeSchema{
					"key":   {Constraint: cons},
					"value": {Constraint: cons},
				},
			},
		}
	}
	return inferredSchema
}
