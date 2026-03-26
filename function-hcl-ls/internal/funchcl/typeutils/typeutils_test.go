package typeutils

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// --- TypeConstraint / KVSchema tests ---

func TestTypeConstraint(t *testing.T) {
	t.Run("string type", func(t *testing.T) {
		c := TypeConstraint(cty.String)
		_, ok := c.(schema.String)
		require.True(t, ok)
	})

	t.Run("object type", func(t *testing.T) {
		c := TypeConstraint(cty.Object(map[string]cty.Type{"name": cty.String}))
		oc, ok := c.(schema.Object)
		require.True(t, ok)
		assert.Contains(t, oc.Attributes, "name")
	})

	t.Run("list type", func(t *testing.T) {
		c := TypeConstraint(cty.List(cty.Number))
		lc, ok := c.(schema.List)
		require.True(t, ok)
		_, ok = lc.Elem.(schema.Number)
		assert.True(t, ok)
	})

	t.Run("map type", func(t *testing.T) {
		c := TypeConstraint(cty.Map(cty.Bool))
		mc, ok := c.(schema.Map)
		require.True(t, ok)
		_, ok = mc.Elem.(schema.Bool)
		assert.True(t, ok)
	})

	t.Run("set type", func(t *testing.T) {
		c := TypeConstraint(cty.Set(cty.String))
		sc, ok := c.(schema.Set)
		require.True(t, ok)
		_, ok = sc.Elem.(schema.String)
		assert.True(t, ok)
	})
}

func TestKvSchema(t *testing.T) {
	t.Run("list type produces number key", func(t *testing.T) {
		base := &schema.AttributeSchema{
			Constraint: schema.List{Elem: schema.String{}},
		}
		kv := KVSchema(base)
		require.NotNil(t, kv)
		kvObj := assertObjectSchema(t, kv, "kv")
		assert.Contains(t, kvObj.Attributes, "key")
		assert.Contains(t, kvObj.Attributes, "value")
	})

	t.Run("map type produces string key", func(t *testing.T) {
		base := &schema.AttributeSchema{
			Constraint: schema.Map{Elem: schema.Number{}},
		}
		kv := KVSchema(base)
		require.NotNil(t, kv)
		kvObj := assertObjectSchema(t, kv, "kv")
		assert.Contains(t, kvObj.Attributes, "key")
		assert.Contains(t, kvObj.Attributes, "value")
	})

	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, KVSchema(nil))
	})
}

// assertObjectSchema asserts the constraint is a schema.Object and returns it.
func assertObjectSchema(t *testing.T, s *schema.AttributeSchema, msg string) schema.Object {
	t.Helper()
	require.NotNil(t, s, msg)
	oc, ok := s.Constraint.(schema.Object)
	require.True(t, ok, "%s: expected schema.Object, got %T", msg, s.Constraint)
	return oc
}
