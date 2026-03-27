package completion

import (
	"reflect"
	"strings"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/writer"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// expressionCompleter provides completion information for expressions.
type expressionCompleter struct {
	extractor
	ctx   decoder.CompletionContext
	pos   hcl.Pos
	depth int
}

// newExpressionCompleter creates an expressionCompleter.
func newExpressionCompleter(ctx decoder.CompletionContext, pos hcl.Pos) *expressionCompleter {
	return &expressionCompleter{extractor: extractor{ctx: ctx}, ctx: ctx, pos: pos}
}

func (e *expressionCompleter) inCompletionRange(expr hclsyntax.Expression) bool {
	r := expr.Range()
	return r.ContainsPos(e.pos) || r.End.Byte == e.pos.Byte
}

func compressString(s string, maxLength int) string {
	if len(s) > maxLength && maxLength > 8 {
		s = s[:maxLength-8] + " ... " + s[len(s)-2:]
	}
	return strings.ReplaceAll(s, "\n", " ")
}

// complete returns completion data for the supplied expression having the supplied schema.
func (e *expressionCompleter) complete(expr hclsyntax.Expression, s *schema.AttributeSchema) []lang.Candidate {
	descend := func(eses ...exprSchema) []lang.Candidate {
		for _, es := range eses {
			if es.expr == nil {
				continue
			}
			if e.inCompletionRange(es.expr) {
				return e.complete(es.expr, es.schema)
			}
		}
		return nil
	}

	if debugCompletion {
		str := compressString(writer.NodeToSource(expr.(hclsyntax.Node)), 60)
		debugLogger.Printf("%-80s %s %s\n",
			strings.Repeat("  ", e.depth)+str,
			strings.TrimPrefix(reflect.TypeOf(expr).String(), "*hclsyntax."),
			s.Constraint.FriendlyName(),
		)
	}

	e.depth++
	defer func() { e.depth-- }()
	pos := e.pos

	if len(s.CompletionHooks) > 0 {
		candidates := e.candidatesFromHooks(expr, s)
		if len(candidates) > 0 {
			return candidates
		}
	}

	if isEmptyExpression(expr) {
		return e.completeEmptyExpression(expr, s)
	}

	switch expr := expr.(type) {
	case *hclsyntax.ExprSyntaxError:
		return e.standardRefs(expr, s)

	case *hclsyntax.ScopeTraversalExpr:
		return e.standardRefs(expr, s)

	case *hclsyntax.ObjectConsExpr:
		objCons, ok := s.Constraint.(schema.Object)
		if !ok {
			objCons = schema.Object{Attributes: map[string]*schema.AttributeSchema{}}
		}
		return e.completeObject(expr, objCons)

	// for tuples we simply descend the list.
	case *hclsyntax.TupleConsExpr:
		itemSchema := unknownSchema
		if listCons, ok := s.Constraint.(schema.List); ok {
			itemSchema = &schema.AttributeSchema{Constraint: listCons.Elem}
		}
		for _, ce := range expr.Exprs {
			if e.inCompletionRange(ce) {
				return e.complete(ce, itemSchema)
			}
		}

	// TODO: figure out how to process these
	case *hclsyntax.RelativeTraversalExpr:
	case *hclsyntax.IndexExpr:
	case *hclsyntax.SplatExpr:

	case *hclsyntax.FunctionCallExpr:
		funcSig, knownFunc := e.ctx.Functions()[expr.Name]
		if !knownFunc {
			funcSig = schema.FunctionSignature{
				Description: "unknown function",
				ReturnType:  cty.DynamicPseudoType,
				VarParam: &function.Parameter{
					Name: "unknown",
					Type: cty.DynamicPseudoType,
				},
			}
		}
		if expr.NameRange.ContainsPos(pos) {
			return nil // TODO: list other functions matching prefix at cursor
		}
		// special processing for merge: make args inherit the LHS schema
		if expr.Name == "merge" {
			return descend(withExpressionsOfSchema(s, expr.Args...)...)
		}
		for i, arg := range expr.Args {
			if !e.inCompletionRange(arg) {
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
			return e.complete(arg, argSchema)
		}

	case *hclsyntax.TemplateExpr:
		return e.completeTemplateExpr(expr)

	case *hclsyntax.ForExpr:
		// TODO: for expressions!

	// simpler descents for the remaining types.
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

	case *hclsyntax.ObjectConsKeyExpr:

	// no-ops: there is no valuable completion that can be provided for these expressions.
	case *hclsyntax.LiteralValueExpr:
	case *hclsyntax.AnonSymbolExpr:
	}
	return nil
}

// candidatesFromHooks returns hook candidates at the supplied position, correctly accounting for
// incomplete expressions and parse failures.
func (e *expressionCompleter) candidatesFromHooks(expr hcl.Expression, aSchema *schema.AttributeSchema) []lang.Candidate {
	return candidatesFromHooks(e.ctx, expr, aSchema, e.pos)
}

func (e *expressionCompleter) standardRefs(expr hclsyntax.Expression, s *schema.AttributeSchema) []lang.Candidate {
	var candidates []lang.Candidate
	candidates = append(candidates, e.completeRef(expr, s)...)
	candidates = append(candidates, e.completeFunction(expr, s)...)
	return candidates
}

func (e *expressionCompleter) completeEmptyExpression(expr hclsyntax.Expression, s *schema.AttributeSchema) []lang.Candidate {
	pos := e.pos
	// TODO: literal booleans
	returnRefs := func() []lang.Candidate {
		return e.standardRefs(expr, s)
	}

	switch cons := s.Constraint.(type) {
	// we don't have these use-cases yet, but keeping the placeholder here
	case schema.TypeDeclaration:
		return nil
	case schema.Object:
		cData := cons.EmptyCompletionData(1, 0)
		return []lang.Candidate{{
			Label:       "{…}",
			Detail:      "object",
			Kind:        lang.ObjectCandidateKind,
			Description: cons.Description,
			TextEdit: lang.TextEdit{
				NewText: cData.NewText,
				Snippet: cData.Snippet,
				Range: hcl.Range{
					Filename: expr.Range().Filename,
					Start:    pos,
					End:      pos,
				},
			},
			TriggerSuggest: cData.TriggerSuggest,
		}}
	case schema.List:
		d := cons.EmptyCompletionData(1, 0)
		return []lang.Candidate{{
			Label:       "[ ]",
			Detail:      cons.FriendlyName(),
			Kind:        lang.ListCandidateKind,
			Description: cons.Description,
			TextEdit: lang.TextEdit{
				NewText: d.NewText,
				Snippet: d.Snippet,
				Range: hcl.Range{
					Filename: expr.Range().Filename,
					Start:    pos,
					End:      pos,
				},
			},
			TriggerSuggest: d.TriggerSuggest,
		}}
	default:
		return returnRefs()
	}
}
