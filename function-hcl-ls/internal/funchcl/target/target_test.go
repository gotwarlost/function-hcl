package target

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// --- SubSchema / processRelativeTraversal tests ---

func TestSubSchema(t *testing.T) {
	root := &schema.AttributeSchema{
		Constraint: schema.Object{
			Attributes: map[string]*schema.AttributeSchema{
				"a": {Constraint: schema.Object{
					Attributes: map[string]*schema.AttributeSchema{
						"b": {Constraint: schema.Object{
							Attributes: map[string]*schema.AttributeSchema{
								"c": {Constraint: schema.Bool{}},
							},
						}},
					},
				}},
				"x": {Constraint: schema.String{}},
			},
		},
	}

	t.Run("empty path returns root", func(t *testing.T) {
		assert.Equal(t, root, SubSchema(root))
	})

	t.Run("single level", func(t *testing.T) {
		assertAnyExprType(t, SubSchema(root, "x"), cty.String, "x")
	})

	t.Run("deep path", func(t *testing.T) {
		assertAnyExprType(t, SubSchema(root, "a", "b", "c"), cty.Bool, "a.b.c")
	})

	t.Run("missing key returns unknown", func(t *testing.T) {
		assert.Equal(t, unknownSchema, SubSchema(root, "missing"))
	})

	t.Run("nil root returns unknown", func(t *testing.T) {
		assert.Equal(t, unknownSchema, SubSchema(nil, "a"))
	})

	t.Run("non-object at intermediate returns unknown", func(t *testing.T) {
		assert.Equal(t, unknownSchema, SubSchema(root, "x", "deeper"))
	})
}

func TestSchemaForRelativeTraversal(t *testing.T) {
	root := &schema.AttributeSchema{
		Constraint: schema.Object{
			Attributes: map[string]*schema.AttributeSchema{
				"spec": {Constraint: schema.Object{
					Attributes: map[string]*schema.AttributeSchema{
						"region": {Constraint: schema.String{}},
					},
				}},
				"items": {Constraint: schema.List{
					Elem: schema.String{},
				}},
				"tags": {Constraint: schema.Map{
					Elem: schema.String{},
				}},
			},
		},
	}

	t.Run("attr traversal", func(t *testing.T) {
		trav := hcl.Traversal{
			hcl.TraverseRoot{Name: "spec"},
			hcl.TraverseAttr{Name: "region"},
		}
		assertAnyExprType(t, SchemaForRelativeTraversal(root, trav), cty.String, "spec.region")
	})

	t.Run("list traversal unwraps element", func(t *testing.T) {
		trav := hcl.Traversal{
			hcl.TraverseRoot{Name: "items"},
			hcl.TraverseIndex{Key: cty.NumberIntVal(0)},
		}
		assertAnyExprType(t, SchemaForRelativeTraversal(root, trav), cty.String, "items[0]")
	})

	t.Run("map traversal unwraps element", func(t *testing.T) {
		trav := hcl.Traversal{
			hcl.TraverseRoot{Name: "tags"},
			hcl.TraverseIndex{Key: cty.StringVal("env")},
		}
		assertAnyExprType(t, SchemaForRelativeTraversal(root, trav), cty.String, `tags["env"]`)
	})

	t.Run("empty traversal returns root", func(t *testing.T) {
		assert.Equal(t, root, SchemaForRelativeTraversal(root, hcl.Traversal{}))
	})

	t.Run("nil root returns unknown", func(t *testing.T) {
		trav := hcl.Traversal{hcl.TraverseRoot{Name: "x"}}
		assert.Equal(t, unknownSchema, SchemaForRelativeTraversal(nil, trav))
	})
}

// --- BuildTargets: global tree structure ---

