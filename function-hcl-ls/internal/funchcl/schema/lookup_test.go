// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type mockDynamicLookup struct {
	schemas map[string]map[string]*schema.AttributeSchema // apiVersion -> kind -> schema
}

func (m *mockDynamicLookup) Schema(apiVersion, kind string) *schema.AttributeSchema {
	if m.schemas == nil {
		return nil
	}
	kinds, ok := m.schemas[apiVersion]
	if !ok {
		return nil
	}
	return kinds[kind]
}

type mockCompositeSchemaLookup struct {
	mockDynamicLookup
	compositeSchema *schema.AttributeSchema
}

func (m *mockCompositeSchemaLookup) CompositeSchema() *schema.AttributeSchema {
	return m.compositeSchema
}

type mockLocalsAttributeLookup struct {
	mockDynamicLookup
	localSchemas map[string]*schema.AttributeSchema
}

func (m *mockLocalsAttributeLookup) LocalSchema(name string) *schema.AttributeSchema {
	if m.localSchemas == nil {
		return nil
	}
	return m.localSchemas[name]
}

// TestLabelSchema_EmptyStack tests behavior with empty block stack
func TestLabelSchema_EmptyStack(t *testing.T) {
	l := &lookup{dyn: &mockDynamicLookup{}}
	bs := schema.NewBlockStack()

	// This tests a potential bug: what happens with empty stack?
	// The code calls bs.Peek(0) and bs.Peek(1) without checking
	result := l.LabelSchema(bs)

	// Should not panic and return anyLabelSchema
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// TestLabelSchema_SingleElementStack tests with only one block in stack
func TestLabelSchema_SingleElementStack(t *testing.T) {
	l := &lookup{dyn: &mockDynamicLookup{}}
	bs := schema.NewBlockStack()
	bs.Push(&hclsyntax.Block{Type: "resource"})

	// BUG: This calls Peek(1) when there's only one element
	// This could panic or return nil depending on BlockStack implementation
	result := l.LabelSchema(bs)

	assert.NotNil(t, result)
	// When parent is empty string (from Peek(1) on single element stack),
	// it should look up std[""] and find resource in nested blocks
}

// TestLabelSchema_ValidNesting tests normal case with proper nesting
func TestLabelSchema_ValidNesting(t *testing.T) {
	l := &lookup{dyn: &mockDynamicLookup{}}

	tests := []struct {
		name           string
		parentType     string
		childType      string
		expectedLabels int
	}{
		{
			name:           "resource under root",
			parentType:     "",
			childType:      "resource",
			expectedLabels: 1, // has name label
		},
		{
			name:           "locals under resource",
			parentType:     "resource",
			childType:      "locals",
			expectedLabels: 0, // no labels
		},
		{
			name:           "arg under function",
			parentType:     "function",
			childType:      "arg",
			expectedLabels: 1, // has name label
		},
		{
			name:           "unknown child",
			parentType:     "resource",
			childType:      "unknown",
			expectedLabels: 0, // returns anyLabelSchema
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := schema.NewBlockStack()
			bs.Push(&hclsyntax.Block{Type: tt.parentType})
			bs.Push(&hclsyntax.Block{Type: tt.childType})

			result := l.LabelSchema(bs)
			assert.Len(t, result, tt.expectedLabels)
		})
	}
}

