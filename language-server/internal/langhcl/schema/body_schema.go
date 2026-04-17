// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
)

// BasicBlockSchema provides a block description and associated label schemas.
type BasicBlockSchema struct {
	Description   lang.MarkupContent
	Labels        []*LabelSchema
	AllowMultiple bool
}

func (b *BasicBlockSchema) Copy() *BasicBlockSchema {
	if b == nil {
		return nil
	}
	newBs := &BasicBlockSchema{
		Description: b.Description,
	}
	if len(b.Labels) > 0 {
		newBs.Labels = make([]*LabelSchema, len(b.Labels))
		for i, l := range b.Labels {
			newBs.Labels[i] = l.Copy()
		}
	}
	return newBs
}

// BodySchema describes the schema for a block
type BodySchema struct {
	Description  lang.MarkupContent
	Attributes   map[string]*AttributeSchema
	NestedBlocks map[string]*BasicBlockSchema
}

func (bs *BodySchema) Copy() *BodySchema {
	if bs == nil {
		return nil
	}
	newBs := &BodySchema{
		Description: bs.Description,
	}
	if bs.NestedBlocks != nil {
		newBs.NestedBlocks = make(map[string]*BasicBlockSchema, len(bs.NestedBlocks))
		for k, v := range bs.NestedBlocks {
			newBs.NestedBlocks[k] = v.Copy()
		}
	}
	if bs.Attributes != nil {
		newBs.Attributes = make(map[string]*AttributeSchema, len(bs.Attributes))
		for k, v := range bs.Attributes {
			newBs.Attributes[k] = v
		}
	}
	return newBs
}
