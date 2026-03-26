package decoderutils

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func IsMultilineStringLiteral(tplExpr *hclsyntax.TemplateExpr) bool {
	if len(tplExpr.Parts) < 1 {
		return false
	}
	for _, part := range tplExpr.Parts {
		expr, ok := part.(*hclsyntax.LiteralValueExpr)
		if !ok {
			return false
		}
		if expr.Val.Type() != cty.String {
			return false
		}
	}
	return true
}

// IsMultilineTemplateExpr returns true if the expression is a template expression
// and spans more than one line.
func IsMultilineTemplateExpr(expr hclsyntax.Expression) bool {
	t, ok := expr.(*hclsyntax.TemplateExpr)
	if !ok {
		return false
	}
	return t.Range().Start.Line != t.Range().End.Line
}
