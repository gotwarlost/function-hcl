package completion

import (
	"fmt"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// expressionHover provides hover information for expressions.
type expressionHover struct {
	extractor
	ctx   decoder.CompletionContext
	pos   hcl.Pos
	depth int
}

// newExpressionHover creates an expressionHover.
func newExpressionHover(ctx decoder.CompletionContext, pos hcl.Pos) *expressionHover {
	return &expressionHover{extractor: extractor{ctx: ctx}, ctx: ctx, pos: pos}
}

// hover returns hover data for the supplied expression having the supplied schema. In many cases, this will
// return good hover content even when the schema is unknown based on the actual type of the underlying
// expression, if it is possible to compute this without the schema. For instance, we know the types of
// most local variables since this is inferred and present in the target schema tree.
func (e *expressionHover) hover(s *schema.AttributeSchema, expr hclsyntax.Expression) *lang.HoverData {
	descend := func(eses ...exprSchema) *lang.HoverData {
		for _, es := range eses {
			if es.expr == nil {
				continue
			}
			if es.expr.Range().ContainsPos(e.pos) {
				return e.hover(es.schema, es.expr)
			}
		}
		return nil
	}
	e.depth++
	defer func() { e.depth-- }()
	pos := e.pos

	switch expr := expr.(type) {

	// for objects, we need to understand whether we are positioned on an attribute name or its value.
	// When positioned on the name, we extract the schema from the supplied schema passed in and, if
	// this is unknown, we infer its type from the expression to which it is assigned. If positioned
	// on a value, we simply process the expression.
	case *hclsyntax.ObjectConsExpr:
		objCons, ok := s.Constraint.(schema.Object)
		if !ok {
			objCons = schema.Object{Attributes: map[string]*schema.AttributeSchema{}}
		}
		for _, item := range expr.Items {
			if !item.KeyExpr.Range().ContainsPos(pos) && !item.ValueExpr.Range().ContainsPos(pos) {
				continue
			}
			itemSchema := unknownSchema
			key, _, found := rawObjectKey(item.KeyExpr)
			if found && key != "" {
				as, ok1 := objCons.Attributes[key]
				if ok1 {
					itemSchema = as
				}
			}
			if item.KeyExpr.Range().ContainsPos(pos) {
				if itemSchema == unknownSchema {
					itemSchema = e.impliedSchema(item.ValueExpr)
				}
				if itemSchema != unknownSchema {
					return &lang.HoverData{
						Content: hoverContentForAttribute(key, itemSchema),
						Range:   item.KeyExpr.Range(),
					}
				}
			}
			if item.ValueExpr.Range().ContainsPos(pos) {
				return e.hover(itemSchema, item.ValueExpr)
			}
		}

	// for tuples we simply descend the list.
	case *hclsyntax.TupleConsExpr:
		itemSchema := unknownSchema
		if listCons, ok := s.Constraint.(schema.List); ok {
			itemSchema = &schema.AttributeSchema{Constraint: listCons.Elem}
		}
		for _, ce := range expr.Exprs {
			if ce.Range().ContainsPos(pos) {
				return e.hover(itemSchema, ce)
			}
		}

	// return hover content for the attribute on which the cursor hovers.
	case *hclsyntax.ScopeTraversalExpr:
		ti := e.extractTraversal(e.ctx.TargetSchema(), expr, expr.Traversal, e.pos)
		if ti == nil {
			return nil
		}
		return &lang.HoverData{
			Content: hoverContentForAttribute(ti.source, ti.schema),
			Range:   ti.rng,
		}

	// ditto for relative traversals, but compute source schema first.
	case *hclsyntax.RelativeTraversalExpr:
		if expr.SrcRange.ContainsPos(pos) {
			return e.hover(unknownSchema, expr.Source)
		}
		rootSchema := e.impliedSchema(expr.Source)
		ti := e.extractTraversal(rootSchema, expr, expr.Traversal, e.pos)
		if ti == nil {
			return nil
		}
		return &lang.HoverData{
			Content: hoverContentForAttribute(ti.source, ti.schema),
			Range:   ti.rng,
		}

	// ditto
	case *hclsyntax.IndexExpr:
		if expr.SrcRange.ContainsPos(pos) {
			return e.hover(unknownSchema, expr.Collection)
		}
		if expr.Key.Range().ContainsPos(pos) {
			sch := e.impliedSchema(expr.Key)
			if sch != unknownSchema {
				return &lang.HoverData{
					Content: hoverContentForAttribute(string(expr.Key.Range().SliceBytes(e.ctx.FileBytes(expr))), sch),
					Range:   expr.Key.Range(),
				}
			}
		}

	// if cursor on relative part, unwrap the LHS splat and re-wrap at the end.
	case *hclsyntax.SplatExpr:
		if expr.Source.Range().ContainsPos(pos) || expr.MarkerRange.ContainsPos(pos) {
			return e.hover(unknownSchema, expr.Source)
		}
		rel, isRel := expr.Each.(*hclsyntax.RelativeTraversalExpr)
		if !isRel {
			return nil
		}
		splatSchema := e.impliedSchema(expr.Source)
		listCons, isList := splatSchema.Constraint.(schema.List)
		var rootSchema *schema.AttributeSchema
		if !isList {
			return nil
		}
		rootSchema = schemaForConstraint(listCons.Elem)
		ti := e.extractTraversal(rootSchema, expr, rel.Traversal, e.pos)
		if ti == nil {
			return nil
		}
		ti.schema = schemaForConstraint(schema.List{Elem: ti.schema.Constraint})
		return &lang.HoverData{
			Content: hoverContentForAttribute(ti.source, ti.schema),
			Range:   ti.rng,
		}

	// for function calls, if positioned on the name return the function signature.
	// else infer a type for the argument on which the cursor hovers and use that.
	// Special case for the `merge` function: arguments are assigned to the LHS
	// attribute schema if known.
	case *hclsyntax.FunctionCallExpr:
		funcSig, knownFunc := e.ctx.Functions()[expr.Name]
		if expr.NameRange.ContainsPos(pos) {
			if !knownFunc {
				break
			}
			return &lang.HoverData{
				Content: hoverContentForFunction(expr.Name, funcSig),
				Range:   expr.NameRange,
			}
		}
		if expr.Name == "merge" {
			return descend(withExpressionsOfSchema(s, expr.Args...)...)
		}
		for i, arg := range expr.Args {
			if !arg.Range().ContainsPos(pos) {
				continue
			}
			argSchema := unknownSchema
			if knownFunc {
				if i < len(funcSig.Params) {
					argSchema = schemaForType(funcSig.Params[i].Type)
				} else if funcSig.VarParam != nil {
					argSchema = schemaForType(funcSig.VarParam.Type)
				}
			}
			return e.hover(argSchema, arg)
		}

	case *hclsyntax.ForExpr:
		// TODO: for expressions!

	// simpler descents for the remaining types.
	case *hclsyntax.TemplateExpr:
		return descend(withExpressionsOfSchema(stringSchema, expr.Parts...)...)

	case *hclsyntax.ConditionalExpr:
		return descend(
			withExpressionSchema(expr.Condition, boolSchema),
			withExpressionSchema(expr.TrueResult, s),
			withExpressionSchema(expr.FalseResult, s),
		)

	case *hclsyntax.BinaryOpExpr:
		switch expr.Op {
		case hclsyntax.OpAdd, hclsyntax.OpSubtract, hclsyntax.OpMultiply, hclsyntax.OpDivide, hclsyntax.OpModulo:
			return descend(withExpressionsOfSchema(numberSchema, expr.LHS, expr.RHS)...)
		case hclsyntax.OpLogicalAnd, hclsyntax.OpLogicalOr, hclsyntax.OpLogicalNot:
			return descend(withExpressionsOfSchema(boolSchema, expr.LHS, expr.RHS)...)
		}
		return descend(withUnknownExpressions(expr.LHS, expr.RHS)...)

	case *hclsyntax.UnaryOpExpr:
		return descend(withExpressionSchema(expr.Val, schemaForType(expr.Op.Type)))

	case *hclsyntax.TemplateWrapExpr:
		return descend(withExpressionSchema(expr.Wrapped, s))

	case *hclsyntax.ParenthesesExpr:
		return descend(withExpressionSchema(expr.Expression, s))

	// no-ops: there is no valuable hover that can be provided for these expressions.
	case *hclsyntax.ObjectConsKeyExpr:
	case *hclsyntax.LiteralValueExpr:
	case *hclsyntax.AnonSymbolExpr:
	case *hclsyntax.ExprSyntaxError:
	}
	return nil
}

func hoverContentForFunction(name string, funcSig schema.FunctionSignature) lang.MarkupContent {
	rawMd := fmt.Sprintf("```\n%s(%s) %s\n```\n\n%s",
		name, parameterNamesAsString(funcSig), funcSig.ReturnType.FriendlyName(), funcSig.Description)
	if funcSig.Detail != "" {
		rawMd += fmt.Sprintf("\n\n%s", funcSig.Detail)
	}
	return lang.Markdown(rawMd)
}
