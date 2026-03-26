package resource

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"

	ourschema "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/schema"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/schema"
	xpv1 "github.com/crossplane/crossplane/v2/apis/apiextensions/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// newSchemas returns a schemas instance given existing mappings.
func newSchemas(m map[Key]*ScopedAttributeSchema) *Schemas {
	s := &Schemas{s: m, l: make([]Key, 0, len(m))}
	for k := range m {
		s.l = append(s.l, k)
	}
	sort.Slice(s.l, func(i, j int) bool {
		if s.l[i].Kind == s.l[j].Kind {
			return s.l[i].ApiVersion < s.l[j].ApiVersion
		}
		return s.l[i].Kind < s.l[j].Kind
	})
	return s
}

func aggregateCRDToMap(crd *v1.CustomResourceDefinition, ret map[Key]*ScopedAttributeSchema) {
	group := crd.Spec.Group
	kind := crd.Spec.Names.Kind
	for _, versionDef := range crd.Spec.Versions {
		version := versionDef.Name
		key := Key{ApiVersion: group + "/" + version, Kind: kind}
		attrSchema := toAttributeSchema(versionDef.Schema.OpenAPIV3Schema, true)
		scope := ScopeCluster
		if crd.Spec.Scope == v1.NamespaceScoped {
			scope = ScopeNamespaced
		}
		typeName := fmt.Sprintf("%s:%s/%s", kind, group, version)
		ret[key] = &ScopedAttributeSchema{
			Scope: scope,
			AttributeSchema: schema.AttributeSchema{
				Description: lang.PlainText(versionDef.Schema.OpenAPIV3Schema.Description),
				Constraint: schema.Object{
					Attributes:  attrSchema,
					Name:        typeName,
					Description: lang.PlainText(versionDef.Schema.OpenAPIV3Schema.Description),
				},
			},
		}
	}
}

func aggregateXRDToMap(crd *xpv1.CompositeResourceDefinition, ret map[Key]*ScopedAttributeSchema) {
	group := crd.Spec.Group
	kind := crd.Spec.Names.Kind
	for _, versionDef := range crd.Spec.Versions {
		version := versionDef.Name
		key := Key{ApiVersion: group + "/" + version, Kind: kind}
		var props v1.JSONSchemaProps
		err := json.Unmarshal(versionDef.Schema.OpenAPIV3Schema.Raw, &props)
		if err != nil {
			log.Printf("unexpected error unmarshalling xrd: %v", err)
			return
		}
		scope := ScopeCluster
		if crd.Spec.Scope != nil && *crd.Spec.Scope == xpv1.CompositeResourceScopeNamespaced {
			scope = ScopeNamespaced
		}
		attrSchema := toAttributeSchema(&props, true)
		typeName := fmt.Sprintf("%s:%s/%s", kind, group, version)
		ret[key] = &ScopedAttributeSchema{
			Scope: scope,
			AttributeSchema: schema.AttributeSchema{
				Description: lang.PlainText(props.Description),
				Constraint: schema.Object{
					Attributes:  attrSchema,
					Name:        typeName,
					Description: lang.PlainText(""),
				},
			},
		}
	}
}

func isRequired(requiredProps []string, name string) bool {
	for _, prop := range requiredProps {
		if prop == name {
			return true
		}
	}
	return false
}

var k8sSchema = ourschema.BasicK8sObjectConstraint()

func toAttributeSchema(obj *v1.JSONSchemaProps, topLevel bool) schema.ObjectAttributes {
	if obj == nil || obj.Type != "object" {
		panic("invalid call to toAttributeSchema")
	}
	attrs := schema.ObjectAttributes{}
	if topLevel {
		for name, attr := range k8sSchema.Attributes {
			attrs[name] = attr
		}
	}
	for name, props := range obj.Properties {
		// prefer pre-filled attribute defs rather than using the ones supplied
		if _, ok := attrs[name]; ok {
			continue
		}
		required := isRequired(obj.Required, name)
		def := &schema.AttributeSchema{
			Description: lang.PlainText(props.Description),
			IsRequired:  required,
			IsOptional:  !required,
			// Default: TODO
			Constraint: constraintFor(&props),
		}
		attrs[name] = def
	}
	return attrs
}

var defaultConstraint = schema.String{}

func constraintFor(props *v1.JSONSchemaProps) schema.Constraint {
	if props == nil {
		log.Println("no props!")
		return defaultConstraint
	}
	switch props.Type {
	case "object":
		if len(props.Properties) == 0 && props.AdditionalProperties != nil && props.AdditionalProperties.Schema != nil {
			return schema.Map{Elem: constraintFor(props.AdditionalProperties.Schema)}
		}
		attrs := toAttributeSchema(props, false)
		return schema.Object{
			Attributes:            attrs,
			AllowInterpolatedKeys: true,
		}
	case "array":
		childConstraint := constraintFor(props.Items.Schema)
		return schema.List{Elem: childConstraint}
	case "string":
		return schema.String{}
	case "integer", "number":
		return schema.Number{}
	case "boolean":
		return schema.Bool{}
	}
	return defaultConstraint
}
