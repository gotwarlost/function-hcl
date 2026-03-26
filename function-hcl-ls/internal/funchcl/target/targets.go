package target

import (
	"fmt"
	"log"

	ourschema "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/schema"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (t *Tree) add(node *Node, pathElements ...string) {
	current := t.root
	found := false
	node.IsContainer = false // force non-container leaf

	for _, el := range pathElements {
		found = false
		for _, child := range current.Children {
			if child.Name == el {
				if !child.IsContainer {
					log.Printf("error: attempt to replace child with name %q with a container", el)
					return
				}
				found = true
				current = child
				break
			}
		}
		if !found {
			tmp := &Node{
				Name:        el,
				IsContainer: true,
			}
			current.Children = append(current.Children, tmp)
			current = tmp
		}
	}
	current.Children = append(current.Children, node)
}

type scopedLocal struct {
	name           string                  // the local variable name
	definition     hcl.Range               // the range where the node is defined
	schema         *schema.AttributeSchema // the read schema for the local
	accessibleFrom hcl.Range               // the range from which this local is accessible
}

type alias struct {
	definition     hcl.Range               // the range where the node is defined
	accessibleFrom hcl.Range               // the range from which the node can be accessed
	schema         *schema.AttributeSchema // the read schema for the local
}

func buildTargets(files map[string]*hcl.File, dyn ourschema.DynamicLookup, compositeSchema *schema.AttributeSchema) *Targets {
	compSchema := compositeSchema
	if compSchema == nil {
		compSchema = unknownSchema
	}
	t := newTree()
	t.add(&Node{
		Name:   "composite",
		Schema: ourschema.WithoutAPIVersionAndKind(compSchema),
	}, "req")
	t.add(&Node{
		Name:   "composite_connection",
		Schema: &schema.AttributeSchema{Constraint: schema.Map{Elem: schema.String{}}},
	}, "req")
	t.add(&Node{
		Name:   "context",
		Schema: unknownSchema,
	}, "req")

	ret := &Targets{
		CompositeSchema:    compositeSchema,
		globals:            t,
		scopedLocalsByFile: map[string][]*scopedLocal{},
		resourceByFile:     map[string][]*alias{},
		collectionByFile:   map[string][]*alias{},
	}

	rootLocals := newLocalsCollection(nil, hcl.Range{})

	for filename, f := range files {
		body := f.Body.(*hclsyntax.Body)
		bs := schema.NewBlockStack()
		walk(body, bs, rootLocals, func(bs schema.BlockStack, coll *localsCollection) *localsCollection {
			out := coll // by default this does not change when scope does not change
			current := bs.Peek(0)
			parent := bs.Peek(1)
			switch current.Type {

			case "resources":
				// set up the targetable each alias for the collection
				eachAttr := current.Body.Attributes["for_each"]
				var eachExpr hclsyntax.Expression
				if eachAttr != nil {
					eachExpr = eachAttr.Expr
				}
				// always create a new locals scope. When processing locals
				// take this into account because the scope of the `each` variable
				// is the same as the scope of locals just under resources.
				out = newLocalsCollection(coll, current.Range())
				if eachExpr != nil {
					// add a scoped collection
					ret.scopedLocalsByFile[filename] = append(ret.scopedLocalsByFile[filename], &scopedLocal{
						name:           "each",
						schema:         unknownSchema,
						definition:     eachAttr.NameRange,
						accessibleFrom: current.Range(),
					})
					out.locals["each"] = &local{
						name: "each",
						expr: eachExpr,
					}
				}
			// a resource implies a global and local addressable entity as long as it has a name.
			case "resource":
				if len(current.Labels) == 0 {
					log.Printf("resource block at %s, %v does not have a name label", filename, current.TypeRange)
					return out
				}
				sch := ourschema.WithStatusOnly(ourschema.DependentSchemaOrDefault(dyn, current))
				defRange := hcl.RangeBetween(current.TypeRange, current.OpenBraceRange)
				ret.declarationRanges = append(ret.declarationRanges, defRange)
				// add a req.resource.<foo> entry and a req.connection.<foo> entry
				t.add(&Node{
					Name:       current.Labels[0],
					Schema:     sch,
					Definition: defRange,
					NameRange:  current.LabelRanges[0],
				}, "req", "resource")
				t.add(&Node{
					Name:       current.Labels[0],
					Schema:     &schema.AttributeSchema{Constraint: schema.Map{Elem: schema.String{}}},
					Definition: defRange,
					NameRange:  current.LabelRanges[0],
				}, "req", "connection")

				// set up the targetable self alias for the resource
				ret.resourceByFile[filename] = append(ret.resourceByFile[filename], &alias{
					schema:         sch,
					definition:     defRange,
					accessibleFrom: current.Range(),
				})

			// a template implies a collection and a resource, provided the correct structure is defined
			case "template":
				if parent.Type != "resources" {
					log.Printf("template block at %s, %v does not have a resources parent, found %q", filename, current.TypeRange, parent.Type)
					return out
				}
				if len(parent.Labels) == 0 {
					log.Printf("resources block at %s, %v does not have a name label", filename, parent.TypeRange)
					return out
				}

				sch := ourschema.DependentSchemaOrDefault(dyn, current)
				defRange := hcl.RangeBetween(parent.TypeRange, parent.OpenBraceRange)
				// add a req.resources.<foo> entry and a req.connections.<foo> entry
				t.add(&Node{
					Name: parent.Labels[0],
					Schema: &schema.AttributeSchema{
						Constraint: schema.List{Elem: sch.Constraint},
					},
					Definition: defRange,
					NameRange:  parent.LabelRanges[0],
				}, "req", "resources")
				t.add(&Node{
					Name: parent.Labels[0],
					Schema: &schema.AttributeSchema{
						Constraint: schema.List{Elem: schema.Map{Elem: schema.String{}}},
					},
					Definition: defRange,
					NameRange:  parent.LabelRanges[0],
				}, "req", "connections")

				// set up the targetable self alias for the collection
				ret.collectionByFile[filename] = append(ret.collectionByFile[filename], &alias{
					schema:         sch,
					definition:     hcl.RangeBetween(parent.TypeRange, parent.OpenBraceRange),
					accessibleFrom: parent.Range(),
				})

				// set up the targetable self alias for the resource
				ret.resourceByFile[filename] = append(ret.resourceByFile[filename], &alias{
					schema:         sch,
					definition:     defRange,
					accessibleFrom: hcl.RangeBetween(current.TypeRange, current.OpenBraceRange),
				})
				// locals are accessible in parent scope or global for file scoped locals
			case "locals":
				if parent.Type != "" && parent.Type != "resources" {
					out = newLocalsCollection(coll, parent.Range()) // create child scope or update root in place
					n := 1
					for {
						b := bs.Peek(n)
						n++
						if b.Type == "" {
							break
						}
						if b.Type == "function" { // a function block cannot see the outer world
							break
						}
						if b.Type == "resource" && len(b.Labels) > 0 {
							out.parentResource = b.Labels[0]
						}
						if b.Type == "resources" && len(b.Labels) > 0 {
							out.parentCollection = b.Labels[0]
						}
					}
				}
				for attrName, attr := range current.Body.Attributes {
					// if root level, add directly to the global tree
					if parent.Type == "" {
						node := &Node{
							Name:       attrName,
							Schema:     unknownSchema,
							Definition: attr.NameRange,
							NameRange:  attr.NameRange,
						}
						t.add(node)
					} else {
						// add a scoped collection
						ret.scopedLocalsByFile[filename] = append(ret.scopedLocalsByFile[filename], &scopedLocal{
							name:           attrName,
							schema:         unknownSchema,
							definition:     attr.NameRange,
							accessibleFrom: parent.Range(),
						})
					}
					out.locals[attrName] = &local{
						name: attrName,
						expr: attr.Expr,
					}
				}
			// an arg is a local in function scope
			case "arg":
				if len(current.Labels) == 0 {
					log.Printf("arg block at %s, %v does not have a name label", filename, current.TypeRange)
					return out
				}
				ret.scopedLocalsByFile[filename] = append(ret.scopedLocalsByFile[filename], &scopedLocal{
					name:           current.Labels[0],
					schema:         &schema.AttributeSchema{Constraint: schema.Any{}},
					definition:     hcl.RangeBetween(current.TypeRange, current.OpenBraceRange),
					accessibleFrom: parent.Range(),
				})
			}
			return out
		})
	}
	ret.enhanceLocalSchemas(rootLocals, t.AsSchema(), func(e hclsyntax.Expression) []byte {
		f, ok := files[e.Range().Filename]
		if !ok {
			panic(fmt.Sprintf("file %s not found in files to get source", e.Range().Filename))
		}
		return f.Bytes
	})
	return ret
}

func (t *Targets) visibleTreeAt(parent *hcl.Block, file string, pos hcl.Pos) *Tree {
	addLocalsToTree := func(tree *Tree) {
		for _, loc := range t.scopedLocalsByFile[file] {
			if loc.accessibleFrom.ContainsPos(pos) {
				tree.add(&Node{
					Name:       loc.name,
					Schema:     loc.schema,
					Definition: loc.definition,
				})
			}
		}
	}
	switch {
	case parent == nil: // at top-level, nothing is accessible
		return newTree()

	// special case for functions where global references are not accessible
	case parent.Type == "function":
		funcTree := newTree()
		addLocalsToTree(funcTree)
		return funcTree

	default:
		ret := &Tree{
			root: &Node{
				Children: t.globals.root.Children,
			},
		}
		addLocalsToTree(ret)
		for _, res := range t.resourceByFile[file] {
			if res.accessibleFrom.ContainsPos(pos) {
				ret.add(&Node{
					Name: "name",
					Schema: &schema.AttributeSchema{
						Description: lang.PlainText("crossplane resource name"),
						Constraint:  schema.String{},
					},
					Definition: res.definition,
				}, "self")
				ret.add(&Node{
					Name: "connection",
					Schema: &schema.AttributeSchema{
						Description: lang.PlainText("resource connection details"),
						Constraint: schema.Map{
							Elem: schema.String{},
						},
					},
					Definition: res.definition,
				}, "self")
				ret.add(&Node{
					Name:       "resource",
					Schema:     res.schema,
					Definition: res.definition,
				}, "self")
			}
		}
		for _, res := range t.collectionByFile[file] {
			if res.accessibleFrom.ContainsPos(pos) {
				ret.add(&Node{
					Name: "basename",
					Schema: &schema.AttributeSchema{
						Description: lang.PlainText("base name of resource collection"),
						Constraint:  schema.String{},
					},
					Definition: res.definition,
				}, "self")
				ret.add(&Node{
					Name: "connections",
					Schema: &schema.AttributeSchema{
						Description: lang.PlainText("resource collection connection details"),
						Constraint: schema.List{
							Elem: schema.Map{
								Elem: schema.String{},
							},
						},
					},
					Definition: res.definition,
				}, "self")
				ret.add(&Node{
					Name: "resources",
					Schema: &schema.AttributeSchema{
						Description: lang.PlainText("resource collection details"),
						Constraint: schema.List{
							Elem: res.schema.Constraint,
						},
					},
					Definition: res.definition,
				}, "self")
			}
		}
		return ret
	}
}

func walk(body *hclsyntax.Body, bs schema.BlockStack, coll *localsCollection, callback func(bs schema.BlockStack, coll *localsCollection) *localsCollection) {
	for _, block := range body.Blocks {
		bs.Push(block)
		child := callback(bs, coll)
		walk(block.Body, bs, child, callback)
		bs.Pop()
	}
}

func (t *Targets) enhanceLocalSchemas(coll *localsCollection, globalSchema *schema.AttributeSchema, fileSource func(e hclsyntax.Expression) []byte) {
	coll.computeSchemas(globalSchema, fileSource)
	for k, v := range coll.locals {
		if v.enhancedSchema == unknownSchema {
			continue
		}
		// update the correct object to account for the schema
		if coll.parent == nil { // at root, look directly into the globals tree
			found := false
			for _, root := range t.globals.Roots() {
				if root.Name == k {
					found = true
					root.Schema = v.enhancedSchema
					break
				}
			}
			if !found {
				log.Printf("internal error: local %s not found in global tree", k)
			}
			continue
		}

		// not at root - find the local in the scoped structure.
		found := false
		for _, sl := range t.scopedLocalsByFile[coll.scopedTo.Filename] {
			if sl.accessibleFrom == coll.scopedTo {
				if sl.name == k {
					found = true
					sl.schema = v.enhancedSchema
					break
				}
			}
		}
		if !found {
			log.Printf("internal error: scoped local %s (%v) not found in scoped locals collection", k, coll.scopedTo)
		}
	}
	for _, child := range coll.children {
		t.enhanceLocalSchemas(child, globalSchema, fileSource)
	}
}