func TestBuildTargetsGlobalTree(t *testing.T) {
	t.Run("req container always present", func(t *testing.T) {
		files := parseFiles(t, withResource(""))
		targets := BuildTargets(files, nilDyn{}, nil)
		assert.Contains(t, rootNames(targets.globals), "req")
	})

	t.Run("req has composite, composite_connection, context", func(t *testing.T) {
		files := parseFiles(t, withResource(""))
		targets := BuildTargets(files, nilDyn{}, nil)
		s := targets.globals.AsSchema()
		reqObj := assertObjectSchema(t, SubSchema(s, "req"), "req")
		assert.Contains(t, reqObj.Attributes, "composite")
		assert.Contains(t, reqObj.Attributes, "composite_connection")
		assert.Contains(t, reqObj.Attributes, "context")
	})

	t.Run("resource adds req.resource.<name> and req.connection.<name>", func(t *testing.T) {
		files := parseFiles(t, withResource(""))
		targets := BuildTargets(files, nilDyn{}, nil)
		s := targets.globals.AsSchema()
		resourceObj := assertObjectSchema(t, SubSchema(s, "req", "resource"), "req.resource")
		assert.Contains(t, resourceObj.Attributes, "vpc")
		connObj := assertObjectSchema(t, SubSchema(s, "req", "connection"), "req.connection")
		assert.Contains(t, connObj.Attributes, "vpc")
	})

	t.Run("multiple resources", func(t *testing.T) {
		text := `
resource alpha {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
resource beta {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
		files := parseFiles(t, text)
		targets := BuildTargets(files, nilDyn{}, nil)
		s := targets.globals.AsSchema()
		resourceObj := assertObjectSchema(t, SubSchema(s, "req", "resource"), "req.resource")
		assert.Contains(t, resourceObj.Attributes, "alpha")
		assert.Contains(t, resourceObj.Attributes, "beta")
	})
}

// --- BuildTargets: locals schema inference ---

func TestLocalsSchemaInference(t *testing.T) {
	t.Run("string literal", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  greeting = "hello"
}
`))
		assertAnyExprType(t, SubSchema(s, "greeting"), cty.String, "greeting")
	})

	t.Run("number literal", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  count = 42
}
`))
		assertAnyExprType(t, SubSchema(s, "count"), cty.Number, "count")
	})

	t.Run("bool literal", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  enabled = true
}
`))
		assertAnyExprType(t, SubSchema(s, "enabled"), cty.Bool, "enabled")
	})

	t.Run("template string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  msg = "hello ${42}"
}
`))
		assertAnyExprType(t, SubSchema(s, "msg"), cty.String, "msg")
	})

	t.Run("nested object", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  config = {
    name = "alice"
    meta = {
      age = 30
    }
  }
}
`))
		configObj := assertObjectSchema(t, SubSchema(s, "config"), "config")
		assert.Contains(t, configObj.Attributes, "name")
		metaObj := assertObjectSchema(t, configObj.Attributes["meta"], "config.meta")
		assert.Contains(t, metaObj.Attributes, "age")
	})

	t.Run("empty object", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  empty = {}
}
`))
		emptyObj := assertObjectSchema(t, SubSchema(s, "empty"), "empty")
		assert.Empty(t, emptyObj.Attributes)
	})

	t.Run("tuple infers list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = ["a", "b", "c"]
}
`))
		assertListSchema(t, SubSchema(s, "items"), "items")
	})

	t.Run("tuple of objects infers list of objects", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  people = [
    { name = "alice" },
    { name = "bob" },
  ]
}
`))
		lc := assertListSchema(t, SubSchema(s, "people"), "people")
		elemObj := assertObjectSchema(t, &schema.AttributeSchema{Constraint: lc.Elem}, "people element")
		assert.Contains(t, elemObj.Attributes, "name")
	})
}

// --- cross-local reference chasing ---

func TestLocalsCrossReference(t *testing.T) {
	t.Run("local referencing another local", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  first  = { name = "alice", age = 30 }
  second = first
}
`))
		secondObj := assertObjectSchema(t, SubSchema(s, "second"), "second")
		assert.Contains(t, secondObj.Attributes, "name")
		assert.Contains(t, secondObj.Attributes, "age")
	})

	t.Run("chained local references", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  base   = { x = 1 }
  middle = base
  final  = middle
}
`))
		finalObj := assertObjectSchema(t, SubSchema(s, "final"), "final")
		assert.Contains(t, finalObj.Attributes, "x")
	})

	t.Run("local traversal into another local", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  config = { db = { host = "localhost" } }
  host   = config.db.host
}
`))
		assertAnyExprType(t, SubSchema(s, "host"), cty.String, "host")
	})

	t.Run("circular reference does not panic", func(t *testing.T) {
		files := parseFiles(t, withResource(`
locals {
  a = b
  b = a
}
`))
		targets := BuildTargets(files, nilDyn{}, nil)
		require.NotNil(t, targets)
	})
}

// --- operator schema inference ---

