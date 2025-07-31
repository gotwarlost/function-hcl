package evaluator

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSourceFinder implements sourceFinder interface for testing.
type mockSourceFinder struct {
	files map[string]string
}

func (m *mockSourceFinder) sourceCode(r hcl.Range) string {
	if content, ok := m.files[r.Filename]; ok {
		if r.End.Byte <= len(content) {
			return content[r.Start.Byte:r.End.Byte]
		}
	}
	return "[unknown source]"
}

// parseHCLContent parses HCL content and returns body content for locals testing.
func parseHCLContent(t *testing.T, content string, filename string) *hcl.BodyContent {
	t.Helper()
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(content), filename)
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

func TestLocalsProcessor_BasicLocals(t *testing.T) {
	hclContent := `
locals {
  simple_string = "hello"
  simple_number = 42
  simple_bool   = true
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "main.hcl")

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

	// check that locals were added to context
	assert.Equal(t, "hello", result.Variables["simple_string"].AsString())
	assert.True(t, result.Variables["simple_number"].AsBigFloat().IsInt())
	assert.True(t, result.Variables["simple_bool"].True())
}

func TestLocalsProcessor_ReqReferences(t *testing.T) {
	hclContent := `
locals {
  composite_name = req.composite.metadata.name
  replica_count  = req.composite.spec.replicas
  is_enabled     = req.composite.spec.enabled
  computed_name  = "${req.composite.metadata.name}-${self.basename}"
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

	assert.Equal(t, "my-composite", result.Variables["composite_name"].AsString())
	assert.Equal(t, "my-composite-test-base", result.Variables["computed_name"].AsString())

	replicaCount, _ := result.Variables["replica_count"].AsBigFloat().Int64()
	assert.Equal(t, int64(3), replicaCount)

	assert.True(t, result.Variables["is_enabled"].True())
}

func TestLocalsProcessor_LocalDependencies(t *testing.T) {
	hclContent := `
locals {
  base_name    = "myapp"
  environment  = "prod"
  full_name    = "${base_name}-${environment}"
  config_name  = "${full_name}-config"
  final_result = "${config_name}-final"
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

	expected := "myapp-prod-config-final"
	assert.Equal(t, expected, result.Variables["final_result"].AsString())
}

func TestLocalsProcessor_CircularDependency(t *testing.T) {
	hclContent := `
locals {
  a = b
  b = c
  c = a
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	_, err := processor.process(ctx, content)
	require.True(t, err.HasErrors())
	assert.Contains(t, err.Error(), "cycle found")
}

func TestLocalsProcessor_SelfReference(t *testing.T) {
	hclContent := `
locals {
  self_ref = self_ref
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	_, err := processor.process(ctx, content)
	require.True(t, err.HasErrors())
	assert.Contains(t, err.Error(), "cycle found")
}

func TestLocalsProcessor_DuplicateLocal(t *testing.T) {
	hclContent := `
locals {
  duplicate_name = "first"
}
locals {
  duplicate_name = "second"
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	_, err := processor.process(ctx, content)
	require.True(t, err.HasErrors())
	assert.Contains(t, err.Error(), `test.hcl:6,3-28: local "duplicate_name": duplicate local declaration;`)
}

func TestLocalsProcessor_ReservedWords(t *testing.T) {
	testCases := []struct {
		name         string
		reservedWord string
	}{
		{"req", "req"},
		{"self", "self"},
		{"arg", "arg"},
		{"each", "each"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hclContent := fmt.Sprintf(`
locals {
  %s = "should fail"
}
`, tc.reservedWord)

			finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
			processor := newLocalsProcessor(finder)
			ctx := createTestEvalContext()

			content := parseHCLContent(t, hclContent, "test.hcl")

			_, err := processor.process(ctx, content)
			require.True(t, err.HasErrors())
			assert.Contains(t, err.Error(), "name is reserved")
		})
	}
}

func TestLocalsProcessor_UndefinedVariable(t *testing.T) {
	hclContent := `
locals {
  undefined_ref = nonexistent_var
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	_, err := processor.process(ctx, content)
	require.True(t, err.HasErrors())
	assert.Contains(t, err.Error(), `test.hcl:3,19-34: reference to non-existent local; nonexistent_var`)
}

func TestLocalsProcessor_DisallowShadowing(t *testing.T) {
	hclContent := `
locals {
  shadow = 10
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()
	ctx.Variables["shadow"] = cty.NumberIntVal(0)
	content := parseHCLContent(t, hclContent, "test.hcl")

	_, err := processor.process(ctx, content)
	require.True(t, err.HasErrors())
	assert.Contains(t, err.Error(), `test.hcl:3,3-14: attempt to shadow local; shadow`)
}

func TestLocalsProcessor_MultipleLocalsBlocks(t *testing.T) {
	hclContent := `
locals {
  first_block_var = "first"
}

locals {
  second_block_var = "second"
  reference_first  = first_block_var
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

	assert.Equal(t, "first", result.Variables["first_block_var"].AsString())
	assert.Equal(t, "second", result.Variables["second_block_var"].AsString())
	assert.Equal(t, "first", result.Variables["reference_first"].AsString())
}

func TestLocalsProcessor_ComplexDependencyChain(t *testing.T) {
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

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

	expected := "a-2-b-2-final"
	assert.Equal(t, expected, result.Variables["level4"].AsString())
}

func TestLocalsProcessor_EmptyLocals(t *testing.T) {
	hclContent := `
locals {
}
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

	// should return the same context since no locals were defined (optimization)
	assert.Equal(t, ctx, result, "expected same context when no locals defined")
}

func TestLocalsProcessor_NoLocalsBlocks(t *testing.T) {
	hclContent := `
# No locals blocks at all
`

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	// parse with empty schema since there are no blocks
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(hclContent), "test.hcl")
	require.False(t, diags.HasErrors(), "failed to parse HCL: %s", diags)

	schema := &hcl.BodySchema{}
	content, diags := file.Body.Content(schema)
	require.False(t, diags.HasErrors(), "failed to get content: %s", diags)

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

	// should return the same context since no locals were defined
	assert.Equal(t, ctx, result, "expected same context when no locals blocks present")
}

func TestLocalsProcessor_EdgeCases(t *testing.T) {
	t.Run("LocalWithNullValue", func(t *testing.T) {
		hclContent := `
locals {
  null_value = null
}
`
		finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
		processor := newLocalsProcessor(finder)
		ctx := createTestEvalContext()

		content := parseHCLContent(t, hclContent, "test.hcl")
		result, err := processor.process(ctx, content)
		require.False(t, err.HasErrors())
		assert.True(t, result.Variables["null_value"].IsNull())
	})

	t.Run("LocalWithListValue", func(t *testing.T) {
		hclContent := `
locals {
  list_value = ["a", "b", "c"]
}
`
		finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
		processor := newLocalsProcessor(finder)
		ctx := createTestEvalContext()

		content := parseHCLContent(t, hclContent, "test.hcl")
		result, err := processor.process(ctx, content)
		require.False(t, err.HasErrors())
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
		finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
		processor := newLocalsProcessor(finder)
		ctx := createTestEvalContext()

		content := parseHCLContent(t, hclContent, "test.hcl")
		result, err := processor.process(ctx, content)
		require.False(t, err.HasErrors())
		assert.True(t, result.Variables["map_value"].Type().IsObjectType())
	})
}

func TestLocalsProcessor_LargeDependencyChain(t *testing.T) {
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

	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()

	content := parseHCLContent(t, hclContent, "test.hcl")

	result, err := processor.process(ctx, content)
	require.False(t, err.HasErrors())

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

func TestLocalsProcessor_EmbeddedBlock(t *testing.T) {
	hclContent := `
locals {
	bar {
		baz = true
	}
}
`
	finder := &mockSourceFinder{files: map[string]string{"test.hcl": hclContent}}
	processor := newLocalsProcessor(finder)
	ctx := createTestEvalContext()
	content := parseHCLContent(t, hclContent, "test.hcl")

	_, err := processor.process(ctx, content)
	require.True(t, err.HasErrors())
	assert.Contains(t, err.Error(), "Blocks are not allowed here")
}
