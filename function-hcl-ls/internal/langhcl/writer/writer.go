// Package writer returns source code for HCL syntax nodes, for debugging use.
package writer

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// NodeToSource returns the source representation of the supplied syntax node.
func NodeToSource(n hclsyntax.Node) string {
	var w writer
	if expr, ok := n.(hclsyntax.Expression); ok {
		w.Expr(expr)
	} else {
		switch n := n.(type) {
		case *hclsyntax.Body:
			w.body(n)
		case *hclsyntax.Block:
			w.block(n)
		case *hclsyntax.Attribute:
			w.attribute(n)
		default:
			panic(fmt.Sprintf("unknown node type: %T", n))
		}
	}
	return w.result.String()
}

type writer struct {
	result    strings.Builder
	canonical bool
}

// Expr unpacks the struct implementing the expression and dispatches to
// the appropriate walker method.
func (w *writer) Expr(expr hclsyntax.Expression) {
	switch e := expr.(type) {
	case *hclsyntax.ExprSyntaxError:
		w.write("✗")
	case *hclsyntax.LiteralValueExpr:
		w.literalValue(e)
	case *hclsyntax.ScopeTraversalExpr:
		w.scopeTraversal(e)
	case *hclsyntax.RelativeTraversalExpr:
		w.relativeTraversal(e)
	case *hclsyntax.FunctionCallExpr:
		w.functionCall(e)
	case *hclsyntax.ConditionalExpr:
		w.conditional(e)
	case *hclsyntax.BinaryOpExpr:
		w.binaryOp(e)
	case *hclsyntax.UnaryOpExpr:
		w.unaryOp(e)
	case *hclsyntax.ObjectConsExpr:
		w.objectCons(e)
	case *hclsyntax.ObjectConsKeyExpr:
		w.objectConsKey(e)
	case *hclsyntax.TupleConsExpr:
		w.tupleCons(e)
	case *hclsyntax.TemplateExpr:
		w.template(e)
	case *hclsyntax.TemplateWrapExpr:
		w.templateWrap(e)
	case *hclsyntax.IndexExpr:
		w.index(e)
	case *hclsyntax.SplatExpr:
		w.splat(e)
	case *hclsyntax.ForExpr:
		w.forExpr(e)
	case *hclsyntax.ParenthesesExpr:
		w.parentheses(e)
	case *hclsyntax.AnonSymbolExpr:
		w.anonSymbol(e)
	default:
		w.write("<unexpected expression type ")
		w.write(reflect.TypeOf(expr).String())
		w.write(">")
	}
}

type writeable struct {
	attr  *hclsyntax.Attribute
	block *hclsyntax.Block
}

func (w writeable) isAttribute() bool {
	return w.attr != nil
}

func (w writeable) Range() hcl.Range {
	if w.isAttribute() {
		return w.attr.Range()
	}
	return w.block.Range()
}

func newAttrW(attr *hclsyntax.Attribute) writeable {
	return writeable{attr: attr}
}

func newBlockW(block *hclsyntax.Block) writeable {
	return writeable{block: block}
}

func (w *writer) body(body *hclsyntax.Body) {
	var ws []writeable
	for _, attr := range body.Attributes {
		ws = append(ws, newAttrW(attr))
	}
	for _, block := range body.Blocks {
		ws = append(ws, newBlockW(block))
	}
	canonSort := func(left, right writeable) bool {
		if left.isAttribute() && right.isAttribute() {
			return left.attr.Name < right.attr.Name
		}
		if !left.isAttribute() && !right.isAttribute() {
			return left.block.Type < right.block.Type
		}
		if left.isAttribute() {
			return true
		}
		return false
	}
	sort.Slice(ws, func(i, j int) bool {
		left := ws[i]
		right := ws[j]
		if w.canonical {
			return canonSort(left, right)
		} else {
			lr := left.Range()
			rr := right.Range()
			if lr.Start.Line == rr.Start.Line {
				return canonSort(left, right)
			}
			return lr.Start.Line < rr.Start.Line
		}
	})
	count := 0
	for _, wk := range ws {
		if wk.isAttribute() {
			w.attribute(wk.attr)
		} else {
			if count > 0 {
				w.write("\n")
			}
			w.block(wk.block)
		}
		count++
	}
}