// TestBodySchema tests BodySchema lookup
func TestBodySchema(t *testing.T) {
	l := &lookup{dyn: &mockDynamicLookup{}}

	tests := []struct {
		name      string
		blockType string
		wantNil   bool
	}{
		{"root block", "", false},
		{"resource block", "resource", false},
		{"locals block", "locals", false},
		{"unknown block", "nonexistent", false}, // returns anyBody
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := schema.NewBlockStack()
			bs.Push(&hclsyntax.Block{Type: tt.blockType})

			result := l.BodySchema(bs)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

// TestCompositeStatusSchema tests compositeStatusSchema method
func TestCompositeStatusSchema(t *testing.T) {
	tests := []struct {
		name    string
		dyn     DynamicLookup
		wantNil bool
	}{
		{
			name:    "DynamicLookup doesn't implement CompositeSchemaLookup",
			dyn:     &mockDynamicLookup{},
			wantNil: true,
		},
		{
			name: "CompositeSchema returns nil",
			dyn: &mockCompositeSchemaLookup{
				compositeSchema: nil,
			},
			wantNil: true,
		},
		{
			name: "CompositeSchema constraint is not Object",
			dyn: &mockCompositeSchemaLookup{
				compositeSchema: &schema.AttributeSchema{
					Constraint: schema.String{},
				},
			},
			wantNil: true,
		},
		{
			name: "CompositeSchema has no status attribute",
			dyn: &mockCompositeSchemaLookup{
				compositeSchema: &schema.AttributeSchema{
					Constraint: schema.Object{
						Attributes: map[string]*schema.AttributeSchema{
							"spec": {IsOptional: true},
						},
					},
				},
			},
			wantNil: true, // BUG: Returns nil for missing status
		},
		{
			name: "CompositeSchema has status attribute",
			dyn: &mockCompositeSchemaLookup{
				compositeSchema: &schema.AttributeSchema{
					Constraint: schema.Object{
						Attributes: map[string]*schema.AttributeSchema{
							"status": {IsOptional: true},
						},
					},
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &lookup{dyn: tt.dyn}
			result := l.compositeStatusSchema()

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

// TestAttributeSchema_Locals tests attribute schema for locals block
func TestAttributeSchema_Locals(t *testing.T) {
	l := &lookup{dyn: &mockDynamicLookup{}}
	bs := schema.NewBlockStack()
	bs.Push(&hclsyntax.Block{
		Type: "locals",
		Body: &hclsyntax.Body{
			Attributes: map[string]*hclsyntax.Attribute{},
		},
	})

	// Any attribute in locals should return anyAttribute
	result := l.AttributeSchema(bs, "anything")
	assert.NotNil(t, result)
	assert.True(t, result.IsOptional)
}

// TestAttributeSchema_CompositeBody tests composite body attribute
func TestAttributeSchema_CompositeBody(t *testing.T) {
	tests := []struct {
		name         string
		labels       []string
		dyn          DynamicLookup
		expectedType string // "any", "status", "connection"
	}{
		{
			name:         "no labels",
			labels:       []string{},
			dyn:          &mockDynamicLookup{},
			expectedType: "any",
		},
		{
			name:         "status label with no composite schema",
			labels:       []string{"status"},
			dyn:          &mockDynamicLookup{},
			expectedType: "object", // returns anyRequiredObjAttribute
		},
		{
			name:   "status label with composite schema",
			labels: []string{"status"},
			dyn: &mockCompositeSchemaLookup{
				compositeSchema: &schema.AttributeSchema{
					Constraint: schema.Object{
						Attributes: map[string]*schema.AttributeSchema{
							"status": {IsOptional: true},
						},
					},
				},
			},
			expectedType: "status",
		},
		{
			name:         "connection label",
			labels:       []string{"connection"},
			dyn:          &mockDynamicLookup{},
			expectedType: "connection",
		},
		{
			name:         "unknown label",
			labels:       []string{"unknown"},
			dyn:          &mockDynamicLookup{},
			expectedType: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &lookup{dyn: tt.dyn}
			bs := schema.NewBlockStack()
			bs.Push(&hclsyntax.Block{
				Type:   "composite",
				Labels: tt.labels,
				Body: &hclsyntax.Body{
					Attributes: map[string]*hclsyntax.Attribute{},
				},
			})

			result := l.AttributeSchema(bs, "body")
			require.NotNil(t, result)

			switch tt.expectedType {
			case "any":
				assert.True(t, result.IsOptional)
			case "object":
				assert.True(t, result.IsRequired)
			case "connection":
				assert.True(t, result.IsRequired)
				_, ok := result.Constraint.(schema.Map)
				assert.True(t, ok, "connection should be a map")
			case "status":
				assert.True(t, result.IsOptional)
			}
		})
	}
}

// TestAttributeSchema_ResourceBody tests resource/template body attribute
func TestAttributeSchema_ResourceBody(t *testing.T) {
	mockSchema := &schema.AttributeSchema{
		Constraint: schema.Object{
			Attributes: map[string]*schema.AttributeSchema{
				"spec": {IsOptional: true},
			},
		},
	}

	tests := []struct {
		name       string
		blockType  string
		bodySource string
		dyn        DynamicLookup
		hasDynamic bool
	}{
		{
			name:      "resource without body attribute",
			blockType: "resource",
			bodySource: `resource "test" {
			}`,
			dyn:        &mockDynamicLookup{},
			hasDynamic: false,
		},
		{
			name:      "resource with valid apiVersion/kind",
			blockType: "resource",
			bodySource: `resource "test" {
				body = {
					apiVersion = "v1"
					kind = "Pod"
				}
			}`,
			dyn: &mockDynamicLookup{
				schemas: map[string]map[string]*schema.AttributeSchema{
					"v1": {
						"Pod": mockSchema,
					},
				},
			},
			hasDynamic: true,
		},
		{
			name:      "template with valid apiVersion/kind",
			blockType: "template",
			bodySource: `template {
				body = {
					apiVersion = "v1"
					kind = "Service"
				}
			}`,
			dyn: &mockDynamicLookup{
				schemas: map[string]map[string]*schema.AttributeSchema{
					"v1": {
						"Service": mockSchema,
					},
				},
			},
			hasDynamic: true,
		},
		{
			name:      "resource with missing kind",
			blockType: "resource",
			bodySource: `resource "test" {
				body = {
					apiVersion = "v1"
				}
			}`,
			dyn:        &mockDynamicLookup{},
			hasDynamic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.bodySource), "test.hcl", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "parse should succeed")

			var block *hclsyntax.Block
			for _, b := range file.Body.(*hclsyntax.Body).Blocks {
				if b.Type == tt.blockType {
					block = b
					break
				}
			}
			require.NotNil(t, block, "should find block")

			l := &lookup{dyn: tt.dyn}
			bs := schema.NewBlockStack()
			bs.Push(block)

			result := l.AttributeSchema(bs, "body")
			assert.NotNil(t, result)

			if tt.hasDynamic {
				// Should return the dynamic schema
				obj, ok := result.Constraint.(schema.Object)
				assert.True(t, ok, "should be object constraint")
				assert.Contains(t, obj.Attributes, "spec")
			}
		})
	}
}

// TestAttributeSchema_DefaultCase tests default attribute lookup
func TestAttributeSchema_DefaultCase(t *testing.T) {
	l := &lookup{dyn: &mockDynamicLookup{}}

	tests := []struct {
		name      string
		blockType string
		attrName  string
		wantNil   bool
	}{
		{
			name:      "known block, known attribute",
			blockType: "resource",
			attrName:  "condition",
			wantNil:   false,
		},
		{
			name:      "known block, unknown attribute",
			blockType: "resource",
			attrName:  "nonexistent",
			wantNil:   false, // returns anyAttribute
		},
		{
			name:      "unknown block, any attribute",
			blockType: "unknown",
			attrName:  "anything",
			wantNil:   false, // returns anyAttribute
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := schema.NewBlockStack()
			bs.Push(&hclsyntax.Block{
				Type: tt.blockType,
				Body: &hclsyntax.Body{
					Attributes: map[string]*hclsyntax.Attribute{},
				},
			})

			result := l.AttributeSchema(bs, tt.attrName)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

// TestImpliedAttributeSchema tests ImpliedAttributeSchema
func TestImpliedAttributeSchema(t *testing.T) {
	tests := []struct {
		name      string
		blockType string
		attrName  string
		dyn       DynamicLookup
		wantNil   bool
	}{
		{
			name:      "non-locals block",
			blockType: "resource",
			attrName:  "foo",
			dyn:       &mockDynamicLookup{},
			wantNil:   true,
		},
		{
			name:      "locals block without LocalsAttributeLookup",
			blockType: "locals",
			attrName:  "foo",
			dyn:       &mockDynamicLookup{},
			wantNil:   true,
		},
		{
			name:      "locals block with LocalsAttributeLookup, no schema",
			blockType: "locals",
			attrName:  "foo",
			dyn: &mockLocalsAttributeLookup{
				localSchemas: map[string]*schema.AttributeSchema{},
			},
			wantNil: true,
		},
		{
			name:      "locals block with LocalsAttributeLookup, has schema",
			blockType: "locals",
			attrName:  "foo",
			dyn: &mockLocalsAttributeLookup{
				localSchemas: map[string]*schema.AttributeSchema{
					"foo": {IsOptional: true},
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &lookup{dyn: tt.dyn}
			bs := schema.NewBlockStack()
			bs.Push(&hclsyntax.Block{Type: tt.blockType})

			result := l.ImpliedAttributeSchema(bs, tt.attrName)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

// TestFunctions tests Functions method
func TestFunctions(t *testing.T) {
	l := &lookup{dyn: &mockDynamicLookup{}}

	funcs := l.Functions()
	assert.NotNil(t, funcs)
	assert.Greater(t, len(funcs), 0, "should have standard functions")

	// Verify some standard functions exist
	assert.Contains(t, funcs, "base64encode")
	assert.Contains(t, funcs, "jsonencode")
}

// TestDependentSchema tests the dependentSchema function
func TestDependentSchema(t *testing.T) {
	mockSchema := &schema.AttributeSchema{
		Constraint: schema.Object{
			Attributes: map[string]*schema.AttributeSchema{
				"spec": {IsOptional: true},
			},
		},
	}

	tests := []struct {
		name      string
		source    string
		dyn       DynamicLookup
		expectOK  bool
		expectNil bool
	}{
		{
			name: "no body attribute",
			source: `resource "test" {
			}`,
			dyn:      &mockDynamicLookup{},
			expectOK: false,
		},
		{
			name: "body with valid string literals",
			source: `resource "test" {
				body = {
					apiVersion = "v1"
					kind = "Pod"
				}
			}`,
			dyn: &mockDynamicLookup{
				schemas: map[string]map[string]*schema.AttributeSchema{
					"v1": {"Pod": mockSchema},
				},
			},
			expectOK:  true,
			expectNil: false,
		},
		{
			name: "body with missing apiVersion",
			source: `resource "test" {
				body = {
					kind = "Pod"
				}
			}`,
			dyn:      &mockDynamicLookup{},
			expectOK: false,
		},
		{
			name: "body with missing kind",
			source: `resource "test" {
				body = {
					apiVersion = "v1"
				}
			}`,
			dyn:      &mockDynamicLookup{},
			expectOK: false,
		},
		{
			name: "body with non-string apiVersion",
			source: `resource "test" {
				body = {
					apiVersion = 123
					kind = "Pod"
				}
			}`,
			dyn:      &mockDynamicLookup{},
			expectOK: false,
		},
		{
			name: "body with non-string kind",
			source: `resource "test" {
				body = {
					apiVersion = "v1"
					kind = true
				}
			}`,
			dyn:      &mockDynamicLookup{},
			expectOK: false,
		},
		{
			name: "no matching schema in DynamicLookup",
			source: `resource "test" {
				body = {
					apiVersion = "v1"
					kind = "Unknown"
				}
			}`,
			dyn:      &mockDynamicLookup{},
			expectOK: false,
		},
		{
			name: "body is not an object",
			source: `resource "test" {
				body = []
			}`,
			dyn:      &mockDynamicLookup{},
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.source), "test.hcl", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "parse should succeed")

			var block *hclsyntax.Block
			for _, b := range file.Body.(*hclsyntax.Body).Blocks {
				if b.Type == "resource" {
					block = b
					break
				}
			}
			require.NotNil(t, block, "should find resource block")

			result, ok := dependentSchema(tt.dyn, block)

			assert.Equal(t, tt.expectOK, ok, "ok value should match")
			if tt.expectOK {
				if tt.expectNil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

// TestDependentSchema_ListBody tests fix for critical panic bug
// BUG #1 (FIXED): dependentSchema was calling AsValueMap() without type checking
// This would panic if body was a list instead of a map/object
func TestDependentSchema_ListBody(t *testing.T) {
	source := `resource "test" {
		body = ["item1", "item2"]
	}`

	file, diags := hclsyntax.ParseConfig([]byte(source), "test.hcl", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	var block *hclsyntax.Block
	for _, b := range file.Body.(*hclsyntax.Body).Blocks {
		if b.Type == "resource" {
			block = b
			break
		}
	}
	require.NotNil(t, block)

	// ✅ FIXED: This now returns false gracefully instead of panicking
	_, ok := dependentSchema(&mockDynamicLookup{}, block)
	assert.False(t, ok, "should return false for list body without panicking")
}

// TestDependentSchema_Variables tests with variables
func TestDependentSchema_Variables(t *testing.T) {
	// When body uses variables, Value() will return cty.DynamicVal
	source := `resource "test" {
		body = {
			apiVersion = var.apiVersion
			kind = var.kind
		}
	}`

	file, diags := hclsyntax.ParseConfig([]byte(source), "test.hcl", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	var block *hclsyntax.Block
	for _, b := range file.Body.(*hclsyntax.Body).Blocks {
		if b.Type == "resource" {
			block = b
			break
		}
	}
	require.NotNil(t, block)

	_, ok := dependentSchema(&mockDynamicLookup{}, block)
	assert.False(t, ok, "should return false when values are variables")
}

// TestNew verifies New constructor
func TestNew(t *testing.T) {
	dyn := &mockDynamicLookup{}
	result := New(dyn)

	assert.NotNil(t, result)
	assert.Implements(t, (*schema.Lookup)(nil), result)

	// Verify it's actually a lookup with the right dyn
	l, ok := result.(*lookup)
	require.True(t, ok)
	assert.Equal(t, dyn, l.dyn)
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("nil DynamicLookup", func(t *testing.T) {
		l := &lookup{dyn: nil}

		// Should not panic
		assert.NotPanics(t, func() {
			bs := schema.NewBlockStack()
			bs.Push(&hclsyntax.Block{Type: "resource", Body: &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{}}})
			_ = l.AttributeSchema(bs, "body")
		})
	})

	t.Run("empty block type", func(t *testing.T) {
		l := &lookup{dyn: &mockDynamicLookup{}}
		bs := schema.NewBlockStack()
		bs.Push(&hclsyntax.Block{Type: "", Body: &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{}}})

		result := l.BodySchema(bs)
		assert.NotNil(t, result)
	})

	t.Run("composite with multiple labels", func(t *testing.T) {
		l := &lookup{dyn: &mockDynamicLookup{}}
		bs := schema.NewBlockStack()
		bs.Push(&hclsyntax.Block{
			Type:   "composite",
			Labels: []string{"status", "extra"},
			Body:   &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{}},
		})

		// Should use first label only
		result := l.AttributeSchema(bs, "body")
		assert.NotNil(t, result)
		assert.True(t, result.IsRequired)
	})
}
