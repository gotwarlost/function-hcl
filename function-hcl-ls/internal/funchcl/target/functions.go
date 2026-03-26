package target

import (
	ourschema "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/schema"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/typeutils"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func (l *localsCollection) schemaFromFunctionCall(e *hclsyntax.FunctionCallExpr) *schema.AttributeSchema {
	sig, ok := ourschema.StandardFunctions[e.Name]
	if !ok {
		return nil
	}
	switch sig.ReturnType {
	case cty.String, cty.Number, cty.Bool:
		return &schema.AttributeSchema{Constraint: typeutils.TypeConstraint(sig.ReturnType)}
	}
	switch e.Name {
	// for these functions the schema of the result is a schema found for any of the args.
	case "coalesce",
		"coalescelist",
		"concat",
		"distinct",
		"reverse",
		"slice",
		"sort",
		"transpose",
		"try",
		"setintersection",
		"setsubtract",
		"setunion":
		return l.findOneFrom(e.Args...)

	case "toset":
		if len(e.Args) > 0 {
			s := l.impliedSchema(e.Args[0])
			if s == nil {
				return nil
			}
			if listCons, ok := s.Constraint.(schema.List); ok {
				return &schema.AttributeSchema{Constraint: schema.Set{Elem: listCons.Elem}}
			}
		}
	// the schema is the union set of the schema from all args
	case "merge":
		return l.findUnionObjectType(e.Args...)

	// unwrap a list schema at the first arg position, if found
	case "element", "flatten", "one":
		if len(e.Args) > 0 {
			s := l.impliedSchema(e.Args[0])
			if s == nil {
				return nil
			}
			switch cons := s.Constraint.(type) {
			case schema.List:
				return &schema.AttributeSchema{Constraint: cons.Elem}
			}
		}
		return nil

	// same sig as the first argument
	case "matchkeys":
		if len(e.Args) > 0 {
			return l.impliedSchema(e.Args[0])
		}

	// list of some list
	case "chunklist":
		s := l.findOneFrom(e.Args...)
		if s == nil {
			return nil
		}
		return &schema.AttributeSchema{Constraint: schema.List{Elem: s.Constraint}}

	// infer from default, or unwrap first argument map
	case "lookup":
		if len(e.Args) > 1 {
			s := l.impliedSchema(e.Args[1])
			if s != nil {
				return s
			}
		}
		if len(e.Args) > 0 {
			s := l.impliedSchema(e.Args[0])
			switch cons := s.Constraint.(type) {
			case schema.Map:
				return &schema.AttributeSchema{Constraint: cons.Elem}
			}
		}
		return nil

	// unwrap map values
	case "values":
		if len(e.Args) > 0 {
			s := l.impliedSchema(e.Args[0])
			switch cons := s.Constraint.(type) {
			case schema.Map:
				return &schema.AttributeSchema{Constraint: schema.List{Elem: cons.Elem}}
			}
		}

	// too complex, weird rules, not worth it...
	case "setproduct",
		"tolist",
		"tomap",
		"zipmap":
		return nil
	}

	if sig.ReturnType.IsListType() {
		return &schema.AttributeSchema{
			Constraint: schema.List{
				Elem: typeutils.TypeConstraint(sig.ReturnType.ElementType()),
			},
		}
	}
	return nil
}