func (w *writer) block(block *hclsyntax.Block) {
	w.write(block.Type)
	for _, label := range block.Labels {
		w.write(" ")
		w.write(unwrapIdent(strconv.Quote(label)))
	}
	w.write(" {\n")

	bodyContent := NodeToSource(block.Body)
	if bodyContent != "" {
		lines := strings.Split(strings.TrimRight(bodyContent, "\n"), "\n")
		for _, line := range lines {
			if line != "" {
				w.write("  ")
				w.write(line)
			}
			w.write("\n")
		}
	}

	w.write("}\n")
}

func (w *writer) attribute(attr *hclsyntax.Attribute) {
	w.write(attr.Name)
	w.write(" = ")
	w.Expr(attr.Expr)
	w.write("\n")
}

func (w *writer) write(s string) {
	w.result.WriteString(s)
}

func (w *writer) literalValue(e *hclsyntax.LiteralValueExpr) {
	w.write(literalValueToHCL(e.Val))
}

func (w *writer) scopeTraversal(e *hclsyntax.ScopeTraversalExpr) {
	traversal := e.Traversal
	for _, step := range traversal {
		switch s := step.(type) {
		case hcl.TraverseRoot:
			w.write(s.Name)
		case hcl.TraverseAttr:
			w.write(".")
			w.write(s.Name)
		case hcl.TraverseIndex:
			key := literalValueToHCL(s.Key)
			if IsQuotedIdentifier(key) {
				w.write(".")
				w.write(unwrapIdent(key))
			} else {
				w.write("[")
				w.write(key)
				w.write("]")
			}
		}
	}
}

func (w *writer) relativeTraversal(e *hclsyntax.RelativeTraversalExpr) {
	w.Expr(e.Source)
	for _, step := range e.Traversal {
		switch s := step.(type) {
		case hcl.TraverseAttr:
			w.write(".")
			w.write(s.Name)
		case hcl.TraverseIndex:
			key := literalValueToHCL(s.Key)
			w.write("[")
			w.write(key)
			w.write("]")
		}
	}
}

func (w *writer) index(e *hclsyntax.IndexExpr) {
	w.Expr(e.Collection)
	w.write("[")
	w.Expr(e.Key)
	w.write("]")
}

func (w *writer) binaryOp(e *hclsyntax.BinaryOpExpr) {
	w.Expr(e.LHS)
	w.write(" ")
	w.write(operationToHCL(e.Op))
	w.write(" ")
	w.Expr(e.RHS)
}

func (w *writer) unaryOp(e *hclsyntax.UnaryOpExpr) {
	w.write(operationToHCL(e.Op))
	w.Expr(e.Val)
}

func (w *writer) conditional(e *hclsyntax.ConditionalExpr) {
	w.Expr(e.Condition)
	w.write(" ? ")
	w.Expr(e.TrueResult)
	w.write(" : ")
	w.Expr(e.FalseResult)
}

func (w *writer) functionCall(e *hclsyntax.FunctionCallExpr) {
	w.write(e.Name)
	w.write("(")
	for i, arg := range e.Args {
		if i > 0 {
			w.write(", ")
		}
		w.Expr(arg)
	}
	w.write(")")
}

func (w *writer) tupleCons(e *hclsyntax.TupleConsExpr) {
	w.write("[")
	for i, elem := range e.Exprs {
		if i > 0 {
			w.write(", ")
		}
		w.Expr(elem)
	}
	w.write("]")
}

func (w *writer) objectCons(e *hclsyntax.ObjectConsExpr) {
	w.write("{\n")
	for _, item := range e.Items {
		w.write(unwrapIdent(NodeToSource(item.KeyExpr)))
		w.write(" = ")
		w.Expr(item.ValueExpr)
		w.write("\n")
	}
	w.write("}")
}

func (w *writer) objectConsKey(e *hclsyntax.ObjectConsKeyExpr) {
	w.Expr(e.Wrapped)
}

func (w *writer) template(e *hclsyntax.TemplateExpr) {
	w.write(templateExprToHCL(e))
}

func (w *writer) templateWrap(e *hclsyntax.TemplateWrapExpr) {
	w.Expr(e.Wrapped)
}

func (w *writer) parentheses(e *hclsyntax.ParenthesesExpr) {
	w.write("(")
	w.Expr(e.Expression)
	w.write(")")
}

