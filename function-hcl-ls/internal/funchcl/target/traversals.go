package target

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func (l *localsCollection) variable(name string) *local {
	current := l
	for current != nil {
		if loc, ok := current.locals[name]; ok {
			if current == l {
				l.computeSchema(loc)
			}
			if loc.enhancedSchema == unknownSchema {
				return nil
			}
			return loc
		}
		current = current.parent
	}
	return nil
}

func (l *localsCollection) selfSchema() *schema.AttributeSchema {
	if l.parentResource == "" && l.parentCollection == "" {
		return nil
	}
	attrs := map[string]*schema.AttributeSchema{}
	if l.parentCollection != "" {
		attrs["basename"] = &schema.AttributeSchema{Constraint: schema.String{}}
		attrs["connections"] = &schema.AttributeSchema{Constraint: schema.List{Elem: schema.Map{Elem: schema.String{}}}}
		attrs["resources"] = SubSchema(l.globalSchema, "req", "resources", l.parentCollection)
	}
	if l.parentResource != "" {
		attrs["name"] = &schema.AttributeSchema{Constraint: schema.String{}}
		attrs["connection"] = &schema.AttributeSchema{Constraint: &schema.Map{Elem: schema.String{}}}
		attrs["resource"] = SubSchema(l.globalSchema, "req", "resource", l.parentResource)
	}
	return &schema.AttributeSchema{
		Constraint: schema.Object{
			Name:        "self",
			Description: lang.PlainText("self reference"),
			Attributes:  attrs,
		},
	}
}

func (l *localsCollection) asStringValue(key hclsyntax.Expression) (string, bool) {
	switch key := key.(type) {
	// allow a single variable as an indirection mechanism - this should account for most cases
	case *hclsyntax.ScopeTraversalExpr:
		if len(key.Traversal) != 1 {
			return "", false
		}
		rootVar := key.Traversal[0].(hcl.TraverseRoot)
		v := l.variable(rootVar.Name)
		if v == nil {
			return "", false
		}
		_, ok1 := v.expr.(*hclsyntax.TemplateExpr)
		_, ok2 := v.expr.(*hclsyntax.LiteralValueExpr)
		if !ok1 && !ok2 { // prevent cycles with this check otherwise foo = foo will keep on spinning
			return "", false
		}
		return l.asStringValue(v.expr)
	default:
		v, diags := key.Value(nil)
		if diags.HasErrors() || !v.IsWhollyKnown() {
			return "", false
		}
		if v.Type() != cty.String {
			return "", false
		}
		return v.AsString(), true
	}
}

func (l *localsCollection) schemaFromScopeTraversal(e *hclsyntax.ScopeTraversalExpr) *schema.AttributeSchema {
	t := e.Traversal
	if len(t) == 0 {
		return nil
	}
	root := t[0].(hcl.TraverseRoot).Name

	switch root {
	case "req": // can only occur in the global tree
		return processRelativeTraversal(l.globalSchema, t)
	case "self":
		return processRelativeTraversal(l.selfSchema(), t[1:])
	default: // must be a local at our scope or a parent scope, note that each is treated as a local
		v := l.variable(root)
		if v == nil {
			return nil
		}
		return processRelativeTraversal(v.enhancedSchema, t[1:])
	}
}

func (l *localsCollection) schemaFromIndexExpression(e *hclsyntax.IndexExpr) *schema.AttributeSchema {
	collSchema := l.impliedSchema(e.Collection)
	if collSchema == nil {
		return nil
	}
	cons := collSchema.Constraint
	// for maps and list we do not care about what the index is
	// the return value is the element schema for the inner element.
	switch cons := cons.(type) {
	case schema.Map:
		return &schema.AttributeSchema{Constraint: cons.Elem}
	case schema.List:
		return &schema.AttributeSchema{Constraint: cons.Elem}
	// for objects, we need to eval the key and turn it into a string, if possible
	case schema.Object:
		key, ok := l.asStringValue(e.Key)
		if !ok {
			return nil
		}
		attr, ok := cons.Attributes[key]
		if !ok {
			return nil
		}
		return &schema.AttributeSchema{Constraint: attr.Constraint}
	}
	return nil
}

func (l *localsCollection) schemaFromRelativeTraversal(e *hclsyntax.RelativeTraversalExpr) *schema.AttributeSchema {
	sourceSchema := l.impliedSchema(e.Source)
	return processRelativeTraversal(sourceSchema, e.Traversal)
}

func (l *localsCollection) schemaFromSplat(e *hclsyntax.SplatExpr) *schema.AttributeSchema {
	sourceSchema := l.impliedSchema(e.Source)
	if sourceSchema == nil {
		return nil
	}
	listCons, ok := sourceSchema.Constraint.(schema.List)
	if !ok {
		return nil
	}
	_, anonEach := e.Each.(*hclsyntax.AnonSymbolExpr)
	if e.Each == nil || anonEach {
		return sourceSchema
	}
	rel, ok := e.Each.(*hclsyntax.RelativeTraversalExpr)
	if !ok {
		return nil
	}
	// unwrap
	rootSchema := &schema.AttributeSchema{Constraint: listCons.Elem}
	elementSchema := processRelativeTraversal(rootSchema, rel.Traversal)
	if elementSchema == nil {
		return nil
	}
	// wrap
	return &schema.AttributeSchema{Constraint: schema.List{Elem: elementSchema.Constraint}}
}