func TestLocalsOperatorInference(t *testing.T) {
	t.Run("addition returns number", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  sum = 1 + 2
}
`))
		assertAnyExprType(t, SubSchema(s, "sum"), cty.Number, "sum")
	})

	t.Run("comparison returns bool", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  check = 1 > 2
}
`))
		assertAnyExprType(t, SubSchema(s, "check"), cty.Bool, "check")
	})

	t.Run("logical and returns bool", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  both = true && false
}
`))
		assertAnyExprType(t, SubSchema(s, "both"), cty.Bool, "both")
	})

	t.Run("negation returns number", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  neg = -5
}
`))
		assertAnyExprType(t, SubSchema(s, "neg"), cty.Number, "neg")
	})
}

// --- function call schema inference ---

func TestLocalsFunctionInference(t *testing.T) {
	// --- primitive return types ---

	t.Run("upper returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = upper("hello")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("lower returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = lower("HELLO")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("length returns number", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = length("hello")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.Number, "val")
	})

	t.Run("tobool returns bool", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = tobool("true")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.Bool, "val")
	})

	t.Run("tonumber returns number", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = tonumber("42")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.Number, "val")
	})

	t.Run("tostring returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = tostring(42)
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("format returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = format("%s-%s", "a", "b")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("join returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = join(",", ["a", "b"])
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("replace returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = replace("hello", "l", "r")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("trimspace returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = trimspace("  hi  ")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("substr returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = substr("hello", 0, 3)
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	// --- merge: union of object schemas ---

	t.Run("merge combines object schemas", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  combined = merge({ a = "x" }, { b = 1 })
}
`))
		combinedObj := assertObjectSchema(t, SubSchema(s, "combined"), "combined")
		assert.Contains(t, combinedObj.Attributes, "a")
		assert.Contains(t, combinedObj.Attributes, "b")
	})

	t.Run("merge with three objects", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  combined = merge({ a = "x" }, { b = 1 }, { c = true })
}
`))
		combinedObj := assertObjectSchema(t, SubSchema(s, "combined"), "combined")
		assert.Contains(t, combinedObj.Attributes, "a")
		assert.Contains(t, combinedObj.Attributes, "b")
		assert.Contains(t, combinedObj.Attributes, "c")
	})

	t.Run("merge with overlapping keys keeps non-unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  combined = merge({ name = "alice" }, { name = "bob", extra = 1 })
}
`))
		combinedObj := assertObjectSchema(t, SubSchema(s, "combined"), "combined")
		assert.Contains(t, combinedObj.Attributes, "name")
		assert.Contains(t, combinedObj.Attributes, "extra")
	})

	// --- coalesce, coalescelist, concat, distinct, reverse, sort etc: findOneFrom ---

	t.Run("coalesce infers from first arg", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = coalesce("a", "b")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("concat infers list from arg", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = concat(["a"], ["b"])
}
`))
		assertListSchema(t, SubSchema(s, "val"), "val")
	})

	t.Run("distinct preserves list type", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = distinct(["a", "a", "b"])
}
`))
		assertListSchema(t, SubSchema(s, "val"), "val")
	})

	t.Run("reverse preserves list type", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = reverse(["a", "b"])
}
`))
		assertListSchema(t, SubSchema(s, "val"), "val")
	})

	t.Run("sort preserves list type", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = sort(["b", "a"])
}
`))
		assertListSchema(t, SubSchema(s, "val"), "val")
	})

	t.Run("try infers from first arg", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = try({ name = "x" }, {})
}
`))
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "name")
	})

	// --- element, flatten, one: unwrap list ---

	t.Run("element unwraps list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = [{ id = "a" }]
  val   = element(items, 0)
}
`))
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "id")
	})

	t.Run("flatten unwraps list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = [{ id = "a" }]
  val   = flatten(items)
}
`))
		// flatten unwraps one level of list nesting
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "id")
	})

	t.Run("one unwraps list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = [{ id = "a" }]
  val   = one(items)
}
`))
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "id")
	})

	// --- toset: list to set ---

	t.Run("toset converts list to set", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = ["a", "b"]
  val   = toset(items)
}
`))
		valSchema := SubSchema(s, "val")
		require.NotNil(t, valSchema)
		_, ok := valSchema.Constraint.(schema.Set)
		assert.True(t, ok, "toset should produce Set, got %T", valSchema.Constraint)
	})

	// --- matchkeys: same as first arg ---

	t.Run("matchkeys preserves first arg type", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = [{ name = "a" }]
  val   = matchkeys(items, ["k1"], ["k1"])
}
`))
		assertListSchema(t, SubSchema(s, "val"), "val")
	})

	// --- chunklist: list of the arg ---

	t.Run("chunklist wraps in list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = ["a", "b", "c"]
  val   = chunklist(items, 2)
}
`))
		lc := assertListSchema(t, SubSchema(s, "val"), "val")
		// the element of chunklist is the original list constraint
		_, ok := lc.Elem.(schema.List)
		assert.True(t, ok, "chunklist element should be List, got %T", lc.Elem)
	})

	// --- lookup: infer from default or unwrap map ---

	t.Run("lookup infers from default arg", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = lookup({ a = "x" }, "a", "default")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("lookup falls back to map element type", func(t *testing.T) {
		// Args[1] is an unknown variable reference, so impliedSchema returns nil.
		// This causes lookup to fall through to unwrapping the first arg's Map
		// constraint. req.composite_connection is Map{Elem: AnyExpression{String}}.
		s := localSchema(t, withResource(`
locals {
  val = lookup(req.composite_connection, unknown_var)
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val from map element")
	})

	// --- values: unwrap map to list ---

	t.Run("values converts map values to list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  src = { a = "x", b = "y" }
  val = values(src)
}
`))
		// values on an object won't work (needs schema.Map),
		// but values on something inferred as Map would.
		// Since src is inferred as Object, this may return nil.
		// Just verify no panic.
		_ = SubSchema(s, "val")
	})

	// --- explicitly unsupported functions ---

	t.Run("tomap returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = tomap({ a = "x" })
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "val"))
	})

	t.Run("zipmap returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = zipmap(["a"], ["x"])
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "val"))
	})

	t.Run("setproduct returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = setproduct(["a"], ["b"])
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "val"))
	})

	// --- unknown function ---

	t.Run("unknown function stays unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = nonexistent_fn("hello")
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "val"))
	})

	// --- functions with list return type (fallback) ---

	t.Run("split returns list of string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = split(",", "a,b,c")
}
`))
		lc := assertListSchema(t, SubSchema(s, "val"), "val")
		elemSchema := &schema.AttributeSchema{Constraint: lc.Elem}
		assertAnyExprType(t, elemSchema, cty.String, "split element")
	})

	t.Run("regexall returns list of string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = regexall("[a-z]+", "abc def")
}
`))
		assertListSchema(t, SubSchema(s, "val"), "val")
	})

	// --- chained function calls ---

	t.Run("upper of lower returns string", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = upper(lower("Hello"))
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("length of merge returns number", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = length(merge({a = 1}, {b = 2}))
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.Number, "val")
	})

	t.Run("element of concat returns object", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  list1 = [{ id = "a" }]
  list2 = [{ id = "b" }]
  val   = element(concat(list1, list2), 0)
}
`))
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "id")
	})

	// --- function with local variable args ---

	t.Run("merge with local variable args", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  base    = { name = "default" }
  overlay = { env  = "prod" }
  val     = merge(base, overlay)
}
`))
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "name")
		assert.Contains(t, valObj.Attributes, "env")
	})

	t.Run("coalesce with local refs", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  primary   = { host = "prod-db" }
  secondary = { host = "backup-db" }
  val       = coalesce(primary, secondary)
}
`))
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "host")
	})
}

// --- conditional inference ---

func TestLocalsConditionalInference(t *testing.T) {
	t.Run("conditional infers from true branch", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = true ? "yes" : "no"
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("conditional with object branches", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = true ? { name = "a" } : { name = "b" }
}
`))
		valObj := assertObjectSchema(t, SubSchema(s, "val"), "val")
		assert.Contains(t, valObj.Attributes, "name")
	})
}

