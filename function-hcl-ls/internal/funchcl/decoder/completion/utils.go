package completion

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder/decoderutils"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"
	"github.com/zclconf/go-cty/cty"
)

// isEmptyExpression returns true if given expression is suspected
// to be empty, e.g. newline after equal sign.
//
// Because upstream HCL parser doesn't always handle incomplete
// configuration gracefully, this may not cover all cases.
func isEmptyExpression(expr hcl.Expression) bool {
	l, ok := expr.(*hclsyntax.LiteralValueExpr)
	if !ok {
		return false
	}
	if l.Val != cty.DynamicVal {
		return false
	}
	return true
}

// newEmptyExpressionAtPos returns a new "artificial" empty expression
// which can be used during completion inside another expression
// in an empty space which isn't already represented by empty expression.
//
// For example, new argument after comma in function call,
// or new element in a list or set.
func newEmptyExpressionAtPos(filename string, pos hcl.Pos) hcl.Expression {
	return &hclsyntax.LiteralValueExpr{
		Val: cty.DynamicVal,
		SrcRange: hcl.Range{
			Filename: filename,
			Start:    pos,
			End:      pos,
		},
	}
}

// recoverLeftBytes seeks left from given pos in given slice of bytes
// and recovers all bytes up until f matches, including that match.
// This allows recovery of incomplete configuration which is not
// present in the parsed AST during completion.
//
// Zero bytes is returned if no match was found.
func recoverLeftBytes(b []byte, pos hcl.Pos, f func(byteOffset int, r rune) bool) []byte {
	firstRune, size := utf8.DecodeLastRune(b[:pos.Byte])
	offset := pos.Byte - size

	// check for early match
	if f(pos.Byte, firstRune) {
		return b[offset:pos.Byte]
	}

	for offset > 0 {
		nextRune, size := utf8.DecodeLastRune(b[:offset])
		if f(offset, nextRune) {
			// record the matched offset
			// and include the matched last rune
			startByte := offset - size
			return b[startByte:pos.Byte]
		}
		offset -= size
	}

	return []byte{}
}

// recoverRightBytes seeks right from given pos in given slice of bytes
// and recovers all bytes up until f matches, including that match.
// This allows recovery of incomplete configuration which is not
// present in the parsed AST during completion.
//
// Zero bytes is returned if no match was found.
func recoverRightBytes(b []byte, pos hcl.Pos, f func(byteOffset int, r rune) bool) []byte {
	nextRune, size := utf8.DecodeRune(b[pos.Byte:])
	offset := pos.Byte + size

	// check for early match
	if f(pos.Byte, nextRune) {
		return b[pos.Byte:offset]
	}

	for offset < len(b) {
		nextRune, size := utf8.DecodeRune(b[offset:])
		if f(offset, nextRune) {
			// record the matched offset
			// and include the matched last rune
			endByte := offset + size
			return b[pos.Byte:endByte]
		}
		offset += size
	}

	return []byte{}
}

// isObjectItemTerminatingRune returns true if the given rune
// is considered a left terminating character for an item
// in hclsyntax.ObjectConsExpr.
func isObjectItemTerminatingRune(r rune) bool {
	return r == '\n' || r == ',' || r == '{'
}

// isNamespacedFunctionNameRune returns true if the given rune
// is a valid character of a namespaced function name.
// This includes letters, digits, dashes, underscores, and colons.
func isNamespacedFunctionNameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ':'
}

