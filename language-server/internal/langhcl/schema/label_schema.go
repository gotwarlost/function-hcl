// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
)

// LabelSchema describes schema for a label on a particular position
type LabelSchema struct {
	Name          string
	Description   lang.MarkupContent
	AllowedValues []string
}

func (ls *LabelSchema) CanComplete() bool {
	return len(ls.AllowedValues) > 0
}

func (ls *LabelSchema) Copy() *LabelSchema {
	if ls == nil {
		return nil
	}

	var v []string
	if ls.AllowedValues != nil {
		v = make([]string, len(ls.AllowedValues))
		copy(v, ls.AllowedValues)
	}
	return &LabelSchema{
		Name:          ls.Name,
		Description:   ls.Description,
		AllowedValues: v,
	}
}