// --- parentheses and template wrap ---

func TestLocalsParenAndWrap(t *testing.T) {
	t.Run("parenthesized expression", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = ("hello")
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})

	t.Run("template wrap", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  name = "world"
  val  = "${name}"
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "val")
	})
}

// --- index expression inference ---

func TestLocalsIndexInference(t *testing.T) {
	t.Run("index into object with literal key", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  obj    = { foo = "bar", baz = 42 }
  picked = obj["foo"]
}
`))
		assertAnyExprType(t, SubSchema(s, "picked"), cty.String, "picked")
	})

	t.Run("index into object with variable key", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  key    = "foo"
  obj    = { foo = "bar" }
  picked = obj[key]
}
`))
		assertAnyExprType(t, SubSchema(s, "picked"), cty.String, "picked via variable key")
	})

	t.Run("index into list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items  = ["a", "b", "c"]
  picked = items[0]
}
`))
		assertAnyExprType(t, SubSchema(s, "picked"), cty.String, "list element")
	})

	t.Run("index into map via req.composite_connection", func(t *testing.T) {
		// req.composite_connection is Map{Elem: AnyExpression{String}}
		s := localSchema(t, withResource(`
locals {
  val = req.composite_connection["key"]
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "map element")
	})

	// The above two tests exercise processRelativeTraversal (TraverseIndex steps),
	// not schemaFromIndexExpression. An IndexExpr is only produced when the
	// collection is a complex expression. The following tests hit the actual
	// schemaFromIndexExpression branches.

	t.Run("IndexExpr into list via complex source", func(t *testing.T) {
		// (items)[0] produces IndexExpr{Collection: ParenthesesExpr, Key: 0}
		s := localSchema(t, withResource(`
locals {
  items  = ["a", "b"]
  picked = (items)[0]
}
`))
		assertAnyExprType(t, SubSchema(s, "picked"), cty.String, "IndexExpr list element")
	})

	t.Run("IndexExpr into map via complex source", func(t *testing.T) {
		// (req.composite_connection)["key"] produces an IndexExpr
		s := localSchema(t, withResource(`
locals {
  val = (req.composite_connection)["key"]
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "IndexExpr map element")
	})

	t.Run("IndexExpr into object via function call", func(t *testing.T) {
		// merge({...})["foo"] produces IndexExpr{Collection: FunctionCallExpr}
		s := localSchema(t, withResource(`
locals {
  picked = merge({ foo = "bar" }, {})["foo"]
}
`))
		assertAnyExprType(t, SubSchema(s, "picked"), cty.String, "IndexExpr object element")
	})

	t.Run("index into object with missing key returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  obj    = { foo = "bar" }
  picked = obj["nonexistent"]
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "picked"))
	})

	t.Run("index into object with non-resolvable key returns unknown", func(t *testing.T) {
		// unknown_var is not a known local, so asStringValue returns false
		s := localSchema(t, withResource(`
locals {
  obj    = { foo = "bar" }
  picked = obj[unknown_var]
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "picked"))
	})

	t.Run("index with nil collection schema returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  picked = unknown_var[0]
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "picked"))
	})

	t.Run("index into string-typed local returns unknown", func(t *testing.T) {
		// string constraint is not Map/List/Object, falls through switch
		s := localSchema(t, withResource(`
locals {
  str    = "hello"
  picked = str[0]
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "picked"))
	})
}

// --- splat expression inference ---

func TestLocalsSplatInference(t *testing.T) {
	t.Run("splat extracts field from list of objects", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  people = [{ name = "alice" }, { name = "bob" }]
  names  = people[*].name
}
`))
		lc := assertListSchema(t, SubSchema(s, "names"), "names")
		assertAnyExprType(t, &schema.AttributeSchema{Constraint: lc.Elem}, cty.String, "names element")
	})

	t.Run("splat identity returns original list", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  items = [{ x = 1 }]
  all   = items[*]
}
`))
		assertListSchema(t, SubSchema(s, "all"), "all")
	})

	t.Run("splat on unknown source returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = unknown_var[*].name
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "val"))
	})

	t.Run("splat on non-list returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  obj = { a = 1 }
  val = obj[*].a
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "val"))
	})

	t.Run("splat traversal into missing field wraps unknown in list", func(t *testing.T) {
		// processRelativeTraversal returns unknownSchema (not nil) for missing fields,
		// so schemaFromSplat wraps it in List{Elem: unknownSchema.Constraint}
		s := localSchema(t, withResource(`