// rawObjectKey extracts raw key (as string) from KeyExpr of
// any hclsyntax.ObjectConsExpr along with the corresponding range
// and boolean indicating whether the extraction was successful.
//
// This accounts for the two common key representations (quoted and unquoted)
// and enables validation, filtering of object attributes and accurate
// calculation of edit range.
//
// It does *not* account for interpolation inside the key,
// such as { (var.key_name) = "foo" }.
func rawObjectKey(expr hcl.Expression) (string, *hcl.Range, bool) {
	if json.IsJSONExpression(expr) {
		val, diags := expr.Value(&hcl.EvalContext{})
		if diags.HasErrors() {
			return "", nil, false
		}
		if val.Type() != cty.String {
			return "", nil, false
		}

		return val.AsString(), expr.Range().Ptr(), true
	}

	// regardless of what expression it is always wrapped
	keyExpr, ok := expr.(*hclsyntax.ObjectConsKeyExpr)
	if !ok {
		return "", nil, false
	}

	switch eType := keyExpr.Wrapped.(type) {
	// most common "naked" keys
	case *hclsyntax.ScopeTraversalExpr:
		if len(eType.Traversal) != 1 {
			return "", nil, false
		}
		return eType.Traversal.RootName(), eType.Range().Ptr(), true

	// less common quoted keys
	case *hclsyntax.TemplateExpr:
		if !eType.IsStringLiteral() {
			return "", nil, false
		}

		// string literals imply exactly 1 part
		lvExpr, ok := eType.Parts[0].(*hclsyntax.LiteralValueExpr)
		if !ok {
			return "", nil, false
		}

		if lvExpr.Val.Type() != cty.String {
			return "", nil, false
		}
		return lvExpr.Val.AsString(), lvExpr.Range().Ptr(), true
	}
	return "", nil, false
}

// detailForAttribute provides additional information from the supplied attribute schema
// that can be displayed at hover, for example.
func detailForAttribute(attr *schema.AttributeSchema) string {
	var details []string
	if attr.IsRequired {
		details = append(details, "*")
	}
	friendlyName := attr.Constraint.FriendlyName()
	if friendlyName != "" {
		details = append(details, friendlyName)
	}
	return strings.Join(details, " ")
}

// attributeSchemaToCandidate returns a completion candidate for an empty expression for the attribute name.
func attributeSchemaToCandidate(name string, attr *schema.AttributeSchema, rng hcl.Range) lang.Candidate {
	var snippet string
	var triggerSuggest bool

	cData := attr.Constraint.EmptyCompletionData(1, 0)
	snippet = fmt.Sprintf("%s = %s", name, cData.Snippet)
	triggerSuggest = cData.TriggerSuggest

	return lang.Candidate{
		Label:       name,
		Detail:      detailForAttribute(attr),
		Description: attr.Description,
		Kind:        lang.AttributeCandidateKind,
		TextEdit: lang.TextEdit{
			NewText: name,
			Snippet: snippet,
			Range:   rng,
		},
		TriggerSuggest: triggerSuggest,
	}
}

// posEqual compares two positions for equality.
func posEqual(pos, other hcl.Pos) bool {
	return pos.Line == other.Line &&
		pos.Column == other.Column &&
		pos.Byte == other.Byte
}

// posToStr returns a friendly representation of a position.
func posToStr(pos hcl.Pos) string {
	return fmt.Sprintf("%d,%d", pos.Line, pos.Column)
}

// toCandidates adapts a list of hook candidates to completion candidates.
func toCandidates(in []lang.HookCandidate, editRange hcl.Range) []lang.Candidate {
	out := make([]lang.Candidate, len(in))
	for i, cd := range in {
		out[i] = lang.Candidate{
			Label:        cd.Label,
			Detail:       cd.Detail,
			Description:  cd.Description,
			Kind:         cd.Kind,
			IsDeprecated: cd.IsDeprecated,
			TextEdit: lang.TextEdit{
				NewText: cd.RawInsertText,
				Snippet: cd.RawInsertText,
				Range:   editRange,
			},
			ResolveHook: cd.ResolveHook,
			SortText:    cd.SortText,
		}
	}
	return out
}

// hoverContentForAttribute provides hover content for an attribute with the supplied name and schema.
func hoverContentForAttribute(name string, aSchema *schema.AttributeSchema) lang.MarkupContent {
	value := fmt.Sprintf("**%s** _%s_", name, detailForAttribute(aSchema))
	value += aSchema.Description.AsDetail()
	if obj, ok := aSchema.Constraint.(schema.Object); ok {
		if preview := objectAttributePreview(obj, "{", "}"); preview != "" {
			value += "\n" + preview
		}
	} else if list, ok := aSchema.Constraint.(schema.List); ok {
		if obj, ok := list.Elem.(schema.Object); ok {
			if preview := objectAttributePreview(obj, "[{", "},...]"); preview != "" {
				value += "\n" + preview
			}
		}
	}
	return lang.Markdown(value)
}

