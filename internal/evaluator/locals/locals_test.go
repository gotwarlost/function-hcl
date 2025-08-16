package locals_test

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/crossplane-contrib/function-hcl/internal/evaluator/locals"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{Variables: map[string]cty.Value{}, Functions: map[string]function.Function{}}
}

// topLevelEvalContext creates a test context with typical variables for resource processing.
func topLevelEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"req": cty.ObjectVal(map[string]cty.Value{
				"composite": cty.ObjectVal(map[string]cty.Value{
					"metadata": cty.ObjectVal(map[string]cty.Value{
						"name":      cty.StringVal("my-composite"),
						"namespace": cty.StringVal("default"),
					}),
					"spec": cty.ObjectVal(map[string]cty.Value{
						"enabled":     cty.BoolVal(true),
						"environment": cty.StringVal("production"),
						"image":       cty.StringVal("nginx:latest"),
						"region":      cty.StringVal("us-west-2"),
						"replicas":    cty.NumberIntVal(3),
					}),
				}),
			}),
			"self": cty.ObjectVal(map[string]cty.Value{
				"name":     cty.StringVal("test-resource"),
				"basename": cty.StringVal("test-base"),
			}),
		},
	}
}

// parseHCLContent parses HCL content and returns body content for locals testing.
func parseHCLContent(t *testing.T, content string) *hcl.BodyContent {
	t.Helper()
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(content), "test.hcl")
	require.False(t, diags.HasErrors(), "Failed to parse HCL: %s", diags)

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "locals"},
		},
	}

	contentBody, diags := file.Body.Content(schema)
	require.False(t, diags.HasErrors(), "Failed to get content: %s", diags)
	return contentBody
}

func stdProcess(t *testing.T, hclText string) (*hcl.EvalContext, hcl.Diagnostics) {
	processor := locals.NewProcessor()
	ctx := topLevelEvalContext()
	content := parseHCLContent(t, hclText)
	return processor.Process(ctx, content)
}

func TestBasicLocals(t *testing.T) {
	hclContent := `
locals {
  simple_string = "hello"
  simple_bool   = true
}

locals {
  simple_number = 42
}
`
	processor := locals.NewProcessor()
	ctx := emptyEvalContext()
	content := parseHCLContent(t, hclContent)
	result, diags := processor.Process(ctx, content)
	require.False(t, diags.HasErrors())

	// check that locals were added to context
	assert.Equal(t, "hello", result.Variables["simple_string"].AsString())
	sn, _ := result.Variables["simple_number"].AsBigFloat().Int64()
	assert.EqualValues(t, 42, sn)
	assert.True(t, result.Variables["simple_bool"].True())
}

func TestExpressions(t *testing.T) {
	hclContent := `
locals {
  simple_string = "hello"
  simple_bool   = true
}

locals {
  simple_number = 42
}

locals {
	complex_object = { s: simple_string, b: simple_bool }
}

`
	processor := locals.NewProcessor()
	content := parseHCLContent(t, hclContent)
	result, diags := processor.Expressions(content)
	require.False(t, diags.HasErrors())
	assert.Contains(t, result, "simple_string")
	assert.Contains(t, result, "simple_bool")
	assert.Contains(t, result, "simple_number")
	assert.Contains(t, result, "complex_object")
}

func TestReqReferences(t *testing.T) {
	hclContent := `
locals {
  composite_name = req.composite.metadata.name
  replica_count  = req.composite.spec.replicas
  is_enabled     = req.composite.spec.enabled
  computed_name  = "${req.composite.metadata.name}-${self.basename}"
}
`
	result, diags := stdProcess(t, hclContent)
	require.False(t, diags.HasErrors())

	assert.Equal(t, "my-composite", result.Variables["composite_name"].AsString())
	assert.Equal(t, "my-composite-test-base", result.Variables["computed_name"].AsString())
	replicaCount, _ := result.Variables["replica_count"].AsBigFloat().Int64()
	assert.Equal(t, int64(3), replicaCount)
	assert.True(t, result.Variables["is_enabled"].True())
}

func TestLocalDependencies(t *testing.T) {
	hclContent := `
locals {
  base_name    = "myapp"
  environment  = "prod"
  full_name    = "${base_name}-${environment}"
  config_name  = "${full_name}-config"
  final_result = "${config_name}-final"
}
`
	result, diags := stdProcess(t, hclContent)
	require.False(t, diags.HasErrors())
	expected := "myapp-prod-config-final"
	assert.Equal(t, expected, result.Variables["final_result"].AsString())
}

func TestCircularDependency(t *testing.T) {
	hclContent := `
locals {
  a = b
  b = c
  c = a
}
`
	_, diags := stdProcess(t, hclContent)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), "cycle found:")
}

func TestLocalsProcessorSelfReference(t *testing.T) {
	hclContent := `
locals {
  self_ref = self_ref
}
`
	_, diags := stdProcess(t, hclContent)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), "cycle found: self_ref → self_ref")
}

func TestLocalsProcessorDuplicateLocal(t *testing.T) {
	hclContent := `
locals {
  duplicate_name = "first"
}
locals {
  duplicate_name = "second"
}
`
	_, diags := stdProcess(t, hclContent)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), `test.hcl:6,3-28: local "duplicate_name": duplicate local declaration;`)
}