locals {
  items = [{ x = 1 }]
  val   = items[*].nonexistent
}
`))
		lc := assertListSchema(t, SubSchema(s, "val"), "val")
		// element is the unknown AnyExpression{DynamicPseudoType}
		assert.Equal(t, unknownSchema.Constraint, lc.Elem)
	})
}

// --- relative traversal inference ---

func TestLocalsRelativeTraversalInference(t *testing.T) {
	t.Run("scope traversal into local", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  combined = merge({ name = "alice", age = 30 }, {})
  name     = combined.name
}
`))
		assertAnyExprType(t, SubSchema(s, "name"), cty.String, "name")
	})

	t.Run("relative traversal on function call", func(t *testing.T) {
		// merge({...}).name parses as RelativeTraversalExpr (source=FunctionCallExpr)
		s := localSchema(t, withResource(`
locals {
  name = merge({ name = "alice" }, {}).name
}
`))
		assertAnyExprType(t, SubSchema(s, "name"), cty.String, "name via relative traversal")
	})

	t.Run("relative traversal deep path", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = merge({ a = { b = "deep" } }, {}).a.b
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.String, "deep relative traversal")
	})

	t.Run("relative traversal on parenthesized expression", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  obj = { x = 42 }
  val = (obj).x
}
`))
		assertAnyExprType(t, SubSchema(s, "val"), cty.Number, "paren relative traversal")
	})

	t.Run("relative traversal with unknown source returns unknown", func(t *testing.T) {
		s := localSchema(t, withResource(`
