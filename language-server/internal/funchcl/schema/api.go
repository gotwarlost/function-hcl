// Package schema provides the standard schema for the function-hcl DSL.
package schema

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

const LanguageHCL = "hcl"

// DynamicLookup provides a schema for the supplied API version and kind.
type DynamicLookup interface {
	Schema(apiVersion, kind string) *schema.AttributeSchema
}

// LocalsAttributeLookup is an optional interface that can be implemented by DynamicLookup
// to dynamically figure out schemas for local variables based on how they are assigned.
// (for example: foo = req.composite.metadata.name => foo is of type string)
type LocalsAttributeLookup interface {
	LocalSchema(name string) *schema.AttributeSchema
}

// CompositeSchemaLookup is an optional interface that can be implemented by DynamicLookup
// to dynamically figure out the schema for the composite.
type CompositeSchemaLookup interface {
	CompositeSchema() *schema.AttributeSchema
}

// New returns a schema.Lookup instance.
func New(dyn DynamicLookup) schema.Lookup {
	return &lookup{dyn: dyn}
}

// BasicK8sObjectConstraint returns a constraint for a generic K8s object.
func BasicK8sObjectConstraint() schema.Object {
	return basicK8sObjectSchema()
}

// DependentSchemaOrDefault returns an available schema for a body attribute or a default.
func DependentSchemaOrDefault(dyn DynamicLookup, bodyBlock *hclsyntax.Block) *schema.AttributeSchema {
	ret, ok := dependentSchema(dyn, bodyBlock)
	if !ok {
		return basicBodyAttributeSchema()
	}
	return ret
}

func withAttributes(cons schema.Object, attrs map[string]*schema.AttributeSchema) schema.Object {
	return schema.Object{
		Name:                  cons.Name,
		Attributes:            attrs,
		Description:           cons.Description,
		AllowInterpolatedKeys: cons.AllowInterpolatedKeys,
		AnyAttribute:          cons.AnyAttribute,
	}
}

// WithoutStatus returns an attribute schema that eliminates the `status` property
// if one is present.
func WithoutStatus(aSchema *schema.AttributeSchema) *schema.AttributeSchema {
	if aSchema == nil {
		return nil
	}
	cons, ok := aSchema.Constraint.(schema.Object)
	if !ok {
		return aSchema
	}
	_, hasStatus := cons.Attributes["status"]
	if !hasStatus {
		return aSchema
	}
	aSchema = aSchema.Copy()
	attrs := make(map[string]*schema.AttributeSchema, len(cons.Attributes))
	for k, v := range cons.Attributes {
		if k == "status" {
			continue
		}
		attrs[k] = v
	}
	aSchema.Constraint = withAttributes(cons, attrs)
	return aSchema
}

// WithStatusOnly returns an attribute schema that only has the `status` property
// if one is present. Otherwise, the schema is returned as-is.
func WithStatusOnly(aSchema *schema.AttributeSchema) *schema.AttributeSchema {
	if aSchema == nil {
		return nil
	}
	cons, ok := aSchema.Constraint.(schema.Object)
	if !ok {
		return aSchema
	}
	_, hasStatus := cons.Attributes["status"]
	if !hasStatus {
		return aSchema
	}
	aSchema = aSchema.Copy()
	aSchema.Constraint = withAttributes(cons, map[string]*schema.AttributeSchema{
		"status": cons.Attributes["status"],
	})
	return aSchema
}

// WithoutAPIVersionAndKind returns an attribute schema that eliminates the `apiVersion`
// and `kind` properties, if present.
func WithoutAPIVersionAndKind(aSchema *schema.AttributeSchema) *schema.AttributeSchema {
	if aSchema == nil {
		return nil
	}
	cons, ok := aSchema.Constraint.(schema.Object)
	if !ok {
		return aSchema
	}
	aSchema = aSchema.Copy()
	attrs := make(map[string]*schema.AttributeSchema, len(cons.Attributes))
	for k, v := range cons.Attributes {
		if k == "apiVersion" || k == "kind" {
			continue
		}
		attrs[k] = v
	}
	aSchema.Constraint = withAttributes(cons, attrs)
	return aSchema
}
