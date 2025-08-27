package evaluator

import (
	"github.com/crossplane-contrib/function-hcl/internal/evaluator/hclutils"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func (e *Evaluator) processRequirement(ctx *hcl.EvalContext, block *hcl.Block) hcl.Diagnostics {
	var curDiags hcl.Diagnostics

	// get name, check duplicates
	name := block.Labels[0]
	if _, ok := e.requirements[name]; ok {
		return hclutils.ToErrorDiag("multiple requirements with name", name, block.DefRange)
	}

	// verify schema
	content, diags := block.Body.Content(requirementSchema())
	curDiags = curDiags.Extend(diags)
	if diags.HasErrors() {
		return diags
	}

	// extract a single select block
	var selBlock *hcl.Block
	for _, b := range content.Blocks {
		if b.Type == blockSelect {
			if selBlock != nil {
				return hclutils.ToErrorDiag("multiple select blocks in requirement", name, b.DefRange)
			}
			selBlock = b
		}
	}
	if selBlock == nil {
		return hclutils.ToErrorDiag("no select block in requirement", name, block.DefRange)
	}

	ctx, diags = e.processLocals(ctx, content)
	curDiags = curDiags.Extend(diags)
	if diags.HasErrors() {
		return diags
	}

	cond, ds := e.evaluateCondition(ctx, content, discardTypeRequirement, name)
	curDiags = curDiags.Extend(diags)
	if ds.HasErrors() {
		return ds
	}
	if !cond {
		return curDiags
	}

	sel, diags := e.selectBlockToSelector(ctx, selBlock)
	curDiags = curDiags.Extend(diags)
	if diags.HasErrors() {
		return diags
	}
	e.requirements[name] = sel
	return curDiags
}

func (e *Evaluator) selectBlockToSelector(ctx *hcl.EvalContext, block *hcl.Block) (*fnv1.ResourceSelector, hcl.Diagnostics) {
	var curDiags hcl.Diagnostics
	content, diags := block.Body.Content(selectSchema())
	curDiags = curDiags.Extend(diags)
	if diags.HasErrors() {
		return nil, diags
	}

	_, hasName := content.Attributes[attrMatchName]
	_, hasLabels := content.Attributes[attrMatchLabels]

	switch {
	case hasName && hasLabels:
		return nil, hclutils.ToErrorDiag("requirement selector has both matchName and matchLabels", "", block.DefRange)
	case !(hasName || hasLabels):
		return nil, hclutils.ToErrorDiag("requirement selector has neither matchName nor matchLabels", "", block.DefRange)
	}

	toStringAttr := func(attrName string) (string, hcl.Diagnostics) {
		val, diags := content.Attributes[attrName].Expr.Value(ctx)
		if diags.HasErrors() {
			return "", diags
		}
		if val.Type() != cty.String {
			return "", hclutils.ToErrorDiag("select attribute was not a string", attrName, block.DefRange)
		}
		return val.AsString(), nil
	}

	apiVersion, diags := toStringAttr(attrAPIVersion)
	curDiags = curDiags.Extend(diags)
	if diags.HasErrors() {
		return nil, diags
	}

	kind, diags := toStringAttr(attrKind)
	curDiags = curDiags.Extend(diags)
	if diags.HasErrors() {
		return nil, diags
	}

	if hasName {
		name, diags := toStringAttr(attrMatchName)
		curDiags = curDiags.Extend(diags)
		if diags.HasErrors() {
			return nil, diags
		}
		return &fnv1.ResourceSelector{
			ApiVersion: apiVersion,
			Kind:       kind,
			Match:      &fnv1.ResourceSelector_MatchName{MatchName: name},
		}, curDiags
	}

	val, diags := content.Attributes[attrMatchLabels].Expr.Value(ctx)
	curDiags = curDiags.Extend(diags)
	if diags.HasErrors() {
		return nil, diags
	}

	if !val.Type().IsObjectType() {
		return nil, hclutils.ToErrorDiag("attribute was not an object", attrMatchLabels, block.DefRange)
	}
	val2 := val.AsValueMap()
	labels := map[string]string{}
	for k, v := range val2 {
		if v.Type() != cty.String {
			return nil, hclutils.ToErrorDiag("match label was not an string", k, block.DefRange)
		}
		labels[k] = v.AsString()
	}
	return &fnv1.ResourceSelector{
		ApiVersion: apiVersion,
		Kind:       kind,
		Match:      &fnv1.ResourceSelector_MatchLabels{MatchLabels: &fnv1.MatchLabels{Labels: labels}},
	}, curDiags
}
