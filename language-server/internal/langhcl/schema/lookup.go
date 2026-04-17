package schema

import (
	"log"

	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var sentinel = &hclsyntax.Block{}

type blockStack struct {
	blocks []*hclsyntax.Block
}

func (s *blockStack) HasAncestorOfType(t string) bool {
	for _, block := range s.blocks {
		if block.Type == t {
			return true
		}
	}
	return false
}

func (s *blockStack) Push(block *hclsyntax.Block) {
	s.blocks = append(s.blocks, block)
}

func (s *blockStack) Pop() {
	if len(s.blocks) == 0 {
		log.Println("error: pop stack at 0 depth")
		return
	}
	s.blocks = s.blocks[:len(s.blocks)-1]
}

func (s *blockStack) Peek(n int) *hclsyntax.Block {
	index := len(s.blocks) - 1 - n
	if index < 0 {
		return sentinel
	}
	return s.blocks[index]
}

// BlockStack provides a mechanism to track the current block structure that is seen
// in LIFO manner. Peek and Pop never panic and always return an empty block for
// out of bounds calls.
type BlockStack interface {
	Push(*hclsyntax.Block)
	Peek(n int) *hclsyntax.Block
	Pop()
	HasAncestorOfType(t string) bool
}

func NewBlockStack() BlockStack {
	return &blockStack{}
}

// Lookup provides a mechanism to provide just-in-time schema information for
// an entity at a specific position.
type Lookup interface {
	// BodySchema returns the body schema at the current block position.
	BodySchema(bs BlockStack) *BodySchema
	// LabelSchema returns the label schema at the current block position.
	LabelSchema(bs BlockStack) []*LabelSchema
	// AttributeSchema returns the attribute schema at the current block and attribute position.
	AttributeSchema(bs BlockStack, attrName string) *AttributeSchema
	// Functions returns available function signatures keyed by function name.
	Functions() map[string]FunctionSignature
	// ImpliedAttributeSchema returns a computed schema implied by the attribute expression.
	// This is typically called when hovering over the name of an attribute.
	ImpliedAttributeSchema(bs BlockStack, attrName string) *AttributeSchema
}
