package target

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// --- Node.AsSchema ---

func TestNodeAsSchema(t *testing.T) {
	t.Run("non-container returns Schema directly", func(t *testing.T) {
		s := &schema.AttributeSchema{Constraint: schema.String{}}
		n := &Node{Name: "leaf", IsContainer: false, Schema: s}
		assert.Equal(t, s, n.AsSchema())
	})

	t.Run("non-container with nil Schema returns nil", func(t *testing.T) {
		n := &Node{Name: "leaf", IsContainer: false, Schema: nil}
		assert.Nil(t, n.AsSchema())
	})

	t.Run("container builds Object from children", func(t *testing.T) {
		child1 := &Node{Name: "a", Schema: &schema.AttributeSchema{Constraint: schema.String{}}}
		child2 := &Node{Name: "b", Schema: &schema.AttributeSchema{Constraint: schema.Number{}}}
		n := &Node{Name: "parent", IsContainer: true, Children: []*Node{child1, child2}}

		result := n.AsSchema()
		require.NotNil(t, result)
		obj := assertObjectSchema(t, result, "container")
		assert.Equal(t, "parent", obj.Name)
		assert.Contains(t, obj.Attributes, "a")
		assert.Contains(t, obj.Attributes, "b")
		assertAnyExprType(t, obj.Attributes["a"], cty.String, "child a")
		assertAnyExprType(t, obj.Attributes["b"], cty.Number, "child b")
	})

	t.Run("container with no children produces empty Object", func(t *testing.T) {
		n := &Node{Name: "empty", IsContainer: true, Children: nil}
		result := n.AsSchema()
		obj := assertObjectSchema(t, result, "empty container")
		assert.Empty(t, obj.Attributes)
	})

	t.Run("nested containers", func(t *testing.T) {
		leaf := &Node{Name: "val", Schema: &schema.AttributeSchema{Constraint: schema.Bool{}}}
		inner := &Node{Name: "inner", IsContainer: true, Children: []*Node{leaf}}
		outer := &Node{Name: "outer", IsContainer: true, Children: []*Node{inner}}

		result := outer.AsSchema()
		outerObj := assertObjectSchema(t, result, "outer")
		innerSchema := outerObj.Attributes["inner"]
		innerObj := assertObjectSchema(t, innerSchema, "inner")
		assertAnyExprType(t, innerObj.Attributes["val"], cty.Bool, "val")
	})
}

// --- Tree ---

func TestTree(t *testing.T) {
	t.Run("newTree with roots", func(t *testing.T) {
		n1 := &Node{Name: "x"}
		n2 := &Node{Name: "y"}
		tree := newTree(n1, n2)
		roots := tree.Roots()
		require.Len(t, roots, 2)
		assert.Equal(t, "x", roots[0].Name)
		assert.Equal(t, "y", roots[1].Name)
	})

	t.Run("newTree with no roots", func(t *testing.T) {
		tree := newTree()
		assert.Empty(t, tree.Roots())
	})

	t.Run("AsSchema wraps roots in Object", func(t *testing.T) {
		n1 := &Node{Name: "a", Schema: &schema.AttributeSchema{Constraint: schema.String{}}}
		n2 := &Node{Name: "b", IsContainer: true, Children: []*Node{
			{Name: "c", Schema: &schema.AttributeSchema{Constraint: schema.Number{}}},
		}}
		tree := newTree(n1, n2)

		s := tree.AsSchema()
		require.NotNil(t, s)
		obj := assertObjectSchema(t, s, "tree schema")
		assert.Contains(t, obj.Attributes, "a")
		assert.Contains(t, obj.Attributes, "b")
		// b is a container, so its schema should be an Object with "c"
		bObj := assertObjectSchema(t, obj.Attributes["b"], "b")
		assert.Contains(t, bObj.Attributes, "c")
	})

	t.Run("AsSchema empty tree", func(t *testing.T) {
		tree := newTree()
		s := tree.AsSchema()
		obj := assertObjectSchema(t, s, "empty tree")
		assert.Empty(t, obj.Attributes)
	})
}

// --- ReferenceMap.FindDefinitionFromReference ---

func TestFindDefinitionFromReference(t *testing.T) {
	defRange := hcl.Range{
		Filename: "test.hcl",
		Start:    hcl.Pos{Line: 2, Column: 3, Byte: 10},
		End:      hcl.Pos{Line: 2, Column: 9, Byte: 16},
	}
	refRange := hcl.Range{
		Filename: "test.hcl",
		Start:    hcl.Pos{Line: 10, Column: 5, Byte: 100},
		End:      hcl.Pos{Line: 10, Column: 11, Byte: 106},
	}

	t.Run("finds definition for reference", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{defRange: {refRange}},
			RefsToDef: RefsToDef{refRange: defRange},
		}
		result := rm.FindDefinitionFromReference("test.hcl", hcl.Pos{Line: 10, Column: 7, Byte: 102})
		require.NotNil(t, result)
		assert.Equal(t, defRange, *result)
	})

	t.Run("returns nil for wrong file", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{defRange: {refRange}},
			RefsToDef: RefsToDef{refRange: defRange},
		}
		result := rm.FindDefinitionFromReference("other.hcl", hcl.Pos{Line: 10, Column: 7})
		assert.Nil(t, result)
	})

	t.Run("returns nil for position outside reference", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{defRange: {refRange}},
			RefsToDef: RefsToDef{refRange: defRange},
		}
		result := rm.FindDefinitionFromReference("test.hcl", hcl.Pos{Line: 5, Column: 1})
		assert.Nil(t, result)
	})

	t.Run("pseudo-range returns nil", func(t *testing.T) {
		pseudoDef := hcl.Range{
			Filename: "", // pseudo-range: no real definition
			Start:    hcl.Pos{Line: 0, Column: 0},
			End:      hcl.Pos{Line: 0, Column: 0},
		}
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{},
			RefsToDef: RefsToDef{refRange: pseudoDef},
		}
		result := rm.FindDefinitionFromReference("test.hcl", hcl.Pos{Line: 10, Column: 7, Byte: 102})
		assert.Nil(t, result)
	})

	t.Run("cursor on definition itself", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{defRange: {refRange}},
			RefsToDef: RefsToDef{refRange: defRange},
		}
		// Position is inside defRange but not inside any refRange
		result := rm.FindDefinitionFromReference("test.hcl", hcl.Pos{Line: 2, Column: 5, Byte: 12})
		require.NotNil(t, result)
		assert.Equal(t, defRange, *result)
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{},
			RefsToDef: RefsToDef{},
		}
		result := rm.FindDefinitionFromReference("test.hcl", hcl.Pos{Line: 1, Column: 1})
		assert.Nil(t, result)
	})
}

