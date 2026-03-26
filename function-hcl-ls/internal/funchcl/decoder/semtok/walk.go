package semtok

import (
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang/semtok"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type writer struct {
	stream []semtok.SemanticToken
}

func (w *writer) write(toks ...semtok.SemanticToken) {
	w.stream = append(w.stream, toks...)
}

func (w *writer) tokens() []semtok.SemanticToken {
	sort.Slice(w.stream, func(i, j int) bool {
		lr := w.stream[i].Range.Start
		rr := w.stream[j].Range.Start
		if lr.Line != rr.Line {
			return lr.Line < rr.Line
		}
		return lr.Column < rr.Column
	})
	return w.stream
}

type walker struct {
	w writer
	f *hcl.File
	b []byte
}

func newWalker(file *hcl.File, fileBytes []byte) *walker {
	return &walker{
		w: writer{},
		f: file,
		b: fileBytes,
	}
}

func (w *walker) write(toks ...semtok.SemanticToken) {
	w.w.write(toks...)
}

func (w *walker) fileTokens() []semtok.SemanticToken {
	body := w.f.Body.(*hclsyntax.Body)
	w.body(body, "")
	return w.w.tokens()
}

func (w *walker) body(body *hclsyntax.Body, parentBlockType string) {
	w.attribute(parentBlockType, body.Attributes)
	for _, block := range body.Blocks {
		w.block(block)
	}
}

func (w *walker) block(block *hclsyntax.Block) {
	bt := block.Type
	tt := semtok.TokenTypeKeyword
	w.write(makeToken(tt, block.TypeRange))

	for i := range block.Labels {
		switch i {
		case 0:
			decl := false
			tt = semtok.TokenTypeVariable
			switch bt {
			case "resource", "resources":
				tt = semtok.TokenTypeClass
				decl = true
			case "composite":
				tt = semtok.TokenTypeEnumMember
			}
			tok := makeToken(tt, block.LabelRanges[i])
			if decl {
				tok = withModifiers(tok, semtok.TokenModifierDefinition)
			}
			w.write(tok)
		}
	}
	w.body(block.Body, bt)
}

func (w *walker) attribute(blockType string, attributes hclsyntax.Attributes) {
	for _, attr := range attributes {
		if blockType != "locals" {
			w.write(makeToken(semtok.TokenTypeKeyword, attr.NameRange))
		} else {
			w.write(withModifiers(makeToken(semtok.TokenTypeVariable, attr.NameRange), semtok.TokenModifierDefinition))
		}
		w.expression(attr.Expr)
	}
}

func makeToken(tt semtok.TokenType, r hcl.Range) semtok.SemanticToken {
	return semtok.SemanticToken{
		Type:  tt,
		Range: r,
	}
}

func withModifiers(tok semtok.SemanticToken, mods ...semtok.TokenModifier) semtok.SemanticToken {
	tok.Modifiers = append(tok.Modifiers, mods...)
	return tok
}

func (w *walker) processScopeTraversal(exp *hclsyntax.ScopeTraversalExpr) bool {
	sourceCode := string(exp.Range().SliceBytes(w.b))
	if strings.Contains(sourceCode, "[") {
		return false
	}
	if strings.Contains(sourceCode, "*") {
		return false
	}
	if exp.Range().Start.Line != exp.Range().End.Line {
		return false
	}
	parts := strings.Split(sourceCode, ".")
	startPos := exp.Range().Start
	startByte := exp.Range().Start.Byte

	fileName := exp.Range().Filename
	makeRange := func(n int) hcl.Range {
		offset := 0
		for i := 0; i < n; i++ {
			offset += len(parts[i]) + 1 // including next dot
		}
		return hcl.Range{
			Filename: fileName,
			Start: hcl.Pos{
				Line:   startPos.Line,
				Column: startPos.Column + offset,
				Byte:   startByte + offset,
			},
			End: hcl.Pos{
				Line:   startPos.Line,
				Column: startPos.Column + offset + len(parts[n]),
				Byte:   startByte + offset + len(parts[n]),
			},
		}
	}
	for i, p := range parts {
		switch i {
		case 0:
			switch p {
			case "req", "each", "self":
				w.write(makeToken(semtok.TokenTypeKeyword, makeRange(i)))
			default:
				w.write(makeToken(semtok.TokenTypeVariable, makeRange(i)))
			}
		case 1:
			switch p {
			case "composite", "composite_connection", "context", "extra_resources":
				if parts[0] == "req" {
					w.write(makeToken(semtok.TokenTypeKeyword, makeRange(i)))
				} else {
					w.write(makeToken(semtok.TokenTypeProperty, makeRange(i)))
				}
			case "resource", "connection", "resources", "connections":
				if parts[0] == "req" || parts[0] == "self" {
					w.write(makeToken(semtok.TokenTypeKeyword, makeRange(i)))
				} else {
					w.write(makeToken(semtok.TokenTypeProperty, makeRange(i)))
				}
			case "name", "basename":
				if parts[0] == "self" {
					w.write(makeToken(semtok.TokenTypeKeyword, makeRange(i)))
				} else {
					w.write(makeToken(semtok.TokenTypeProperty, makeRange(i)))
				}
			}
		default:
			w.write(makeToken(semtok.TokenTypeProperty, makeRange(i)))
		}
	}
	return true
}