locals {
  val = unknown_fn("x").field
}
`))
		assert.Equal(t, unknownSchema, SubSchema(s, "val"))
	})
}

// --- VisibleTreeAt scoping ---

func TestVisibleTreeAtScoping(t *testing.T) {
	t.Run("top level returns empty tree", func(t *testing.T) {
		files := parseFiles(t, withResource(`
locals {
  foo = "bar"
}
`))
		targets := BuildTargets(files, nilDyn{}, nil)
		tree := targets.VisibleTreeAt(nil, testFile, hcl.Pos{Line: 1, Column: 1, Byte: 0})
		assert.Empty(t, tree.Roots())
	})

	t.Run("inside resource sees globals and self", func(t *testing.T) {
		_, tree := buildAndVisible(t, withResource(`
locals {
  greeting = "hello"
}
`), nil, "resource")
		names := rootNames(tree)
		assert.Contains(t, names, "req")
		assert.Contains(t, names, "greeting")
		assert.Contains(t, names, "self")
	})

	t.Run("scoped locals inside resource block", func(t *testing.T) {
		text := `
resource vpc {
  locals {
    inner = "scoped"
  }
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
		_, tree := buildAndVisible(t, text, nil, "resource")
		assertAnyExprType(t, SubSchema(tree.AsSchema(), "inner"), cty.String, "inner")
	})

	t.Run("function block does not see globals", func(t *testing.T) {
		text := `
locals {
  global_var = "hello"
}
function my_func {
  arg name {
    type = string
  }
  body = {
    result = name
  }
}
`
		files := parseFiles(t, text)
		targets := BuildTargets(files, nilDyn{}, nil)
		body := files[testFile].Body.(*hclsyntax.Body)
		for _, b := range body.Blocks {
			if b.Type == "function" {
				pos := b.Body.SrcRange.Start
				pos.Column += 2
				tree := targets.VisibleTreeAt(b.AsHCLBlock(), testFile, pos)
				names := rootNames(tree)
				assert.NotContains(t, names, "req")
				assert.NotContains(t, names, "global_var")
				assert.Contains(t, names, "name")
				return
			}
		}
		t.Fatal("function block not found")
	})
}

// --- self.resource and self.resources ---

func TestSelfResourceScoping(t *testing.T) {
	t.Run("self.resource visible inside resource block", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
    spec = {
      forProvider = {
        region = self.name
      }
    }
  }
}
`
		_, tree := buildAndVisible(t, text, nil, "resource")
		names := rootNames(tree)
		assert.Contains(t, names, "self")

		s := tree.AsSchema()
		selfObj := assertObjectSchema(t, SubSchema(s, "self"), "self")
		assert.Contains(t, selfObj.Attributes, "name", "self should have name")
		assert.Contains(t, selfObj.Attributes, "connection", "self should have connection")
		assert.Contains(t, selfObj.Attributes, "resource", "self should have resource")

		// self.name is a string
		assertAnyExprType(t, selfObj.Attributes["name"], cty.String, "self.name")

		// self.connection is a map of strings
		connSchema := selfObj.Attributes["connection"]
		require.NotNil(t, connSchema)
		mc, ok := connSchema.Constraint.(schema.Map)
		require.True(t, ok, "self.connection should be Map, got %T", connSchema.Constraint)
		_, ok = mc.Elem.(schema.String)
		assert.True(t, ok, "self.connection element should be string")

		// self.resource is the resource's own schema (with status only)
		require.NotNil(t, selfObj.Attributes["resource"])
	})

	t.Run("self.resource schema matches req.resource schema", func(t *testing.T) {
		text := `
