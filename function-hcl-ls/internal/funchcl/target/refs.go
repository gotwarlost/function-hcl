package target

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func hasLocals(b *hclsyntax.Block) bool {
	if b.Body == nil {
		return false
	}
	for _, block := range b.Body.Blocks {
		if block.Type == "locals" {
			return true
		}
	}
	return false
}

func findDefinition(tree *Tree, pathElements []string) *hcl.Range {
	rootName := pathElements[0]
	var current *Node
	rest := pathElements[1:]
	for _, root := range tree.Roots() {
		if root.Name == rootName {
			current = root
			break
		}
	}
	if current == nil {
		return nil
	}
	for _, pathElement := range rest {
		found := false
		for _, node := range current.Children {
			if node.Name == pathElement {
				found = true
				current = node
				break
			}
		}
		if !found {
			break
		}
	}
	return &current.Definition
}

func walkRefs(body *hclsyntax.Body, fileContent []byte, targets *Targets, visibleTree *Tree, p *ReferenceMap) {
	if body == nil {
		return
	}
	for _, attr := range body.Attributes {
		_ = hclsyntax.VisitAll(attr.Expr, func(node hclsyntax.Node) hcl.Diagnostics {
			expr, ok := node.(*hclsyntax.ScopeTraversalExpr)
			if !ok {
				return nil
			}
			if expr.Traversal.IsRelative() { // don't know how to process relative traversals
				return nil
			}
			var pathElements []string
			for _, traverser := range expr.Traversal {
				nameBytes := traverser.SourceRange().SliceBytes(fileContent)
				pathElements = append(pathElements, extractTraversalIdentifier(nameBytes))
			}
			defRange := findDefinition(visibleTree, pathElements)
			if defRange != nil {
				r := *defRange
				p.RefsToDef[(expr.Range())] = r
				p.DefToRefs[r] = append(p.DefToRefs[r], expr.Range())
			}
			return nil
		})
	}
	getVisibleTree := func(b *hclsyntax.Block) *Tree {
		return targets.VisibleTreeAt(b.AsHCLBlock(), b.TypeRange.Filename, b.TypeRange.End)
	}
	for _, block := range body.Blocks {
		if block.Body == nil {
			continue
		}
		childTree := visibleTree
		switch {
		// when new scopes are introduced, recalculate the visible tree
		case block.Type == "resource" || block.Type == "resources" || block.Type == "template" || block.Type == "function":
			childTree = getVisibleTree(block)
		// locals introduced new scoped variables
		case hasLocals(block):
			childTree = getVisibleTree(block)
		}
		walkRefs(block.Body, fileContent, targets, childTree, p)
	}
}

func extractTraversalIdentifier(bytes []byte) string {
	s := string(bytes)
	if strings.HasPrefix(s, ".") {
		s = s[1:]
	} else if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
	}
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s = s[1 : len(s)-1]
	}
	return s
}

func buildReferenceMap(files map[string]*hcl.File, targets *Targets) *ReferenceMap {
	p := &ReferenceMap{
		DefToRefs: DefToRefs{},
		RefsToDef: RefsToDef{},
	}
	for _, r := range targets.declarationRanges {
		p.DefToRefs[r] = nil // ensure key for all definitions
	}
	for _, f := range files {
		// start with the global visible tree
		visibleTree := targets.globals
		body := f.Body.(*hclsyntax.Body)
		walkRefs(body, f.Bytes, targets, visibleTree, p)
	}
	return p
}
