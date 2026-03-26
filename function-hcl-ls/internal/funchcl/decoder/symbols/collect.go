package symbols

import (
	"fmt"
	"sort"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder/decoderutils"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func (c *Collector) symbolsForBody(body hcl.Body) []Symbol {
	var symbols []Symbol
	if body == nil {
		return symbols
	}
	content := body.(*hclsyntax.Body)
	for name, attr := range content.Attributes {
		symbols = append(symbols, &AttributeSymbol{
			AttrName:      name,
			ExprKind:      symbolExprKind(attr.Expr),
			path:          c.path,
			rng:           attr.Range(),
			nestedSymbols: c.nestedSymbolsForExpr(attr.Expr),
		})
	}
	for _, block := range content.Blocks {
		symbols = append(symbols, &BlockSymbol{
			Type:          block.Type,
			Labels:        block.Labels,
			path:          c.path,
			rng:           block.Range(),
			nestedSymbols: c.symbolsForBody(block.Body),
		})
	}
	sort.SliceStable(symbols, func(i, j int) bool {
		return symbols[i].Range().Start.Byte < symbols[j].Range().Start.Byte
	})
	return symbols
}

func symbolExprKind(expr hcl.Expression) lang.SymbolExprKind {
	switch e := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		return lang.ReferenceExprKind{}
	case *hclsyntax.LiteralValueExpr:
		return lang.LiteralTypeKind{Type: e.Val.Type()}
	case *hclsyntax.TemplateExpr:
		if e.IsStringLiteral() {
			return lang.LiteralTypeKind{Type: cty.String}
		}
		if decoderutils.IsMultilineStringLiteral(e) {
			return lang.LiteralTypeKind{Type: cty.String}
		}
	case *hclsyntax.TupleConsExpr:
		return lang.TupleConsExprKind{}
	case *hclsyntax.ObjectConsExpr:
		return lang.ObjectConsExprKind{}
	default:
	}
	return nil
}

func (c *Collector) nestedSymbolsForExpr(expr hcl.Expression) []Symbol {
	var symbols []Symbol

	switch e := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		for i, item := range e.ExprList() {
			symbols = append(symbols, &ExprSymbol{
				ExprName:      fmt.Sprintf("%d", i),
				ExprKind:      symbolExprKind(item),
				path:          c.path,
				rng:           item.Range(),
				nestedSymbols: c.nestedSymbolsForExpr(item),
			})
		}
	case *hclsyntax.ObjectConsExpr:
		for _, item := range e.Items {
			key, _ := item.KeyExpr.Value(nil)
			if key.IsNull() || !key.IsWhollyKnown() || key.Type() != cty.String {
				// skip items keys that can't be interpolated
				// without further context
				continue
			}
			symbols = append(symbols, &ExprSymbol{
				ExprName:      key.AsString(),
				ExprKind:      symbolExprKind(item.ValueExpr),
				path:          c.path,
				rng:           hcl.RangeBetween(item.KeyExpr.Range(), item.ValueExpr.Range()),
				nestedSymbols: c.nestedSymbolsForExpr(item.ValueExpr),
			})
		}
	}
	return symbols
}
