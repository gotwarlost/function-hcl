// Package target provides definition and reference information for a function-hcl module.
package target

import (
	ourschema "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/schema"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
)

// Node represents a specific leaf or intermediate target in a target tree.
type Node struct {
	Name        string                  // the name using which the node is referenced
	IsContainer bool                    // container node that has an implied schema of "any object"
	Schema      *schema.AttributeSchema // the schema for non-container nodes
	NameRange   hcl.Range               // the range where the name of the node is declared
	Definition  hcl.Range               // the range where the node is defined
	Children    []*Node                 // the child nodes of this one.
}

func attrsForNodes(nodes []*Node) map[string]*schema.AttributeSchema {
	attrs := map[string]*schema.AttributeSchema{}
	for _, child := range nodes {
		attrs[child.Name] = child.AsSchema()
	}
	return attrs
}

func (n *Node) AsSchema() *schema.AttributeSchema {
	if !n.IsContainer {
		return n.Schema
	}
	return &schema.AttributeSchema{
		Constraint: schema.Object{
			Name:       n.Name,
			Attributes: attrsForNodes(n.Children),
		},
	}
}

// Tree is a tree of nodes accessible in a given scope.
type Tree struct {
	root *Node // a synthetic container for the actual, multiple, roots of the tree
}

func newTree(roots ...*Node) *Tree {
	return &Tree{
		root: &Node{
			Children: roots,
		},
	}
}

// Roots returns the roots of the tree. These are the "top-level variables"
// like locals, `req`, `self`, `each` etc.
func (t *Tree) Roots() []*Node {
	return t.root.Children
}

// AsSchema returns the contents of the tree as an attribute schema of a top-level
// object that represents the tree as a whole.
func (t *Tree) AsSchema() *schema.AttributeSchema {
	return &schema.AttributeSchema{
		Constraint: schema.Object{
			Attributes: attrsForNodes(t.Roots()),
		},
	}
}

// Targets provides a mechanism to find all accessible symbols at a given position.
// It contains a global tree for references that are accessible globally (e.g.
// file-scoped variables, req.composite etc.) and stores extra information for
// block scoped locals, self aliases for resources and collections. Given a position in a
// file, these extras allow construction of a "visible" tree that contains all
// accessible references from that position.
type Targets struct {
	CompositeSchema    *schema.AttributeSchema   // the schema used for the composite or nil
	globals            *Tree                     // the global tree visible from any position
	scopedLocalsByFile map[string][]*scopedLocal // scoped locals accessible in a file range
	resourceByFile     map[string][]*alias       // the meaning of self.resource within a range
	collectionByFile   map[string][]*alias       // the meaning of self.resources within a range
	declarationRanges  []hcl.Range               // ranges where resources are declared
}

// VisibleTreeAt returns a tree that contains all references visible from the supplied
// position in a specific file.
func (t *Targets) VisibleTreeAt(parent *hcl.Block, file string, pos hcl.Pos) *Tree {
	return t.visibleTreeAt(parent, file, pos)
}

type (
	DefToRefs map[hcl.Range][]hcl.Range // map of definition ranges to multiple reference ranges
	RefsToDef map[hcl.Range]hcl.Range   // maps of reference ranges to definition ranges
)

// ReferenceMap provides mappings between definitions and references in both directions.
type ReferenceMap struct {
	DefToRefs DefToRefs
	RefsToDef RefsToDef
}

// FindDefinitionFromReference returns the definition range for a reference range that includes
// the supplied position.
func (p *ReferenceMap) FindDefinitionFromReference(filename string, pos hcl.Pos) *hcl.Range {
	for ref, def := range p.RefsToDef {
		if ref.Filename == filename && ref.ContainsPos(pos) {
			if def.Filename == "" { // pseudo-range that doesn't have a definition
				return nil
			}
			return &def
		}
	}
	// account for when people try to find a definition when the cursor is on the definition itself
	for def := range p.DefToRefs {
		if def.Filename == filename && def.ContainsPos(pos) {
			return &def
		}
	}
	return nil
}

// FindReferencesFromDefinition returns reference ranges for a definition range that includes
// the supplied position.
func (p *ReferenceMap) FindReferencesFromDefinition(filename string, pos hcl.Pos) []hcl.Range {
	for ref, def := range p.DefToRefs {
		if ref.Filename == filename && ref.ContainsPos(pos) {
			return def
		}
	}
	return nil
}

// BuildTargets builds targets for a module.
func BuildTargets(files map[string]*hcl.File, dyn ourschema.DynamicLookup, compositeSchema *schema.AttributeSchema) *Targets {
	return buildTargets(files, dyn, compositeSchema)
}

// BuildReferenceMap builds the reference map for a module.
func BuildReferenceMap(files map[string]*hcl.File, targets *Targets) *ReferenceMap {
	return buildReferenceMap(files, targets)
}

// SchemaForRelativeTraversal returns the child schema implied by the root schema and the supplied traversal.
// The traversal does not have to be a relative traversal; an absolute one is also ok.
// It returns an unknown schema if a schema could not be found.
func SchemaForRelativeTraversal(root *schema.AttributeSchema, traversal hcl.Traversal) (ret *schema.AttributeSchema) {
	s := processRelativeTraversal(root, traversal)
	if s == nil {
		return unknownSchema
	}
	return s
}

// SubSchema returns a known schema at the supplied path relative to the supplied root schema.
// It returns an unknown schema if one could not be found.
func SubSchema(s *schema.AttributeSchema, path ...string) *schema.AttributeSchema {
	s = subSchema(s, path...)
	if s == nil {
		return unknownSchema
	}
	return s
}