func (w *writer) forExpr(e *hclsyntax.ForExpr) {
	// start bracket - [ for tuple, { for object
	if e.KeyExpr != nil {
		w.write("{")
	} else {
		w.write("[")
	}

	w.write("for ")

	// key variable (optional)
	if e.KeyVar != "" {
		w.write(e.KeyVar)
		w.write(", ")
	}

	// value variable
	w.write(e.ValVar)
	w.write(" in ")
	w.Expr(e.CollExpr)
	w.write(" : ")

	// key expression (for objects)
	if e.KeyExpr != nil {
		w.Expr(e.KeyExpr)
		w.write(" => ")
	}

	// value expression
	w.Expr(e.ValExpr)

	// condition (optional)
	if e.CondExpr != nil {
		w.write(" if ")
		w.Expr(e.CondExpr)
	}

	// group operator (optional)
	if e.Group {
		w.write("...")
	}

	// end bracket
	if e.KeyExpr != nil {
		w.write("}")
	} else {
		w.write("]")
	}
}

func (w *writer) splat(e *hclsyntax.SplatExpr) {
	w.Expr(e.Source)
	w.write("[*]")
	w.Expr(e.Each)
}

func (w *writer) anonSymbol(_ *hclsyntax.AnonSymbolExpr) {
}

// templateExprToHCL handles template expressions (strings with interpolation)
func templateExprToHCL(expr *hclsyntax.TemplateExpr) string {
	if len(expr.Parts) == 0 {
		return `""`
	}

	if len(expr.Parts) == 1 {
		if lit, ok := expr.Parts[0].(*hclsyntax.LiteralValueExpr); ok && lit.Val.Type() == cty.String {
			return strconv.Quote(lit.Val.AsString())
		}
	}

	var result strings.Builder
	result.WriteString(`"`)

	for _, part := range expr.Parts {
		switch p := part.(type) {
		case *hclsyntax.LiteralValueExpr:
			if p.Val.Type() == cty.String {
				// Escape special characters in string literals within templates
				s := p.Val.AsString()
				s = strings.ReplaceAll(s, `\`, `\\`)
				s = strings.ReplaceAll(s, `"`, `\"`)
				s = strings.ReplaceAll(s, `${`, `$${`)
				result.WriteString(s)
			} else {
				// Non-string literal in template (shouldn't happen normally)
				result.WriteString("${")
				result.WriteString(literalValueToHCL(p.Val))
				result.WriteString("}")
			}
		default:
			// Interpolation
			result.WriteString("${")
			result.WriteString(NodeToSource(p))
			result.WriteString("}")
		}
	}

	result.WriteString(`"`)
	return result.String()
}

// literalValueToHCL converts a cty.Value to its HCL literal representation
func literalValueToHCL(val cty.Value) string {
	if val.IsNull() {
		return "null"
	}
	switch val.Type() {
	case cty.String:
		return strconv.Quote(val.AsString())
	case cty.Number:
		bf := val.AsBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return strconv.FormatInt(i, 10)
		}
		f, _ := bf.Float64()
		return strconv.FormatFloat(f, 'f', -1, 64)
	case cty.Bool:
		if val.True() {
			return "true"
		}
		return "false"
	default:
		return "<unknown>"
	}
}

// operationToHCL converts an operation to its string representation
func operationToHCL(op *hclsyntax.Operation) string {
	switch op {
	case hclsyntax.OpAdd:
		return "+"
	case hclsyntax.OpSubtract:
		return "-"
	case hclsyntax.OpMultiply:
		return "*"
	case hclsyntax.OpDivide:
		return "/"
	case hclsyntax.OpModulo:
		return "%"
	case hclsyntax.OpEqual:
		return "=="
	case hclsyntax.OpNotEqual:
		return "!="
	case hclsyntax.OpLessThan:
		return "<"
	case hclsyntax.OpLessThanOrEqual:
		return "<="
	case hclsyntax.OpGreaterThan:
		return ">"
	case hclsyntax.OpGreaterThanOrEqual:
		return ">="
	case hclsyntax.OpLogicalAnd:
		return "&&"
	case hclsyntax.OpLogicalOr:
		return "||"
	case hclsyntax.OpLogicalNot:
		return "!"
	case hclsyntax.OpNegate:
		return "-"
	default:
		panic(fmt.Errorf("unable to handle operation of type %T", op))
	}
}

var reIdent = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

func IsIdentifier(s string) bool {
	return reIdent.MatchString(s)
}

func IsQuotedIdentifier(s string) bool {
	return len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && IsIdentifier(s[1:len(s)-1])
}

func unwrapIdent(s string) string {
	if IsQuotedIdentifier(s) {
		return s[1 : len(s)-1]
	}
	return s
}
