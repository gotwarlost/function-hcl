// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package completion

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (e *expressionCompleter) completeFunction(expr hclsyntax.Expression, as *schema.AttributeSchema) []lang.Candidate {
	pos := e.pos
	if isEmptyExpression(expr) {
		editRange := hcl.Range{
			Filename: expr.Range().Filename,
			Start:    pos,
			End:      pos,
		}
		return e.matchingFunctions("", editRange, as)
	}

	switch eType := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		if len(eType.Traversal) > 1 {
			// we assume that function names cannot contain dots
			return nil
		}

		prefixLen := pos.Byte - eType.Traversal.SourceRange().Start.Byte
		rootName := eType.Traversal.RootName()

		// There can be a single segment with trailing dot which cannot
		// be a function anymore as functions cannot contain dots.
		if prefixLen < 0 || prefixLen > len(rootName) {
			return nil
		}

		prefix := rootName[0:prefixLen]
		return e.matchingFunctions(prefix, eType.Range(), as)

	case *hclsyntax.ExprSyntaxError:
		// Note: this range can range up until the end of the file in case of invalid config.
		// The HCL parser does support the :: namespace syntax for function names (since
		// hashicorp/hcl#639), but it can still produce ExprSyntaxError when the user is
		// in the middle of typing a namespaced function name and the expression is not yet
		// syntactically complete (e.g. "provider::" with nothing after it, or missing parens).
		if eType.SrcRange.ContainsPos(pos) {
			// recover bytes around the cursor to check whether the user is partially
			// typing a namespaced function name
			fileBytes := e.ctx.FileBytes(eType)

			recoveredPrefixBytes := recoverLeftBytes(fileBytes, pos, func(offset int, r rune) bool {
				return !isNamespacedFunctionNameRune(r)
			})
			// recoveredPrefixBytes also contains the rune before the function name, so we need to trim it
			_, lengthFirstRune := utf8.DecodeRune(recoveredPrefixBytes)
			recoveredPrefixBytes = recoveredPrefixBytes[lengthFirstRune:]

			recoveredSuffixBytes := recoverRightBytes(fileBytes, pos, func(offset int, r rune) bool {
				return !isNamespacedFunctionNameRune(r) && r != '('
			})
			// recoveredSuffixBytes also contains the rune after the function name, so we need to trim it
			_, lengthLastRune := utf8.DecodeLastRune(recoveredSuffixBytes)
			recoveredSuffixBytes = recoveredSuffixBytes[:len(recoveredSuffixBytes)-lengthLastRune]

			recoveredIdentifier := append(recoveredPrefixBytes, recoveredSuffixBytes...)

			// check if our recovered identifier contains "::"
			// Why two colons? For no colons the parser would return a traversal expression
			// and a single colon will apparently be treated as a traversal and a partial object expression
			// (refer to this follow-up issue for more on that case: https://github.com/hashicorp/vscode-terraform/issues/1697)
			if bytes.Contains(recoveredIdentifier, []byte("::")) {
				editRange := hcl.Range{
					Filename: expr.Range().Filename,
					Start: hcl.Pos{
						Line:   pos.Line, // we don't recover newlines, so we can keep the original line number
						Byte:   pos.Byte - len(recoveredPrefixBytes),
						Column: pos.Column - len(recoveredPrefixBytes),
					},
					End: hcl.Pos{
						Line:   pos.Line,
						Byte:   pos.Byte + len(recoveredSuffixBytes),
						Column: pos.Column + len(recoveredSuffixBytes),
					},
				}
				return e.matchingFunctions(string(recoveredPrefixBytes), editRange, as)
			}
		}
		return nil
	}
	return nil
}

func (e *expressionCompleter) matchingFunctions(prefix string, editRange hcl.Range, as *schema.AttributeSchema) []lang.Candidate {
	var candidates []lang.Candidate

	// DODGY: we are completing literal true and false here instead of in a sance place :(
	if _, ok := as.Constraint.(schema.Bool); ok {
		candidates = boolLiteralTypeCandidates(prefix, editRange)
	}
	for name, f := range e.ctx.Functions() {
		/*
			the terraform language server makes special checks to ensure that the return type of the function
			matches the LHS. However, consider:
				foo_boolean = cidrhost(..) == "something"
			In this case, even though the cidrhost function doesn't return a boolean, it can still be used as part of an
			expression for a boolean LHS. In fact, literally any function can be used given conditional expressions,
			logical comparisons etc. So we don't bother to check the return type and send _all_ functions that match
			the prefix.
		*/
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		candidates = append(candidates, lang.Candidate{
			Label:       name,
			Detail:      fmt.Sprintf("%s(%s) %s", name, parameterNamesAsString(f), f.ReturnType.FriendlyName()),
			Kind:        lang.FunctionCandidateKind,
			Description: lang.Markdown(f.Description),
			TextEdit: lang.TextEdit{
				NewText: fmt.Sprintf("%s()", name),
				Snippet: fmt.Sprintf("%s(${0})", name),
				Range:   editRange,
			},
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Label < candidates[j].Label
	})
	return candidates
}
