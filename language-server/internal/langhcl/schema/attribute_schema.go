// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"errors"
	"fmt"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
)

// AttributeSchema describes schema for an attribute
type AttributeSchema struct {
	Description lang.MarkupContent
	IsRequired  bool
	IsOptional  bool
	// Constraint represents expression constraint e.g. what types of
	// expressions are expected for the attribute
	//
	// Constraints are immutable after construction by convention. It is
	// particularly important not to mutate a constraint after it has been
	// added to an AttributeSchema.
	Constraint Constraint
	// CompletionHooks represent any hooks which provide
	// additional completion candidates for the attribute.
	// These are typically candidates which cannot be provided
	// via schema and come from external APIs or other sources.
	CompletionHooks lang.CompletionHooks
}

func (as *AttributeSchema) Validate() error {
	if as.IsOptional && as.IsRequired {
		return errors.New("IsOptional or IsRequired must be set, not both")
	}
	if !as.IsRequired && !as.IsOptional {
		return errors.New("one of IsRequired or IsOptional must be set")
	}
	if con, ok := as.Constraint.(Validatable); ok {
		err := con.Validate()
		if err != nil {
			return fmt.Errorf("constraint: %T: %s", as.Constraint, err)
		}
	}
	return nil
}

func (as *AttributeSchema) Copy() *AttributeSchema {
	if as == nil {
		return nil
	}
	newAs := &AttributeSchema{
		IsRequired:      as.IsRequired,
		IsOptional:      as.IsOptional,
		Description:     as.Description,
		CompletionHooks: as.CompletionHooks.Copy(),
		// We do not copy Constraint as it should be immutable
		Constraint: as.Constraint,
	}
	return newAs
}
