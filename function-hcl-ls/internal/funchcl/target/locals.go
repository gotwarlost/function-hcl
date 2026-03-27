package target

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/typeutils"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// this file contains the implementation of inferring schemas for local variables.
// For simple cases like constant string and some others, this can be done easily.
//
// In addition, we also implement "reference chasing".
// That is, if a local is defined as req.composite.spec, then assign it the attribute schema for
// spec. This works recursively such that derived locals based on another local that has a schema
// gets the appropriate sub-schema.

// local is a local variable found at a specific scope with an associated expression.
type local struct {
	name           string
	expr           hclsyntax.Expression
	enhancedSchema *schema.AttributeSchema
}

// localsCollection is a collection of locals at a specific scope
type localsCollection struct {
	parent           *localsCollection   // parent for this collection, if any
	parentCollection string              // if it is a descendant of a resources block
	parentResource   string              // if it is a descendant of a resource block
	locals           map[string]*local   // local variables
	scopedTo         hcl.Range           // scope for collection, zero-value is root
	children         []*localsCollection // child scopes

	// attributes set before enhancement
	globalSchema *schema.AttributeSchema
	fileSource   func(e hclsyntax.Expression) []byte
}

func newLocalsCollection(parent *localsCollection, scopedTo hcl.Range) *localsCollection {
	lc := &localsCollection{
		parent:   parent,
		locals:   map[string]*local{},
		scopedTo: scopedTo,
	}
	if parent != nil {
		parent.children = append(parent.children, lc)
		lc.parentCollection = parent.parentCollection
	}
	return lc
}

func (l *localsCollection) computeSchemas(globalSchema *schema.AttributeSchema, fileSource func(e hclsyntax.Expression) []byte) {
	l.globalSchema = globalSchema
	l.fileSource = fileSource
	for _, v := range l.locals {
		l.computeSchema(v)
	}
}

func (l *localsCollection) computeSchema(loc *local) {
	if loc.enhancedSchema != nil {
		return // already computed or circular ref
	}
	// first mark as seen to avoid infinite loops due to circular refs
	loc.enhancedSchema = unknownSchema
	// then try and compute the real thing
	e := l.impliedSchema(loc.expr)
	// for the `each` variable create the kv schema based on the collection
	// schema presumably returned.
	if loc.name == "each" {
		e = typeutils.KVSchema(e)
	}
	if e != nil {
		loc.enhancedSchema = e
	}
}

func (l *localsCollection) findOneFrom(expressions ...hclsyntax.Expression) *schema.AttributeSchema {
	for _, expr := range expressions {
		s := l.impliedSchema(expr)
		if s != nil {
			return s
		}
	}
	return nil
}

func (l *localsCollection) findUnionObjectType(expressions ...hclsyntax.Expression) *schema.AttributeSchema {
	attrs := map[string]*schema.AttributeSchema{}
	for _, expr := range expressions {
		s := l.impliedSchema(expr)
		if s == nil {
			return nil
		}
		// if it's a map, assume other args are also maps
		if _, ok := s.Constraint.(schema.Map); ok {
			return s
		}
		cons, ok := s.Constraint.(schema.Object)
		if !ok {
			return nil
		}
		for k, v := range cons.Attributes {
			seen, ok := attrs[k]
			if ok {
				if seen != unknownSchema {
					v = seen
				}
			}
			attrs[k] = v
		}
	}
	return &schema.AttributeSchema{Constraint: schema.Object{Attributes: attrs}}
}

func (l *localsCollection) impliedSchema(expr hclsyntax.Expression) (ret *schema.AttributeSchema) {
	switch e := expr.(type) {
	// literal values
	case *hclsyntax.LiteralValueExpr:
		v, diags := expr.Value(nil)
		if diags.HasErrors() || !v.IsWhollyKnown() {
			return nil
		}
		return &schema.AttributeSchema{Constraint: typeutils.TypeConstraint(v.Type())}
	case *hclsyntax.TemplateExpr:
		return &schema.AttributeSchema{Constraint: schema.String{}}

	// from function call signatures
	case *hclsyntax.FunctionCallExpr:
		return l.schemaFromFunctionCall(e)

	// from traversals
	case *hclsyntax.ScopeTraversalExpr:
		return l.schemaFromScopeTraversal(e)
	case *hclsyntax.RelativeTraversalExpr:
		return l.schemaFromRelativeTraversal(e)
	case *hclsyntax.IndexExpr:
		return l.schemaFromIndexExpression(e)
	case *hclsyntax.SplatExpr:
		return l.schemaFromSplat(e)

	// from operators: implied by operator used
	case *hclsyntax.BinaryOpExpr:
		return l.operationToSchema(e.Op)
	case *hclsyntax.UnaryOpExpr:
		return l.operationToSchema(e.Op)

	// descend into these
	case *hclsyntax.ObjectConsExpr:
		return l.objectSchemaFromItems(e.Items)
	case *hclsyntax.ConditionalExpr:
		return l.findOneFrom(e.TrueResult, e.FalseResult)
	case *hclsyntax.TupleConsExpr:
		inner := l.findOneFrom(e.Exprs...)
		if inner == nil {
			return nil
		}
		return &schema.AttributeSchema{Constraint: schema.List{Elem: inner.Constraint}}
	case *hclsyntax.ParenthesesExpr:
		return l.impliedSchema(e.Expression)
	case *hclsyntax.TemplateWrapExpr:
		return l.impliedSchema(e.Wrapped)

	case *hclsyntax.ExprSyntaxError:
		return

	// we don't know how to process these yet
	case *hclsyntax.ForExpr:
		return
	}
	return
}

func (l *localsCollection) objectSchemaFromItems(items []hclsyntax.ObjectConsItem) *schema.AttributeSchema {
	attrs := map[string]*schema.AttributeSchema{}
	for _, item := range items {
		k, diags := item.KeyExpr.Value(nil)
		if diags.HasErrors() {
			return nil
		}
		if k.Type() != cty.String {
			return nil
		}
		vs := l.impliedSchema(item.ValueExpr)
		if vs == nil {
			vs = unknownSchema
		}
		attrs[k.AsString()] = vs
	}
	return &schema.AttributeSchema{Constraint: schema.Object{Attributes: attrs}}
}

func (l *localsCollection) operationToSchema(op *hclsyntax.Operation) *schema.AttributeSchema {
	switch op {
	case hclsyntax.OpAdd,
		hclsyntax.OpSubtract,
		hclsyntax.OpMultiply,
		hclsyntax.OpDivide,
		hclsyntax.OpModulo,
		hclsyntax.OpNegate:
		return &schema.AttributeSchema{Constraint: schema.Number{}}

	case hclsyntax.OpEqual,
		hclsyntax.OpNotEqual,
		hclsyntax.OpLessThan,
		hclsyntax.OpLessThanOrEqual,
		hclsyntax.OpGreaterThan,
		hclsyntax.OpGreaterThanOrEqual,
		hclsyntax.OpLogicalAnd,
		hclsyntax.OpLogicalOr,
		hclsyntax.OpLogicalNot:
		return &schema.AttributeSchema{Constraint: schema.Bool{}}

	default:
		return nil
	}
}
