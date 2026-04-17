// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package completion

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (e *expressionCompleter) completeTemplateExpr(eType *hclsyntax.TemplateExpr) []lang.Candidate {
	if eType.IsStringLiteral() {
		return nil
	}
	pos := e.pos
	for _, partExpr := range eType.Parts {
		// We overshot the position and stop
		if partExpr.Range().Start.Byte > pos.Byte {
			break
		}

		// we're not checking the end byte position here, because we don't
		// allow completion after the "}"
		if partExpr.Range().ContainsPos(pos) || partExpr.Range().End.Byte == pos.Byte {
			return e.complete(partExpr, &schema.AttributeSchema{Constraint: schema.String{}})
		}

		// trailing dot may be ignored by the parser so we attempt to recover it
		if pos.Byte-partExpr.Range().End.Byte == 1 {
			fileBytes := e.ctx.FileBytes(partExpr)
			trailingRune := fileBytes[partExpr.Range().End.Byte:pos.Byte][0]
			if trailingRune == '.' {
				return e.complete(partExpr, &schema.AttributeSchema{Constraint: schema.String{}})
			}
		}
	}
	return nil
}