func TestLocalsProcessorShadowTopLevel(t *testing.T) {
	testCases := []struct {
		name         string
		reservedWord string
	}{
		{"req", "req"},
		{"self", "self"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hclContent := fmt.Sprintf(`
locals {
  %s = "should fail"
}
`, tc.reservedWord)

			_, diags := stdProcess(t, hclContent)
			require.True(t, diags.HasErrors())
			assert.Contains(t, diags.Error(), "attempt to shadow variable")
		})
	}
}

func TestUndefinedVariable(t *testing.T) {
	hclContent := `
locals {
  undefined_ref = nonexistent_var
}
`

	_, diags := stdProcess(t, hclContent)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), `test.hcl:3,19-34: reference to non-existent variable; nonexistent_var`)
}

func TestDisallowShadowing(t *testing.T) {
	hclContent := `
locals {
  shadow = 10
}
`
	processor := locals.NewProcessor()
	ctx := emptyEvalContext()
	ctx.Variables["shadow"] = cty.NumberIntVal(0)
	content := parseHCLContent(t, hclContent)
	_, diags := processor.Process(ctx, content)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), `test.hcl:3,3-14: attempt to shadow variable; shadow`)
}

func TestComplexDependencyChain(t *testing.T) {
	hclContent := `
locals {
  level1_a = "a"
  level1_b = "b"
  level2_a = "${level1_a}-2"
  level2_b = "${level1_b}-2"
  level3   = "${level2_a}-${level2_b}"
  level4   = "${level3}-final"
}
`

	result, diags := stdProcess(t, hclContent)
	require.False(t, diags.HasErrors())
	expected := "a-2-b-2-final"
	assert.Equal(t, expected, result.Variables["level4"].AsString())
}

func TestEmptyLocals(t *testing.T) {
	hclContent := `
locals {
}
`
	processor := locals.NewProcessor()
	ctx := topLevelEvalContext()
	content := parseHCLContent(t, hclContent)
	result, diags := processor.Process(ctx, content)
	require.False(t, diags.HasErrors())
	// should return the same context since no locals were defined (optimization)
	assert.Equal(t, ctx, result, "expected same context when no locals defined")
}

func TestNoLocalsBlocks(t *testing.T) {
	hclContent := `
# No locals blocks at all
`
	processor := locals.NewProcessor()
	ctx := topLevelEvalContext()
	content := parseHCLContent(t, hclContent)
	result, diags := processor.Process(ctx, content)
	require.False(t, diags.HasErrors())
	// should return the same context since no locals were defined (optimization)
	assert.Equal(t, ctx, result, "expected same context when no locals defined")
}

func TestEdgeCases(t *testing.T) {
	t.Run("LocalWithNullValue", func(t *testing.T) {
		hclContent := `
locals {
  null_value = null
}
`
		result, diags := stdProcess(t, hclContent)
		require.False(t, diags.HasErrors())
		assert.True(t, result.Variables["null_value"].IsNull())
	})

	t.Run("LocalWithListValue", func(t *testing.T) {
		hclContent := `
locals {
  list_value = ["a", "b", "c"]
}
`

		result, diags := stdProcess(t, hclContent)
		require.False(t, diags.HasErrors())
		// hcl parses ["a", "b", "c"] as a tuple type, not a list type
		assert.True(t, result.Variables["list_value"].Type().IsTupleType())

		// verify it has 3 elements
		elements := result.Variables["list_value"].AsValueSlice()
		assert.Len(t, elements, 3)
		assert.Equal(t, "a", elements[0].AsString())
		assert.Equal(t, "b", elements[1].AsString())
		assert.Equal(t, "c", elements[2].AsString())
	})

	t.Run("LocalWithMapValue", func(t *testing.T) {
		hclContent := `
locals {
  map_value = {
    key1 = "value1"
    key2 = "value2"
  }
}
`
		result, diags := stdProcess(t, hclContent)
		require.False(t, diags.HasErrors())
		assert.True(t, result.Variables["map_value"].Type().IsObjectType())
	})
}

func TestLargeDependencyChain(t *testing.T) {
	// test performance with many interdependent locals in a true dependency chain
	hclContent := `
locals {
  step0 = "start"
`
	// generate a chain where each step depends on the previous step
	for i := 1; i <= 10; i++ {
		hclContent += fmt.Sprintf("\n  step%d = \"${step%d}-%d\"", i, i-1, i)
	}
	hclContent += "\n}"

	result, diags := stdProcess(t, hclContent)
	require.False(t, diags.HasErrors())

	// check that the final step contains the entire chain
	// should be "start-1-2-3-4-5-6-7-8-9-10"
	expected := "start"
	for i := 1; i <= 10; i++ {
		expected += fmt.Sprintf("-%d", i)
	}
	assert.Equal(t, expected, result.Variables["step10"].AsString())
	// verify intermediate steps to ensure proper dependency resolution
	assert.Equal(t, "start-1", result.Variables["step1"].AsString())
	assert.Equal(t, "start-1-2-3", result.Variables["step3"].AsString())
}

func TestEmbeddedBlock(t *testing.T) {
	hclContent := `
locals {
	bar {
		baz = true
	}
}
`
	_, diags := stdProcess(t, hclContent)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), "Blocks are not allowed here")
}

func TestEmbeddedBlockExpressions(t *testing.T) {
	hclContent := `
locals {
	bar {
		baz = true
	}
}
`
	content := parseHCLContent(t, hclContent)
	p := locals.NewProcessor()
	_, diags := p.Expressions(content)
	require.True(t, diags.HasErrors())
	assert.Contains(t, diags.Error(), "Blocks are not allowed here")
}
