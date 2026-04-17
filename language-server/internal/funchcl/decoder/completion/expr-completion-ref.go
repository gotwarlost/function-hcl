package completion

import (
	"log"
	"strings"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (e *expressionCompleter) findRefMatches(cons schema.Constraint, pathElements []string, editRange hcl.Range) []lang.Candidate {
	objCons, ok := cons.(schema.Object)
	if !ok {
		return nil
	}
	if len(pathElements) == 0 {
		log.Println("internal error: no path elements")
		return nil
	}
	first := pathElements[0]
	rest := pathElements[1:]
	remaining := len(rest)

	var ret []lang.Candidate
	switch remaining {
	case 0:
		for name, attr := range objCons.Attributes {
			if strings.HasPrefix(name, first) {
				text := name
				ret = append(ret, lang.Candidate{
					Label:       name,
					Description: attr.Description,
					Kind:        lang.ReferenceCandidateKind,
					TextEdit: lang.TextEdit{
						NewText: text,
						Snippet: text,
						Range:   editRange,
					},
				})
			}
		}
		return ret
	default:
		for name, attr := range objCons.Attributes {
			if name == first {
				ret = e.findRefMatches(attr.Constraint, rest, editRange)
			}
		}
	}
	return ret
}

func (e *expressionCompleter) completeRef(expr hclsyntax.Expression, _ *schema.AttributeSchema) []lang.Candidate {
	var candidates []lang.Candidate
	var prefixRange hcl.Range

	ctx := e.ctx
	pos := e.pos
	if isEmptyExpression(expr) {
		prefixRange = hcl.Range{
			Filename: expr.Range().Filename,
			Start:    pos,
			End:      pos,
		}
	}

	switch expr := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr, *hclsyntax.ExprSyntaxError:
		prefixRange = expr.Range()
		prefixRange.End = pos
	}

	if prefixRange.Filename == "" {
		return candidates
	}
	prefix := string(prefixRange.SliceBytes(ctx.FileBytes(expr)))
	if strings.Contains(prefix, "[") || strings.Contains(prefix, "*") { // we don't yet support anything other than a.b.c.d for now
		return candidates
	}
	elements := strings.Split(prefix, ".")
	editRange := prefixRange

	lastDot := strings.LastIndex(prefix, ".")
	if lastDot >= 0 {
		editRange.Start.Column += lastDot + 1
		editRange.Start.Byte += lastDot + 1
	}
	return e.findRefMatches(ctx.TargetSchema().Constraint, elements, editRange)
}