func (w *walker) expression(node hclsyntax.Expression) {
	if node == nil {
		return
	}
	switch node := node.(type) {
	case *hclsyntax.LiteralValueExpr:
		vt := node.Val.Type()
		switch vt {
		case cty.String:
			w.write(makeToken(semtok.TokenTypeString, node.Range()))
		case cty.Number:
			w.write(makeToken(semtok.TokenTypeNumber, node.Range()))
		case cty.Bool:
			w.write(makeToken(semtok.TokenTypeKeyword, node.Range()))
		default:
			// nop
		}

	case *hclsyntax.FunctionCallExpr:
		w.write(makeToken(semtok.TokenTypeFunction, node.NameRange))
		for _, arg := range node.Args {
			w.expression(arg)
		}

	case *hclsyntax.ForExpr:
		w.write(makeToken(semtok.TokenTypeKeyword, node.OpenRange))
		// do something about `in`, `if`, key name, value name etc.
		w.expression(node.KeyExpr)
		w.expression(node.CollExpr)
		w.expression(node.ValExpr)
		w.expression(node.CondExpr)

	case *hclsyntax.ObjectConsExpr:
		for _, item := range node.Items {
			w.expression(item.KeyExpr)
			w.expression(item.ValueExpr)
		}

	case *hclsyntax.ScopeTraversalExpr:
		if !w.processScopeTraversal(node) {
			w.write(makeToken(semtok.TokenTypeVariable, node.Range()))
		}

	case *hclsyntax.ObjectConsKeyExpr:
		switch node.Wrapped.(type) {
		case *hclsyntax.ScopeTraversalExpr:
			w.write(makeToken(semtok.TokenTypeProperty, node.Wrapped.Range()))
		default:
			w.expression(node.Wrapped)
		}

	case *hclsyntax.ConditionalExpr:
		w.expression(node.Condition)
		//w.write(makeToken(semtok.TokenTypeOperator, hcl.RangeBetween(node.Condition.Range(), node.TrueResult.Range())))
		w.expression(node.TrueResult)
		//w.write(makeToken(semtok.TokenTypeOperator, hcl.RangeBetween(node.TrueResult.Range(), node.FalseResult.Range())))
		w.expression(node.FalseResult)

	case *hclsyntax.BinaryOpExpr:
		w.expression(node.LHS)
		w.write(makeToken(semtok.TokenTypeOperator, hcl.RangeBetween(node.LHS.Range(), node.RHS.Range())))
		w.expression(node.RHS)

	case *hclsyntax.UnaryOpExpr:
		w.write(makeToken(semtok.TokenTypeOperator, node.SymbolRange))
		w.expression(node.Val)

	case *hclsyntax.TupleConsExpr:
		for _, e := range node.Exprs {
			w.expression(e)
		}
	case *hclsyntax.TemplateExpr:
		for _, e := range node.Parts {
			w.expression(e)
		}
	case *hclsyntax.TemplateWrapExpr:
		w.expression(node.Wrapped)
	case *hclsyntax.IndexExpr:
		w.expression(node.Key)
	case *hclsyntax.SplatExpr:
		// FIXME
	case *hclsyntax.ParenthesesExpr:
		w.expression(node.Expression)
	case *hclsyntax.AnonSymbolExpr:
		w.write(makeToken(semtok.TokenTypeVariable, node.Range()))
		// noop
	case *hclsyntax.ExprSyntaxError:
		// noop
	case *hclsyntax.RelativeTraversalExpr:
		w.expression(node.Source)
	default:
		log.Println("[warn] unknown expression type:", reflect.TypeOf(node))
	}
}
