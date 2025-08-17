package functions_test

import (
	"strings"
	"testing"

	"github.com/crossplane-contrib/function-hcl/internal/evaluator/functions"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// parseFunctionsHCL parses HCL content containing function declarations and returns body content.
func parseFunctionsHCL(t *testing.T, content string) *hcl.BodyContent {
	t.Helper()
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(content), "test.hcl")
	require.False(t, diags.HasErrors(), "Failed to parse HCL: %s", diags)

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "function", LabelNames: []string{"name"}},
		},
	}

	contentBody, diags := file.Body.Content(schema)
	require.False(t, diags.HasErrors(), "Failed to get content: %s", diags)
	return contentBody
}

func parseExpression(t *testing.T, str string) hclsyntax.Expression {
	if strings.HasPrefix(str, "${") {
		expr, err := hclsyntax.ParseTemplate([]byte(str), "template.hcl", hcl.Pos{Line: 1, Column: 1})
		require.False(t, err.HasErrors())
		return expr
	}
	expr, err := hclsyntax.ParseExpression([]byte(str), "expr.hcl", hcl.Pos{Line: 1, Column: 1})
	require.False(t, err.HasErrors())
	return expr
}

func TestBasicFunctions(t *testing.T) {
	defs := parseFunctionsHCL(t, `
function mX {
	arg n {
		description = "input"
	}
	arg m {
		default = 2
		description = "multiplier"
	}
	body = m * n
}

function plusK {
	arg n {
		description = "input"
	}
	arg k {
		default = 1
		description = "addend"
	}
	body = n + k
}

function twoXPlus1 {
	arg n {
		description = "input"
	}
	locals {
		mult = invoke("mX", { n: n})
		add = invoke("plusK", { n: mult})
	}
	body = add
}
`)
	p := functions.NewProcessor()
	diags := p.Process(defs)
	require.False(t, diags.HasErrors())

	expr := parseExpression(t, `invoke("twoXPlus1", {n: 100})`)
	ctx := p.RootContext(nil)
	v, diags := expr.Value(ctx)
	require.False(t, diags.HasErrors())
	require.Equal(t, v.Type(), cty.Number)
	out, _ := v.AsBigFloat().Int64()
	assert.EqualValues(t, 201, out)
}

func TestRecursiveFunction(t *testing.T) {
	defs := parseFunctionsHCL(t, `
function factorial {
	arg n {}
	body = n < 1 ? 1 : n * invoke("factorial", { n: n - 1 })
}
`)

	p := functions.NewProcessor()
	diags := p.Process(defs)
	require.False(t, diags.HasErrors())
	expr := parseExpression(t, `invoke("factorial", { n: 5 })`)
	ctx := p.RootContext(nil)
	v, diags := expr.Value(ctx)
	require.False(t, diags.HasErrors())
	require.Equal(t, v.Type(), cty.Number)
	out, _ := v.AsBigFloat().Int64()
	assert.EqualValues(t, 120, out)

	// check for max depth
	expr = parseExpression(t, `invoke("factorial", { n: 101 })`)
	_, diags = expr.Value(ctx)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), "user function calls: max depth 100 exceeded")
}

func TestFunctionCallsNegative(t *testing.T) {
	defs := parseFunctionsHCL(t, `
function mX {
	arg n {
		description = "input"
	}
	arg m {
		default = 2
		description = "multiplier"
	}
	body = m * n
}
`)

	p := functions.NewProcessor()
	diags := p.Process(defs)
	require.False(t, diags.HasErrors())
	ctx := p.RootContext(nil)

	tests := []struct {
		name string
		expr string
		msg  string
	}{
		{
			name: "missing function arg",
			expr: `invoke("mX", {})`,
			msg:  `function: mX, argument "n" expected but not supplied`,
		},
		{
			name: "extra function arg",
			expr: `invoke("mX", { n: 10, m2: 4})`,
			msg:  `function: mX, invalid argument "m2"`,
		},
		{
			name: "bad call syntax 1",
			expr: `invoke("mX")`,
			msg:  `Function "invoke" expects 2 argument(s)`,
		},
		{
			name: "bad call syntax 2",
			expr: `invoke("mX", 10)`,
			msg:  `arguments to user function 'mX' is not an object, found cty.Number`,
		},
		{
			name: "bad function name",
			expr: `invoke("nX", { n: 10})`,
			msg:  `user function 'nX' not found`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := parseExpression(t, test.expr)
			_, diags := e.Value(ctx)
			require.True(t, diags.HasErrors())
			assert.Contains(t, diags.Error(), test.msg)
		})
	}
}
