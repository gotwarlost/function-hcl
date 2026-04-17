package target

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/typeutils"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

const testFile = "test.hcl"

// nilDyn is a DynamicLookup that always returns nil.
type nilDyn struct{}

func (n nilDyn) Schema(apiVersion, kind string) *schema.AttributeSchema { return nil }

// parseFiles parses HCL text and returns a file map suitable for BuildTargets.
func parseFiles(t *testing.T, text string) map[string]*hcl.File {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(text), testFile, hcl.InitialPos)
	if diags.HasErrors() {
		t.Logf("parse diagnostics: %s", diags.Error())
	}
	require.NotNil(t, f, "parsed file should not be nil")
	return map[string]*hcl.File{testFile: f}
}

// buildAndVisible builds targets and returns the visible tree at a position
// inside a block of the given type.
func buildAndVisible(t *testing.T, text string, compositeSchema *schema.AttributeSchema, blockType string) (*Targets, *Tree) {
	t.Helper()
	files := parseFiles(t, text)
	targets := BuildTargets(files, nilDyn{}, compositeSchema)
	require.NotNil(t, targets)

	body := files[testFile].Body.(*hclsyntax.Body)
	for _, b := range body.Blocks {
		if b.Type == blockType {
			pos := b.Body.SrcRange.Start
			pos.Column += 2
			tree := targets.VisibleTreeAt(b.AsHCLBlock(), testFile, pos)
			return targets, tree
		}
	}
	t.Fatalf("block %q not found", blockType)
	return nil, nil
}

// localSchema is a shortcut: build targets from text with a resource block
// and return the schema for the named local variable.
func localSchema(t *testing.T, text string) *schema.AttributeSchema {
	t.Helper()
	_, tree := buildAndVisible(t, text, nil, "resource")
	return tree.AsSchema()
}

// assertAnyExprType asserts that the schema's constraint is an exression
// with the given type.
func assertAnyExprType(t *testing.T, s *schema.AttributeSchema, expected cty.Type, msg string) {
	t.Helper()
	require.NotNil(t, s, msg)
	expectCons := typeutils.TypeConstraint(expected)
	assert.EqualValues(t, expectCons, s.Constraint)
}

// assertObjectSchema asserts the constraint is a schema.Object and returns it.
func assertObjectSchema(t *testing.T, s *schema.AttributeSchema, msg string) schema.Object {
	t.Helper()
	require.NotNil(t, s, msg)
	oc, ok := s.Constraint.(schema.Object)
	require.True(t, ok, "%s: expected schema.Object, got %T", msg, s.Constraint)
	return oc
}

// assertListSchema asserts the constraint is a schema.List and returns it.
func assertListSchema(t *testing.T, s *schema.AttributeSchema, msg string) schema.List {
	t.Helper()
	require.NotNil(t, s, msg)
	lc, ok := s.Constraint.(schema.List)
	require.True(t, ok, "%s: expected schema.List, got %T", msg, s.Constraint)
	return lc
}

// rootNames returns the names of all root nodes in a tree.
func rootNames(tree *Tree) []string {
	var names []string
	for _, n := range tree.Roots() {
		names = append(names, n.Name)
	}
	return names
}

// withResource wraps locals text inside a resource block so buildAndVisible works.
func withResource(localsText string) string {
	return localsText + `
resource vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
}
