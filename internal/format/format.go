package format

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type Options struct {
	StandardizeObjectLiterals bool
}

// Source returns the formatted source code, optionally standardizing object literals
// to always be in key = value format, for consistency and better indentation.
func Source(source string, opts Options) string {
	file, diags := hclwrite.ParseConfig([]byte(source), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return source
	}
	if opts.StandardizeObjectLiterals {
		processBody(file.Body())
	}
	tokens := file.Body().BuildTokens(nil)
	return string(hclwrite.Format(tokens.Bytes()))
}

func processBody(body *hclwrite.Body) {
	for _, block := range body.Blocks() {
		processBody(block.Body())
	}
	processAttributes(body)
}

func processAttributes(body *hclwrite.Body) {
	attrs := body.Attributes()
	for name, attr := range attrs {
		tokens := fixObjectLiteralStyle(attr.BuildTokens(nil))
		body.SetAttributeRaw(name, tokens)
	}
}

type opType int

const (
	_ opType = iota
	NoContextAvailable
	ObjectLiteral
	ForExpression
	TernaryExpression
)

func (t opType) String() string {
	switch t {
	case NoContextAvailable:
		return "NoContextAvailable"
	case ObjectLiteral:
		return "ObjectLiteral"
	case ForExpression:
		return "ForExpression"
	case TernaryExpression:
		return "TernaryExpression"
	default:
		return "Unknown"
	}
}

type stack struct {
	ops []opType
}

func (s *stack) push(op opType) {
	s.ops = append(s.ops, op)
}

func (s *stack) pop() {
	if len(s.ops) == 0 {
		return
	}
	s.ops = s.ops[:len(s.ops)-1]
}

func (s *stack) peek() opType {
	if len(s.ops) == 0 {
		return NoContextAvailable
	}
	return s.ops[len(s.ops)-1]
}

func (s *stack) hasItems() bool {
	return len(s.ops) != 0
}

func newStack() *stack {
	return &stack{ops: []opType{NoContextAvailable}}
}

// extractContent extracts the attribute value from the full attribute declaration
// by eliminating the identifier and equals token from the left and line feeds and comments
// from the right. This is due to the asymmetry of getting attribute expression that include
// the variable being assigned and trailing comments versus setting the attribute tokens
// that should omit both these things.
func extractContent(tokens hclwrite.Tokens) (hclwrite.Tokens, error) {
	baseIndex := 0
	for i, t := range tokens {
		if t.Type == hclsyntax.TokenEqual {
			baseIndex = i + 1
			break
		}
	}
	if baseIndex == 0 {
		return tokens, fmt.Errorf("no assignment found")
	}
	end := len(tokens)
	for end > baseIndex {
		if tokens[end-1].Type == hclsyntax.TokenNewline || tokens[end-1].Type == hclsyntax.TokenComment {
			end--
		} else {
			break
		}
	}
	return tokens[baseIndex:end], nil
}

// fixObjectLiteralStyle replaces colons with equal signs for object
// attributes so that they are consistent and format better.
// This requires selectively replacing colons in a way that they
// do not get accidentally replaced for `for` expresssions or
// ternary conditions.
func fixObjectLiteralStyle(input hclwrite.Tokens) hclwrite.Tokens {
	tokens, err := extractContent(input)
	if err != nil {
		log.Println(err)
		return tokens
	}
	result := make(hclwrite.Tokens, 0, len(tokens))
	s := newStack()
	for _, t := range tokens {
		token := *t

		switch token.Type {
		case hclsyntax.TokenOBrace:
			s.push(ObjectLiteral)
		case hclsyntax.TokenQuestion:
			s.push(TernaryExpression)
		case hclsyntax.TokenIdent:
			if string(token.Bytes) == "for" {
				s.push(ForExpression)
			}
		case hclsyntax.TokenCBrace:
			if !s.hasItems() {
				log.Println("[warn] fixObjectLiteralStyle: stack has no items")
				return tokens
			}
			have := s.peek()
			if have != ObjectLiteral {
				log.Println("[warn] fixObjectLiteralStyle: close brace was not matched")
				return tokens
			}
			s.pop()
		case hclsyntax.TokenColon:
			have := s.peek()
			switch have {
			case ObjectLiteral:
				token.Type = hclsyntax.TokenEqual
				token.Bytes = bytes.ReplaceAll(token.Bytes, []byte{':'}, []byte{'='})
			case ForExpression, TernaryExpression:
				s.pop()
			default:
				log.Println("fixObjectLiteralStyle: no context available for colon")
				return tokens
			}
		}
		result = append(result, &token)
	}
	return result
}
