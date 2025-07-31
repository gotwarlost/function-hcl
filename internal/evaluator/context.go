package evaluator

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func (e *Evaluator) processContext(ctx *hcl.EvalContext, block *hcl.Block) hcl.Diagnostics {
	content, diags := block.Body.Content(contextSchema())
	if diags.HasErrors() {
		return diags
	}

	ctx, ds := e.processLocals(ctx, content)
	diags = diags.Extend(ds)
	if ds.HasErrors() {
		return ds
	}

	ex := content.Attributes[attrKey].Expr
	key, ds := ex.Value(ctx)
	diags = diags.Extend(ds)
	if ds.HasErrors() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to evaluate context key",
			Detail:   ds.Error(),
			Subject:  ptr(ex.Range()),
		})
	}
	if !key.IsWhollyKnown() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "context key is unknown",
			Subject:  ptr(ex.Range()),
		})
	}
	if key.Type() != cty.String {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("context key was not a string, got %s", key.Type().FriendlyName()),
			Subject:  ptr(ex.Range()),
		})
	}
	keyString := key.AsString()

	ex = content.Attributes[attrValue].Expr
	val, ds := ex.Value(ctx)
	if diags.HasErrors() || !val.IsWhollyKnown() {
		e.discard(DiscardItem{
			Type:        discardTypeContext,
			Reason:      discardReasonIncomplete,
			SourceRange: ex.Range().String(),
			Context:     e.messagesFromDiags(diags),
		})
		// map unknown context value errors to warnings as we'll handle them later
		return diags.Extend(mapDiagnosticSeverity(ds, hcl.DiagError, hcl.DiagWarning))
	}
	diags = diags.Extend(ds)

	goVal, err := valueToInterface(val)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "cannot convert value to interface",
			Detail:   err.Error(),
			Subject:  ptr(ex.Range()),
		})
	}
	e.contexts = append(e.contexts, Object{keyString: goVal})
	return diags
}
