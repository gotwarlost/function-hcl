package target

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

var unknownSchema = &schema.AttributeSchema{Constraint: schema.Any{}}

// subSchema returns the sub-schema for the supplied traversal if possible, or nil otherwise.
func subSchema(s *schema.AttributeSchema, path ...string) *schema.AttributeSchema {
	if s == nil {
		return nil
	}
	if len(path) == 0 {
		return s
	}
	objCons, ok := s.Constraint.(schema.Object)
	if !ok {
		return nil
	}
	sub, ok := objCons.Attributes[path[0]]
	if !ok {
		return nil
	}
	return subSchema(sub, path[1:]...)
}

// processRelativeTraversal returns the child schema implied by the root schema and the supplied traversal.
// The traversal does not have to be a relative traversal; an absolute one is also ok.
// It returns nil if a schema could not be found.
func processRelativeTraversal(root *schema.AttributeSchema, traversal hcl.Traversal) (ret *schema.AttributeSchema) {
	defer func() {
		if ret == nil {
			ret = unknownSchema
		}
	}()

outer:
	for _, child := range traversal {
		if root == nil {
			break
		}
		// if current is a map or a list, then indexing it further just yields the element type
		switch cons := root.Constraint.(type) {
		case schema.Map:
			root = &schema.AttributeSchema{Constraint: cons.Elem}
			continue outer
		case schema.List:
			root = &schema.AttributeSchema{Constraint: cons.Elem}
			continue outer
		}
		if _, ok := root.Constraint.(schema.Object); !ok {
			return nil
		}
		switch child := child.(type) {
		case hcl.TraverseRoot: // root is allowed
			root = subSchema(root, child.Name)
		case hcl.TraverseAttr:
			root = subSchema(root, child.Name)
		case hcl.TraverseIndex:
			if child.Key.Type() != cty.String {
				return nil
			}
			root = subSchema(root, child.Key.AsString())
		}
	}
	return root
}