// --- ReferenceMap.FindReferencesFromDefinition ---

func TestFindReferencesFromDefinition(t *testing.T) {
	defRange := hcl.Range{
		Filename: "test.hcl",
		Start:    hcl.Pos{Line: 2, Column: 3, Byte: 10},
		End:      hcl.Pos{Line: 2, Column: 9, Byte: 16},
	}
	ref1 := hcl.Range{
		Filename: "test.hcl",
		Start:    hcl.Pos{Line: 10, Column: 5, Byte: 100},
		End:      hcl.Pos{Line: 10, Column: 11, Byte: 106},
	}
	ref2 := hcl.Range{
		Filename: "test.hcl",
		Start:    hcl.Pos{Line: 15, Column: 5, Byte: 200},
		End:      hcl.Pos{Line: 15, Column: 11, Byte: 206},
	}

	t.Run("finds references for definition", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{defRange: {ref1, ref2}},
			RefsToDef: RefsToDef{},
		}
		result := rm.FindReferencesFromDefinition("test.hcl", hcl.Pos{Line: 2, Column: 5, Byte: 12})
		require.Len(t, result, 2)
		assert.Equal(t, ref1, result[0])
		assert.Equal(t, ref2, result[1])
	})

	t.Run("returns nil for wrong file", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{defRange: {ref1}},
		}
		result := rm.FindReferencesFromDefinition("other.hcl", hcl.Pos{Line: 2, Column: 5})
		assert.Nil(t, result)
	})

	t.Run("returns nil for position outside definition", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{defRange: {ref1}},
		}
		result := rm.FindReferencesFromDefinition("test.hcl", hcl.Pos{Line: 99, Column: 1})
		assert.Nil(t, result)
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		rm := &ReferenceMap{
			DefToRefs: DefToRefs{},
		}
		result := rm.FindReferencesFromDefinition("test.hcl", hcl.Pos{Line: 1, Column: 1})
		assert.Nil(t, result)
	})
}

// --- Integration: ReferenceMap with real HCL ---

func TestReferenceMapIntegration(t *testing.T) {
	t.Run("local used in resource body creates ref and def", func(t *testing.T) {
		text := `
locals {
  region = "us-east-1"
}
resource vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
    spec = {
      forProvider = {
        region = region
      }
    }
  }
}
`
		files := parseFiles(t, text)
		targets := BuildTargets(files, nilDyn{}, nil)
		rm := BuildReferenceMap(files, targets)
		require.Greater(t, len(rm.RefsToDef), 0, "should have references")

		// Find a ref that points to a real definition (non-pseudo)
		var foundDef *hcl.Range
		var refPos hcl.Pos
		for ref, def := range rm.RefsToDef {
			if def.Filename != "" && ref.Filename == testFile {
				d := def
				foundDef = &d
				refPos = ref.Start
				break
			}
		}
		require.NotNil(t, foundDef, "should have at least one real definition")

		// FindDefinitionFromReference should return the same def
		result := rm.FindDefinitionFromReference(testFile, refPos)
		require.NotNil(t, result)
		assert.Equal(t, *foundDef, *result)

		// FindReferencesFromDefinition: find any def that has refs
		for defR, refList := range rm.DefToRefs {
			if defR.Filename == testFile && len(refList) > 0 {
				refs := rm.FindReferencesFromDefinition(testFile, defR.Start)
				assert.Equal(t, len(refList), len(refs), "should find all references")
				return
			}
		}
		t.Log("no DefToRefs entries with real filename found, skipping FindReferencesFromDefinition check")
	})

	t.Run("req.composite reference has pseudo-range definition", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
    spec = {
      forProvider = {
        region = req.composite.spec.region
      }
    }
  }
}
`
		files := parseFiles(t, text)
		compositeSchema := &schema.AttributeSchema{
			Constraint: schema.Object{
				Attributes: map[string]*schema.AttributeSchema{
					"spec": {Constraint: schema.Object{
						Attributes: map[string]*schema.AttributeSchema{
							"region": {Constraint: schema.String{}},
						},
					}},
				},
			},
		}
		targets := BuildTargets(files, nilDyn{}, compositeSchema)
		rm := BuildReferenceMap(files, targets)

		// "req" is a built-in; its definition may be a pseudo-range
		// Just verify no panic and we get some references
		require.NotNil(t, rm)
		assert.Greater(t, len(rm.RefsToDef), 0)
	})
}