// objectAttributePreview returns a Markdown code block showing the attribute names
// and their types for the given object. If the object has more than 4 attributes,
// only the first 2 and last 2 are shown with "..." in between.
// The open and close parameters control the delimiters (e.g. "{" / "}" for objects,
// "[{" / "},...]" for lists of objects).
func objectAttributePreview(obj schema.Object, open, close string) string {
	if len(obj.Attributes) == 0 {
		return ""
	}

	names := make([]string, 0, len(obj.Attributes))
	for n := range obj.Attributes {
		names = append(names, n)
	}
	sort.Strings(names)

	const maxInline = 4
	var lines []string
	lines = append(lines, open)
	if len(names) <= maxInline {
		for _, n := range names {
			lines = append(lines, fmt.Sprintf("  %s: %s", n, obj.Attributes[n].Constraint.FriendlyName()))
		}
	} else {
		for _, n := range names[:2] {
			lines = append(lines, fmt.Sprintf("  %s: %s", n, obj.Attributes[n].Constraint.FriendlyName()))
		}
		lines = append(lines, "  ...")
		for _, n := range names[len(names)-2:] {
			lines = append(lines, fmt.Sprintf("  %s: %s", n, obj.Attributes[n].Constraint.FriendlyName()))
		}
	}
	lines = append(lines, close)

	return "```\n" + strings.Join(lines, "\n") + "\n```"
}

// candidatesFromHooks returns hook candidates at the supplied position, correctly accounting for
// incomplete expressions and parse failures.
func candidatesFromHooks(ctx decoder.CompletionContext, expr hcl.Expression, aSchema *schema.AttributeSchema, pos hcl.Pos) []lang.Candidate {
	var candidates []lang.Candidate
	con, ok := aSchema.Constraint.(schema.TypeAwareConstraint)
	if !ok {
		// Return early as we only support string values for now
		return candidates
	}
	typ, ok := con.ConstraintType()
	if !ok || typ != cty.String {
		// Return early as we only support string values for now
		return candidates
	}

	editRng := expr.Range()
	if isEmptyExpression(expr) || decoderutils.IsMultilineTemplateExpr(expr.(hclsyntax.Expression)) {
		// An empty expression or a string without a closing quote will lead to
		// an attribute expression spanning multiple lines.
		// Since text edits only support a single line, we're resetting the End
		// position here.
		editRng.End = pos
	}
	prefixRng := expr.Range()
	prefixRng.End = pos
	prefixBytes := prefixRng.SliceBytes(ctx.FileBytes(expr))
	prefix := string(prefixBytes)
	prefix = strings.TrimLeft(prefix, `"`)

	for _, hook := range aSchema.CompletionHooks {
		if completionFunc := ctx.CompletionFunc(hook.Name); completionFunc != nil {
			res, _ := completionFunc(decoder.CompletionFuncContext{
				PathContext: ctx,
				Dir:         ctx.Dir(),
				Filename:    expr.Range().Filename,
				Pos:         pos,
			}, prefix)
			candidates = append(candidates, toCandidates(res, editRng)...)
		}
	}
	return candidates
}

func boolLiteralTypeCandidates(prefix string, editRange hcl.Range) []lang.Candidate {
	var candidates []lang.Candidate
	if strings.HasPrefix("false", prefix) {
		candidates = append(candidates, lang.Candidate{
			Label:  "false",
			Detail: cty.Bool.FriendlyNameForConstraint(),
			Kind:   lang.BoolCandidateKind,
			TextEdit: lang.TextEdit{
				NewText: "false",
				Snippet: "false",
				Range:   editRange,
			},
		})
	}
	if strings.HasPrefix("true", prefix) {
		candidates = append(candidates, lang.Candidate{
			Label:  "true",
			Detail: cty.Bool.FriendlyNameForConstraint(),
			Kind:   lang.BoolCandidateKind,
			TextEdit: lang.TextEdit{
				NewText: "true",
				Snippet: "true",
				Range:   editRange,
			},
		})
	}
	return candidates
}
