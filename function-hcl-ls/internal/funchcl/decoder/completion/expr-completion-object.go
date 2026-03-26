package completion

import (
	"bytes"
	"sort"
	"strings"
	"unicode"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type declaredAttributes map[string]hcl.Range

func (e *expressionCompleter) completeObject(expr *hclsyntax.ObjectConsExpr, obj schema.Object) []lang.Candidate {
	pos := e.pos
	betweenBraces := hcl.Range{
		Filename: expr.Range().Filename,
		Start:    expr.OpenRange.End,
		End:      expr.Range().End,
	}

	if !betweenBraces.ContainsPos(pos) {
		return nil
	}

	editRange := hcl.Range{
		Filename: expr.Range().Filename,
		Start:    pos,
		End:      pos,
	}

	declared := declaredAttributes{}
	recoveryPos := expr.OpenRange.Start
	var lastItemRange, nextItemRange *hcl.Range

	for _, item := range expr.Items {
		emptyRange := hcl.Range{
			Filename: expr.Range().Filename,
			Start:    item.KeyExpr.Range().End,
			End:      item.ValueExpr.Range().Start,
		}
		if emptyRange.ContainsPos(pos) {
			// exit early if we're in empty space between key and value
			return nil
		}

		attrName, attrRange, isRawName := rawObjectKey(item.KeyExpr)
		if isRawName {
			// collect all declared attributes
			declared[attrName] = hcl.RangeBetween(item.KeyExpr.Range(), item.ValueExpr.Range())
		}

		if nextItemRange != nil {
			continue
		}
		// check if we've just missed the position
		if pos.Byte < item.KeyExpr.Range().Start.Byte {
			// record current (next) item so we can avoid completion
			// on the same line in multi-line mode (without comma)
			nextItemRange = hcl.RangeBetween(item.KeyExpr.Range(), item.ValueExpr.Range()).Ptr()
			// enable recovery of incomplete configuration
			// between last item's end and position
			continue
		}
		lastItemRange = hcl.RangeBetween(item.KeyExpr.Range(), item.ValueExpr.Range()).Ptr()
		recoveryPos = item.ValueExpr.Range().End

		if item.KeyExpr.Range().ContainsPos(pos) {
			// handle any interpolation if it is allowed
			keyExpr, ok := item.KeyExpr.(*hclsyntax.ObjectConsKeyExpr)
			if ok && obj.AllowInterpolatedKeys {
				parensExpr, ok := keyExpr.Wrapped.(*hclsyntax.ParenthesesExpr)
				if ok {
					return e.complete(parensExpr, &schema.AttributeSchema{Constraint: schema.String{}})
				}
			}

			if isRawName {
				prefix := ""
				// if we're before start of the attribute
				// it means the attribute is likely quoted
				if pos.Byte >= attrRange.Start.Byte {
					prefixLen := pos.Byte - attrRange.Start.Byte
					prefix = attrName[0:prefixLen]
				}
				editRange = hcl.RangeBetween(item.KeyExpr.Range(), item.ValueExpr.Range())
				return attributeCandidates(prefix, obj.Attributes, declared, editRange)
			}
			return nil
		}

		if e.inCompletionRange(item.ValueExpr) {
			aSchema, ok := obj.Attributes[attrName]
			if !ok {
				if obj.AnyAttribute != nil {
					aSchema = &schema.AttributeSchema{Constraint: obj.AnyAttribute}
				} else {
					aSchema = &schema.AttributeSchema{Constraint: schema.Any{}}
				}
			}
			return e.complete(item.ValueExpr, aSchema)
		}
	}

	// check any incomplete configuration up to a terminating character
	fileBytes := e.ctx.FileBytes(expr)
	leftBytes := recoverLeftBytes(fileBytes, pos, func(offset int, r rune) bool {
		return isObjectItemTerminatingRune(r) && offset > recoveryPos.Byte
	})
	trimmedBytes := bytes.TrimRight(leftBytes, " \t")

	if len(trimmedBytes) == 0 {
		// no terminating character was found which indicates
		// we're on the same line as an existing item,
		// and we're missing preceding comma
		return nil
	}

	if len(trimmedBytes) == 1 && isObjectItemTerminatingRune(rune(trimmedBytes[0])) {
		// avoid completing on the same line as next item
		if nextItemRange != nil && nextItemRange.Start.Line == pos.Line {
			return nil
		}
		// avoid completing on the same line as last item
		if lastItemRange != nil && lastItemRange.End.Line == pos.Line {
			// if it is not single-line notation
			if trimmedBytes[0] != ',' {
				return nil
			}
		}
		return attributeCandidates("", obj.Attributes, declared, editRange)
	}

	// trim left side as well now
	// to make prefix/attribute extraction easier below
	trimmedBytes = bytes.TrimLeftFunc(trimmedBytes, func(r rune) bool {
		return isObjectItemTerminatingRune(r) || unicode.IsSpace(r)
	})

	// parenthesis implies interpolated attribute name
	if trimmedBytes[len(trimmedBytes)-1] == '(' && obj.AllowInterpolatedKeys {
		emptyExpr := newEmptyExpressionAtPos(expr.Range().Filename, pos)
		return e.complete(emptyExpr.(hclsyntax.Expression), &schema.AttributeSchema{Constraint: schema.String{}})
	}

	// if last byte is =, then it's incomplete attribute
	//nolint:staticcheck
	if len(trimmedBytes) > 0 && trimmedBytes[len(trimmedBytes)-1] == '=' {
		emptyExpr := newEmptyExpressionAtPos(expr.Range().Filename, pos)

		attrName := string(bytes.TrimFunc(trimmedBytes[:len(trimmedBytes)-1], func(r rune) bool {
			return unicode.IsSpace(r) || r == '"'
		}))
		aSchema, ok := obj.Attributes[attrName]
		if !ok {
			if obj.AnyAttribute != nil {
				aSchema = &schema.AttributeSchema{Constraint: obj.AnyAttribute}
			} else {
				// unknown attribute
				return nil
			}
		}
		return e.complete(emptyExpr.(hclsyntax.Expression), aSchema)
	}

	prefix := string(bytes.TrimFunc(trimmedBytes, func(r rune) bool {
		return unicode.IsSpace(r) || r == '"'
	}))

	// calculate appropriate edit range in case there
	// are also characters on the right from position
	// which are worth replacing
	remainingRange := hcl.Range{
		Filename: expr.Range().Filename,
		Start:    pos,
		End:      expr.SrcRange.End,
	}
	editRange = e.objectItemPrefixBasedEditRange(remainingRange, fileBytes, trimmedBytes)

	return attributeCandidates(prefix, obj.Attributes, declared, editRange)
}

func (e *expressionCompleter) objectItemPrefixBasedEditRange(remainingRange hcl.Range, fileBytes []byte, rawPrefixBytes []byte) hcl.Range {
	remainingBytes := remainingRange.SliceBytes(fileBytes)
	roughEndByteOffset := bytes.IndexFunc(remainingBytes, func(r rune) bool {
		return r == '\n' || r == '}'
	})
	if roughEndByteOffset >= 0 {
		remainingBytes = remainingBytes[roughEndByteOffset:]
	}
	// avoid editing over whitespace
	trimmedRightBytes := bytes.TrimRightFunc(remainingBytes, func(r rune) bool {
		return unicode.IsSpace(r)
	})
	trimmedOffset := len(trimmedRightBytes)
	return hcl.Range{
		Filename: remainingRange.Filename,
		Start: hcl.Pos{
			// TODO: Calculate Line+Column for multi-line keys?
			Line:   remainingRange.Start.Line,
			Column: remainingRange.Start.Column - len(rawPrefixBytes),
			Byte:   remainingRange.Start.Byte - len(rawPrefixBytes),
		},
		End: hcl.Pos{
			// TODO: Calculate Line+Column for multi-line values?
			Line:   remainingRange.Start.Line,
			Column: remainingRange.Start.Column + trimmedOffset,
			Byte:   remainingRange.Start.Byte + trimmedOffset,
		},
	}
}

// attributeCandidates returns completion candidates for attributes matching prefix.
// Already-declared attributes are excluded unless their range overlaps editRange.
func attributeCandidates(prefix string, attrs map[string]*schema.AttributeSchema, declared declaredAttributes, editRange hcl.Range) []lang.Candidate {
	if len(attrs) == 0 {
		return nil
	}
	names := make([]string, 0, len(attrs))
	for n := range attrs {
		names = append(names, n)
	}
	sort.Strings(names)
	var candidates []lang.Candidate
	for _, name := range names {
		if len(prefix) > 0 && !strings.HasPrefix(name, prefix) {
			continue
		}
		if declared != nil {
			if declaredRng, ok := declared[name]; ok && !declaredRng.Overlaps(editRange) {
				continue
			}
		}
		candidates = append(candidates, attributeSchemaToCandidate(name, attrs[name], editRange))
	}
	return candidates
}
