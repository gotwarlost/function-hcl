// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package completion

import (
	"fmt"
	"sort"
	"strings"

	ourschema "github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/schema"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/writer"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (c *Completer) startCompletion(filename string, pos hcl.Pos) ([]lang.Candidate, error) {
	var candidates []lang.Candidate
	f, err := c.fileByName(filename)
	if err != nil {
		return candidates, err
	}
	rootBody, err := c.bodyForFileAndPos(filename, f, pos)
	if err != nil {
		return candidates, err
	}
	if dumpDebugSource {
		debugLogger.Printf("using source:\n%s\n", writer.NodeToSource(rootBody))
	}
	return c.completeBodyAtPos(rootBody, schema.NewBlockStack(), pos)
}

func (c *Completer) completeBodyAtPos(body *hclsyntax.Body, bs schema.BlockStack, pos hcl.Pos) ([]lang.Candidate, error) {
	var candidates []lang.Candidate
	filename := body.Range().Filename

	declared := make(declaredAttributes, len(body.Attributes))
	for _, attr := range body.Attributes {
		declared[attr.Name] = attr.Range()
	}

	// process position inside an attribute
	for _, attr := range body.Attributes {
		if c.isPosInsideAttrExpr(attr, pos) {
			aSchema := c.ctx.AttributeSchema(bs, attr.Name)
			// special-case: for resource bodies having an object schema, remove the status attribute
			parentBlock := bs.Peek(0).Type
			if attr.Name == "body" && (parentBlock == "resource" || parentBlock == "template") {
				aSchema = ourschema.WithoutStatus(aSchema)
			}
			return c.attrValueCompletionAtPos(attr, aSchema, pos)
		}
		if attr.NameRange.ContainsPos(pos) || posEqual(attr.NameRange.End, pos) {
			prefixRng := attr.NameRange
			prefixRng.End = pos
			return c.bodySchemaCandidates(c.ctx.BodySchema(bs), prefixRng, attr.Range(), declared), nil
		}
		if attr.EqualsRange.ContainsPos(pos) {
			return candidates, nil
		}
	}

	rng := hcl.Range{
		Filename: filename,
		Start:    pos,
		End:      pos,
	}

	processBlock := func(block *hclsyntax.Block) ([]lang.Candidate, error) {
		parentSchema := c.ctx.BodySchema(bs)
		bs.Push(block)
		childSchema := c.ctx.BodySchema(bs)
		if childSchema == nil {
			return candidates, nil
		}
		if block.TypeRange.ContainsPos(pos) {
			prefixRng := block.TypeRange
			prefixRng.End = pos
			return c.bodySchemaCandidates(parentSchema, prefixRng, block.Range(), declared), nil
		}

		for _, labelRange := range block.LabelRanges {
			if labelRange.ContainsPos(pos) || posEqual(labelRange.End, pos) {
				return candidates, nil // we've already sent allowed values for labels, nothing more to see here
			}
		}

		if isPosOutsideBody(block, pos) {
			return candidates, &positionalError{
				filename: filename,
				pos:      pos,
				msg:      fmt.Sprintf("position outside of %q body", block.Type),
			}
		}

		if block.Body != nil && block.Body.Range().ContainsPos(pos) {
			return c.completeBodyAtPos(block.Body, bs, pos)
		}
		return nil, nil
	}

	// process position inside blocks
	for _, block := range body.Blocks {
		if !block.Range().ContainsPos(pos) {
			continue
		}
		return processBlock(block)
	}

	tokenRng, err := c.nameTokenRangeAtPos(body.Range().Filename, pos)
	if err == nil {
		rng = tokenRng
	}
	return c.bodySchemaCandidates(c.ctx.BodySchema(bs), rng, rng, declared), nil
}

func (c *Completer) blockSchemaToCandidate(blockType string, block *schema.BasicBlockSchema, rng hcl.Range) lang.Candidate {
	triggerSuggest := false
	if len(block.Labels) > 0 {
		triggerSuggest = block.Labels[0].CanComplete()
	}
	return lang.Candidate{
		Label:       blockType,
		Detail:      detailForBlock(block),
		Description: block.Description,
		Kind:        lang.BlockCandidateKind,
		TextEdit: lang.TextEdit{
			NewText: blockType,
			Snippet: snippetForBlock(blockType, block),
			Range:   rng,
		},
		TriggerSuggest: triggerSuggest,
	}
}

// bodySchemaCandidates returns candidates for completion of fields inside a body or block.
func (c *Completer) bodySchemaCandidates(schema *schema.BodySchema, prefixRng, editRng hcl.Range, declared declaredAttributes) []lang.Candidate {
	prefix, _ := c.bytesFromRange(prefixRng)
	pfx := string(prefix)
	candidates := attributeCandidates(pfx, schema.Attributes, declared, editRng)
	for name, block := range schema.NestedBlocks {
		if len(pfx) > 0 && !strings.HasPrefix(name, pfx) {
			continue
		}
		candidates = append(candidates, c.blockSchemaToCandidate(name, block, editRng))
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Label < candidates[j].Label
	})
	return candidates
}

func (c *Completer) isPosInsideAttrExpr(attr *hclsyntax.Attribute, pos hcl.Pos) bool {
	if attr.Expr.Range().ContainsPos(pos) {
		return true
	}
	// edge case: near end (typically newline char)
	if attr.Expr.Range().End.Byte == pos.Byte {
		return true
	}
	// edge case: near the beginning (right after '=')
	if attr.EqualsRange.End.Byte == pos.Byte {
		return true
	}
	// edge case: end of incomplete expression with trailing '.' (which parser ignores)
	endByte := attr.Expr.Range().End.Byte
	if pos.Byte-endByte == 1 {
		suspectedDotRng := hcl.Range{
			Filename: attr.Expr.Range().Filename,
			Start:    attr.Expr.Range().End,
			End:      pos,
		}
		b, err := c.bytesFromRange(suspectedDotRng)
		if err == nil && string(b) == "." {
			return true
		}
	}
	return false
}

func (c *Completer) attrValueCompletionAtPos(attr *hclsyntax.Attribute, s *schema.AttributeSchema, pos hcl.Pos) ([]lang.Candidate, error) {
	if len(s.CompletionHooks) > 0 {
		return candidatesFromHooks(c.ctx, attr.Expr, s, pos), nil
	}
	ec := newExpressionCompleter(c.ctx, pos)
	return ec.complete(attr.Expr, s), nil
}
