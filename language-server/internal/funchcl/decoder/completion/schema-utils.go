package completion

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/target"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/typeutils"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// extractor extracts interesting information from expressions.
type extractor struct {
	ctx decoder.CompletionContext
}

// traversalInfo holds the result of inspecting a (possibly partial) traversal.
type traversalInfo struct {
	source string
	rng    hcl.Range
	schema *schema.AttributeSchema
}

// extractTraversal extracts traversal information for the supplied traversal and hover position.
// For example, if the expression is `a.b.c` and the user positions on `b`, then information for `a.b`
// is returned.
func (e *extractor) extractTraversal(rootSchema *schema.AttributeSchema, expr hclsyntax.Expression,
	t hcl.Traversal, pos hcl.Pos) *traversalInfo {

	foundIndex := -1
	for i, t := range t {
		if t.SourceRange().ContainsPos(pos) {
			foundIndex = i
			break
		}
	}
	if foundIndex == -1 {
		return nil
	}
	wantTraversals := t[:foundIndex+1]
	sourceRange := hcl.RangeBetween(expr.StartRange(), t[foundIndex].SourceRange())
	sc := target.SchemaForRelativeTraversal(rootSchema, wantTraversals)
	return &traversalInfo{
		source: string(sourceRange.SliceBytes(e.ctx.FileBytes(expr))),
		rng:    sourceRange,
		schema: sc,
	}
}

// impliedSchema returns a schema for the supplied expression or an unknown schema.
func (e *extractor) impliedSchema(expr hclsyntax.Expression) *schema.AttributeSchema {
	switch expr := expr.(type) {

	// process a scope traversal fully or partially by taking the cursor position into account.
	case *hclsyntax.ScopeTraversalExpr:
		pos := hcl.Pos{
			Line:   expr.Range().End.Line,
			Column: expr.Range().End.Column,
			Byte:   expr.Range().End.Byte - 1,
		}
		ti := e.extractTraversal(e.ctx.TargetSchema(), expr, expr.Traversal, pos)
		if ti == nil {
			return unknownSchema
		}
		return ti.schema

	// ditto for a relative traversal except we need to compute the base implied schema first.
	case *hclsyntax.RelativeTraversalExpr:
		rootSchema := e.impliedSchema(expr.Source)
		return target.SchemaForRelativeTraversal(rootSchema, expr.Traversal)

	// for a splat expression we get the schema for the LHS. Then we need
	// to unwrap a list schema, calculate the schema relative to the unwrapped
	// schema and re-wrap the result into a list.
	case *hclsyntax.SplatExpr:
		checkRootSchema := e.impliedSchema(expr.Source)
		unwrapSchema := unknownSchema
		if s, ok := checkRootSchema.Constraint.(schema.List); ok {
			unwrapSchema = schemaForConstraint(s.Elem)
		}
		switch expr := expr.Each.(type) {
		case *hclsyntax.RelativeTraversalExpr:
			s := target.SchemaForRelativeTraversal(unwrapSchema, expr.Traversal)
			return schemaForConstraint(schema.List{Elem: s.Constraint})
		}

	// for an index expression we calculate the schema on the left. For maps,
	// and lists we don't care about the value of the index key and simply unwrap it.
	// For objects, we try and figure out what the index key is, if possible,
	// and return that subschema.
	case *hclsyntax.IndexExpr:
		rootSchema := e.impliedSchema(expr.Collection)
		switch cons := rootSchema.Constraint.(type) {
		case schema.List:
			return schemaForConstraint(cons.Elem)
		case schema.Map:
			return schemaForConstraint(cons.Elem)
		case schema.Object:
			v, diags := expr.Key.Value(nil)
			if diags.HasErrors() || !v.IsWhollyKnown() || v.Type() != cty.String {
				return unknownSchema
			}
			key := v.AsString()
			attrSchema, ok := cons.Attributes[key]
			if !ok {
				return unknownSchema
			}
			return attrSchema
		}

	// for an object constructor we return an object schema filled in to the extent possible.
	case *hclsyntax.ObjectConsExpr:
		attrs := map[string]*schema.AttributeSchema{}
		for _, item := range expr.Items {
			key, _, found := rawObjectKey(item.KeyExpr)
			if !found {
				continue
			}
			valSchema := e.impliedSchema(item.ValueExpr)
			attrs[key] = valSchema
		}
		return schemaForConstraint(schema.Object{Attributes: attrs})

	// literal schemas are reverse-engineered from their type.
	case *hclsyntax.LiteralValueExpr:
		return schemaForType(expr.Val.Type())

	// template expressions are, by definition, strings.
	case *hclsyntax.TemplateExpr:
		return stringSchema

	// for a function call, the implied schema is for its return type.
	case *hclsyntax.FunctionCallExpr:
		funcSig, knownFunc := e.ctx.Functions()[expr.Name]
		if !knownFunc {
			break
		}
		return schemaForType(funcSig.ReturnType)
	}

	// and we don't know how to process anything else.
	return unknownSchema
}

var (
	unknownConstraint = schema.Any{}
	unknownSchema     = &schema.AttributeSchema{Constraint: unknownConstraint}
	boolSchema        = &schema.AttributeSchema{Constraint: schema.Bool{}}
	stringSchema      = &schema.AttributeSchema{Constraint: schema.String{}}
	numberSchema      = &schema.AttributeSchema{Constraint: schema.Number{}}
)

func constraintForType(t cty.Type) schema.Constraint {
	return typeutils.TypeConstraint(t)
}

// schemaForConstraint returns an attribute schema that wraps the supplied constraint.
func schemaForConstraint(t schema.Constraint) *schema.AttributeSchema {
	return &schema.AttributeSchema{Constraint: t}
}

// schemaForType returns a schema for a value of the specified type
func schemaForType(t cty.Type) *schema.AttributeSchema {
	return schemaForConstraint(constraintForType(t))
}

// exprSchema pairs an expression with an attribute schema.
type exprSchema struct {
	expr   hclsyntax.Expression
	schema *schema.AttributeSchema
}

// withExpressionSchema returns an expSchema for the supplied expression and schema.
func withExpressionSchema(e hclsyntax.Expression, s *schema.AttributeSchema) exprSchema {
	return exprSchema{expr: e, schema: s}
}

// withUnknownExpression returns an expSchema for the supplied expression and an unknown schema.
func withUnknownExpression(e hclsyntax.Expression) exprSchema {
	return exprSchema{expr: e, schema: unknownSchema}
}

// withUnknownExpressions returns a list of expSchema for the supplied expressions and an unknown schema.
func withUnknownExpressions(e ...hclsyntax.Expression) []exprSchema {
	var ret []exprSchema
	for _, e := range e {
		ret = append(ret, withUnknownExpression(e))
	}
	return ret
}

// withExpressionsOfSchema returns a list of expSchema for the supplied schema an expression list.
func withExpressionsOfSchema(s *schema.AttributeSchema, e ...hclsyntax.Expression) []exprSchema {
	var ret []exprSchema
	for _, e := range e {
		ret = append(ret, withExpressionSchema(e, s))
	}
	return ret
}
