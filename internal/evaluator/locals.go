package evaluator

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

// contains code to process locals.

func (e *Evaluator) processLocals(ctx *hcl.EvalContext, content *hcl.BodyContent) (*hcl.EvalContext, hcl.Diagnostics) {
	lp := newLocalsProcessor(e)
	return lp.process(ctx, content)
}

type sourceFinder interface {
	sourceCode(hcl.Range) string
}

// localsProcessor processes local declarations in a block. This is the workhorse of the evaluator.
// It computes dependencies across local variables and evaluates them in dependency order checking for circularity.
// Given an eval context, it checks whether there is any expression that relies on an unknown local or on unknown first level properties
// of the `req` and `self` namespaces and produces an error if that is the case.
// At the end of processing, it returns a child context with the locals having computed values.
// Note that "computed" does not mean "complete" - locals may have incomplete values if they refer to resource
// properties that are not yet known.
type localsProcessor struct {
	finder sourceFinder
}

// newLocalsProcessor returns a locals processor.
func newLocalsProcessor(finder sourceFinder) *localsProcessor {
	return &localsProcessor{
		finder: finder,
	}
}

type localInfo struct {
	expr hcl.Expression
	deps []string
}

// process processes all local blocks found in the supplied body contents as a single unit and returns a child
// context which has values for all locals.
func (l *localsProcessor) process(ctx *hcl.EvalContext, content *hcl.BodyContent) (*hcl.EvalContext, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var attrsList []hcl.Attributes
	for _, block := range content.Blocks {
		if block.Type == blockLocals {
			attrs, ds := block.Body.JustAttributes()
			diags = diags.Extend(ds)
			if ds.HasErrors() {
				return nil, diags
			}
			attrsList = append(attrsList, attrs)
		}
	}
	childCtx, ds := l.processLocals(ctx, attrsList)
	return childCtx, diags.Extend(ds)
}

func (l *localsProcessor) getLocalExpressions(content *hcl.BodyContent) (map[string]hcl.Expression, hcl.Diagnostics) {
	var attrsList []hcl.Attributes
	for _, block := range content.Blocks {
		if block.Type == blockLocals {
			attrs, diags := block.Body.JustAttributes()
			if diags.HasErrors() {
				return nil, diags
			}
			attrsList = append(attrsList, attrs)
		}
	}
	ret := map[string]hcl.Expression{}
	for _, attrs := range attrsList {
		for _, attr := range attrs {
			ret[attr.Name] = attr.Expr
		}
	}
	return ret, nil
}

func toErrorDiag(summary string, details string, r hcl.Range) hcl.Diagnostics {
	ret := &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   details,
		Subject:  &r,
		Context:  &r,
	}
	return []*hcl.Diagnostic{ret}
}

// processLocals returns a child context that has the values for all supplied locals evaluated in dependency order.
func (l *localsProcessor) processLocals(ctx *hcl.EvalContext, attrsList []hcl.Attributes) (*hcl.EvalContext, hcl.Diagnostics) {
	locals := map[string]*localInfo{}
	for _, attrs := range attrsList {
		for name, attr := range attrs {
			if _, ok := locals[name]; ok {
				return nil, toErrorDiag(fmt.Sprintf("local %q: duplicate local declaration", name), "", attr.Range)
			}
			if reservedWords[name] {
				return nil, toErrorDiag(fmt.Sprintf("local %q: name is reserved and cannot be used", name), "", attr.Range)
			}
			if hasVariable(ctx, name) {
				return nil, toErrorDiag("attempt to shadow local", name, attr.Range)
			}
			locals[name] = &localInfo{expr: attr.Expr}
		}
	}

	// if no locals defined, there is nothing to do.
	if len(locals) == 0 {
		return ctx, nil
	}

	diags := l.computeDeps(ctx, locals)
	if diags.HasErrors() {
		return nil, diags
	}
	childCtx := ctx.NewChild()
	childCtx.Variables = DynamicObject{}
	return childCtx, diags.Extend(l.eval(childCtx, locals))
}

// computeDeps computes the dependency maps between locals, also checking for typos in expressions in the process.
// Note that it does *not* check for circularity at this stage.
func (l *localsProcessor) computeDeps(ctx *hcl.EvalContext, locals map[string]*localInfo) hcl.Diagnostics {
	for name, info := range locals {
		deps := info.expr.Variables()
		for _, dep := range deps {
			dep = normalizeTraversal(dep)
			getText := func() string {
				return l.finder.sourceCode(dep.SourceRange())
			}
			// all references must be of the form `req.<something>`, `self.<something>` or a local variable
			// that could have any name.
			switch dep.RootName() {
			case reservedReq, reservedSelf, reservedArg:
				// no checks done here. analyzer will add some checks though.
			default:
				reference := dep.RootName()
				if _, ok := locals[reference]; ok {
					locals[name].deps = append(locals[name].deps, reference)
				} else if !hasVariable(ctx, reference) {
					return toErrorDiag("reference to non-existent local", getText(), dep.SourceRange())
				}
			}
		}
	}
	return nil
}

type evalPath struct {
	path []string
}

func (e *evalPath) push(name string) error {
	for index, v := range e.path {
		if v == name {
			//nolint:gocritic
			subPath := append(e.path[index:], name)
			return fmt.Errorf("cycle found: %s", strings.Join(subPath, " \u2192 "))
		}
	}
	e.path = append(e.path, name)
	return nil
}

func (e *evalPath) pop() {
	e.path = e.path[:len(e.path)-1]
}

type localContext struct {
	ctx       *hcl.EvalContext
	locals    map[string]*localInfo
	seen      *evalPath
	remaining map[string]bool
}

// eval evaluates all locals in dependency order.
func (l *localsProcessor) eval(ctx *hcl.EvalContext, locals map[string]*localInfo) hcl.Diagnostics {
	var diags hcl.Diagnostics

	remaining := map[string]bool{}
	for name := range locals {
		remaining[name] = true
	}
	for name := range locals {
		diags = diags.Extend(l.evalLocal(&localContext{
			ctx:       ctx,
			locals:    locals,
			seen:      &evalPath{},
			remaining: remaining,
		}, name))
	}
	return diags
}

// evalLocal evals a single local, ensuring that its dependencies are evaluated first.
func (l *localsProcessor) evalLocal(c *localContext, name string) hcl.Diagnostics {
	var diags hcl.Diagnostics
	if !c.remaining[name] { // already processed
		return nil
	}
	// check cycles
	if err := c.seen.push(name); err != nil {
		return toErrorDiag(err.Error(), "", c.locals[name].expr.Range())
	}
	defer func() {
		c.seen.pop()
		delete(c.remaining, name)
	}()

	// ensure dependencies are evaluated first
	info := c.locals[name]
	for _, dep := range info.deps {
		if c.remaining[dep] {
			diags = diags.Extend(l.evalLocal(c, dep))
		}
	}
	if diags.HasErrors() {
		return diags
	}

	// evaluate local
	// val will be an unknown value if it cannot be eval-ed
	// we ignore errors due to incomplete values.
	val, ds := info.expr.Value(c.ctx)
	// rewrite the severity of errors due to incomplete values to warnings as we'll handle them later
	diags = diags.Extend(mapDiagnosticSeverity(ds, hcl.DiagError, hcl.DiagWarning))

	// having evaluated it, update the context with the new kv.
	c.ctx.Variables[name] = val
	return diags
}