resource vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
		_, tree := buildAndVisible(t, text, nil, "resource")
		s := tree.AsSchema()

		selfResource := SubSchema(s, "self", "resource")
		reqResource := SubSchema(s, "req", "resource", "vpc")
		assert.Equal(t, selfResource, reqResource,
			"self.resource should match req.resource.vpc")
	})

	t.Run("self not visible at top level", func(t *testing.T) {
		files := parseFiles(t, withResource(""))
		targets := BuildTargets(files, nilDyn{}, nil)
		tree := targets.VisibleTreeAt(nil, testFile, hcl.Pos{Line: 1, Column: 1})
		names := rootNames(tree)
		assert.NotContains(t, names, "self")
	})

	t.Run("self.resources visible inside resources/template block", func(t *testing.T) {
		text := `
resources subnets {
  for_each = req.composite.spec.subnets
  template {
    body = {
      apiVersion = "ec2.aws.upbound.io/v1beta1"
      kind       = "Subnet"
    }
  }
}
`
		files := parseFiles(t, text)
		targets := BuildTargets(files, nilDyn{}, nil)

		body := files[testFile].Body.(*hclsyntax.Body)
		for _, b := range body.Blocks {
			if b.Type == "resources" {
				for _, inner := range b.Body.Blocks {
					if inner.Type == "template" {
						pos := inner.Body.SrcRange.Start
						pos.Column += 2
						tree := targets.VisibleTreeAt(inner.AsHCLBlock(), testFile, pos)
						s := tree.AsSchema()

						selfObj := assertObjectSchema(t, SubSchema(s, "self"), "self in template")
						assert.Contains(t, selfObj.Attributes, "basename", "self should have basename")
						assert.Contains(t, selfObj.Attributes, "connections", "self should have connections")
						assert.Contains(t, selfObj.Attributes, "resources", "self should have resources")

						// self.basename is a string
						assertAnyExprType(t, selfObj.Attributes["basename"], cty.String, "self.basename")

						// self.connections is list of map of strings
						connsSchema := selfObj.Attributes["connections"]
						require.NotNil(t, connsSchema)
						lc, ok := connsSchema.Constraint.(schema.List)
						require.True(t, ok, "self.connections should be List, got %T", connsSchema.Constraint)
						_, ok = lc.Elem.(schema.Map)
						assert.True(t, ok, "self.connections element should be Map")

						// self.resources is a list
						resSchema := selfObj.Attributes["resources"]
						require.NotNil(t, resSchema)
						_, ok = resSchema.Constraint.(schema.List)
						assert.True(t, ok, "self.resources should be List, got %T", resSchema.Constraint)
						return
					}
				}
			}
		}
		t.Fatal("template block not found inside resources")
	})

	t.Run("local referencing self.name infers string", func(t *testing.T) {
		text := `
resource vpc {
  locals {
    myname = self.name
  }
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
		_, tree := buildAndVisible(t, text, nil, "resource")
		assertAnyExprType(t, SubSchema(tree.AsSchema(), "myname"), cty.String, "myname from self.name")
	})

	t.Run("local referencing self.connection infers map", func(t *testing.T) {
		text := `
resource vpc {
  locals {
    conn = self.connection
  }
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
		_, tree := buildAndVisible(t, text, nil, "resource")
		s := SubSchema(tree.AsSchema(), "conn")
		require.NotNil(t, s)
		// selfSchema uses *schema.Map (pointer), so check both forms
		switch s.Constraint.(type) {
		case schema.Map, *schema.Map:
			// ok
		default:
			t.Fatalf("conn from self.connection: expected Map, got %T", s.Constraint)
		}
	})

	t.Run("local referencing self.resource in standalone resource resolves", func(t *testing.T) {
		text := `
resource vpc {
  locals {
    res = self.resource
  }
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
		_, tree := buildAndVisible(t, text, nil, "resource")
		s := SubSchema(tree.AsSchema(), "res")
		reqResource := SubSchema(tree.AsSchema(), "req", "resource", "vpc")
		assert.Equal(t, reqResource, s,
			"local from self.resource should match req.resource.vpc")
	})

	t.Run("local referencing self.basename in template infers string", func(t *testing.T) {
		text := `
resources subnets {
  for_each = req.composite.spec.subnets
  template {
    locals {
      bn = self.basename
    }
    body = {
      apiVersion = "ec2.aws.upbound.io/v1beta1"
      kind       = "Subnet"
    }
  }
}
`
		files := parseFiles(t, text)
		targets := BuildTargets(files, nilDyn{}, nil)

		body := files[testFile].Body.(*hclsyntax.Body)
		for _, b := range body.Blocks {
			if b.Type == "resources" {
				for _, inner := range b.Body.Blocks {
					if inner.Type == "template" {
						pos := inner.Body.SrcRange.Start
						pos.Column += 2
						tree := targets.VisibleTreeAt(inner.AsHCLBlock(), testFile, pos)
						assertAnyExprType(t, SubSchema(tree.AsSchema(), "bn"), cty.String, "bn from self.basename")
						return
					}
				}
			}
		}
		t.Fatal("template block not found")
	})

	t.Run("selfSchema returns nil outside resource/resources", func(t *testing.T) {
		text := `
locals {
  val = self.name
}
resource vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
  }
}
`
		_, tree := buildAndVisible(t, text, nil, "resource")
		// "val" is a top-level local referencing self, but selfSchema returns nil
		// at the global scope, so val should remain unknown
		assert.Equal(t, unknownSchema, SubSchema(tree.AsSchema(), "val"))
	})

	t.Run("each variable visible inside resources/template", func(t *testing.T) {
		text := `
resources subnets {
  for_each = req.composite.spec.subnets
  template {
    body = {
      apiVersion = "ec2.aws.upbound.io/v1beta1"
      kind       = "Subnet"
    }
  }
}
`
		files := parseFiles(t, text)
		targets := BuildTargets(files, nilDyn{}, nil)

		body := files[testFile].Body.(*hclsyntax.Body)
		for _, b := range body.Blocks {
			if b.Type == "resources" {
				for _, inner := range b.Body.Blocks {
					if inner.Type == "template" {
						pos := inner.Body.SrcRange.Start
						pos.Column += 2
						tree := targets.VisibleTreeAt(inner.AsHCLBlock(), testFile, pos)
						names := rootNames(tree)
						assert.Contains(t, names, "each", "each should be visible inside resources/template")
						return
					}
				}
			}
		}
		t.Fatal("resources block not found")
	})
}

// --- composite schema integration ---

func TestBuildTargetsWithCompositeSchema(t *testing.T) {
	compositeSchema := &schema.AttributeSchema{
		Constraint: schema.Object{
			Attributes: map[string]*schema.AttributeSchema{
				"apiVersion": {Constraint: schema.String{}},
				"kind":       {Constraint: schema.String{}},
				"metadata": {Constraint: schema.Object{
					Attributes: map[string]*schema.AttributeSchema{
						"name": {Constraint: schema.String{}},
					},
				}},
				"spec": {Constraint: schema.Object{
					Attributes: map[string]*schema.AttributeSchema{
						"region": {Constraint: schema.String{}},
					},
				}},
			},
		},
	}

	t.Run("req.composite has schema without apiVersion and kind", func(t *testing.T) {
		_, tree := buildAndVisible(t, withResource(""), compositeSchema, "resource")
		compObj := assertObjectSchema(t, SubSchema(tree.AsSchema(), "req", "composite"), "req.composite")
		assert.NotContains(t, compObj.Attributes, "apiVersion")
		assert.NotContains(t, compObj.Attributes, "kind")
		assert.Contains(t, compObj.Attributes, "metadata")
		assert.Contains(t, compObj.Attributes, "spec")
	})

	t.Run("local referencing req.composite.spec", func(t *testing.T) {
		_, tree := buildAndVisible(t, withResource(`
locals {
  spec = req.composite.spec
}
`), compositeSchema, "resource")
		specObj := assertObjectSchema(t, SubSchema(tree.AsSchema(), "spec"), "spec local")
		assert.Contains(t, specObj.Attributes, "region")
	})
}

// --- ReferenceMap tests ---

func TestBuildReferenceMap(t *testing.T) {
	t.Run("traversal creates reference", func(t *testing.T) {
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
		refMap := BuildReferenceMap(files, targets)
		require.NotNil(t, refMap)
		assert.Greater(t, len(refMap.RefsToDef), 0,
			"should have at least one reference")
	})
}
