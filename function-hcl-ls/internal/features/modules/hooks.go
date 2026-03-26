package modules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/zclconf/go-cty/cty"
)

type fieldType int

const (
	typeAPIVersion fieldType = iota
	typeKind
)

func (m *Modules) makeCandidates(ctx decoder.CompletionFuncContext, t fieldType, prefix string) []lang.HookCandidate {
	filterValue, err := m.findOtherValueFor(ctx, t)
	if err != nil {
		m.logger.Println(err) // and continue
	}
	dyn := m.provider(ctx.Dir)
	seen := map[string]bool{}
	for _, ak := range dyn.Keys() {
		switch t {
		case typeAPIVersion:
			if filterValue != "" && ak.Kind != filterValue {
				continue
			}
			if !strings.HasPrefix(ak.ApiVersion, prefix) {
				continue
			}
			seen[ak.ApiVersion] = true
		default:
			if filterValue != "" && ak.ApiVersion != filterValue {
				continue
			}
			if !strings.HasPrefix(ak.Kind, prefix) {
				continue
			}
			seen[ak.Kind] = true
		}
	}
	ret := make([]lang.HookCandidate, 0, len(seen))
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// log.Println("candidate:", k)
		ret = append(ret, lang.ExpressionCompletionCandidate(lang.ExpressionCandidate{
			Value: cty.StringVal(k),
		}))
	}
	return ret
}

func (m *Modules) findOtherValueFor(ctx decoder.CompletionFuncContext, t fieldType) (string, error) {
	body, ok := ctx.PathContext.HCLFileByName(ctx.Filename)
	if !ok {
		return "", fmt.Errorf("find position: no body for file %s", ctx.Filename)
	}
	bodyAttr := body.AttributeAtPos(ctx.Pos)
	if bodyAttr == nil {
		return "", fmt.Errorf("find position: nil body attr at pos %v", ctx.Pos)
	}
	fld := "apiVersion"
	if t == typeAPIVersion {
		fld = "kind"
	}
	v, _ := bodyAttr.Expr.Value(nil)
	if v.IsNull() || !v.IsKnown() {
		return "", fmt.Errorf("find position: incomplete value for %s", fld)
	}
	if !v.Type().IsObjectType() {
		return "", fmt.Errorf("find position: expected object attribute, found %v", v.Type())
	}
	obj := v.AsValueMap()
	v2 := obj[fld]
	if v2.Type() != cty.String {
		return "", fmt.Errorf("find position: expected string attribute for %s, found %v", fld, v2.Type())
	}
	return v2.AsString(), nil
}

func (m *Modules) apiVersionCompletion(ctx decoder.CompletionFuncContext, matchPrefix string) ([]lang.HookCandidate, error) {
	return m.makeCandidates(ctx, typeAPIVersion, matchPrefix), nil
}

func (m *Modules) kindCompletion(ctx decoder.CompletionFuncContext, matchPrefix string) ([]lang.HookCandidate, error) {
	return m.makeCandidates(ctx, typeKind, matchPrefix), nil
}
