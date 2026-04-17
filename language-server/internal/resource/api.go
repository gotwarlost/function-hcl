// Package resource provides facilities to convert CRDs and XRDs to HCL schemas.
package resource

import (
	"fmt"
	"log"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/schema"
	xpv1 "github.com/crossplane/crossplane/v2/apis/apiextensions/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Scope is the scope for which CRDs are loaded. This can be "cluster", "namespaced" or
// "both".
type Scope string

const (
	ScopeNamespaced Scope = "namespaced"
	ScopeCluster    Scope = "cluster"
	ScopeBoth       Scope = "both"
)

// Key is the key of a K8s object.
type Key struct {
	ApiVersion string
	Kind       string
}

func (k Key) String() string {
	return fmt.Sprintf("%s %s", k.Kind, k.ApiVersion)
}

// ScopedAttributeSchema wraps an attribute schema and also tracks the scope for the top-level objects.
type ScopedAttributeSchema struct {
	schema.AttributeSchema
	Scope Scope
}

// Schemas maintains attribute schemas for a set of K8s types.
type Schemas struct {
	s map[Key]*ScopedAttributeSchema
	l []Key
}

// Keys returns the keys known to this instance.
func (s *Schemas) Keys() []Key { return s.l }

// Schema returns the attribute schema for the specified API version and kind,
// or nil if a schema for this key could not be found.
func (s *Schemas) Schema(apiVersion, kind string) *schema.AttributeSchema {
	as := s.s[Key{ApiVersion: apiVersion, Kind: kind}]
	if as == nil {
		return nil
	}
	return &as.AttributeSchema
}

// FilterScope returns schemas filtered by scope.
func (s *Schemas) FilterScope(scope Scope) *Schemas {
	if scope != ScopeNamespaced && scope != ScopeCluster {
		return s
	}
	m := map[Key]*ScopedAttributeSchema{}
	for k, v := range s.s {
		if v.Scope == scope {
			m[k] = v
		}
	}
	return newSchemas(m)
}

// ToSchemas converts CRDs and XRDs present in the supplied objects into a Schemas instance.
func ToSchemas(objects ...runtime.Object) *Schemas {
	ret := map[Key]*ScopedAttributeSchema{}
	for _, o := range objects {
		switch c := o.(type) {
		case *v1.CustomResourceDefinition:
			aggregateCRDToMap(c, ret)
		case *xpv1.CompositeResourceDefinition:
			aggregateXRDToMap(c, ret)
		}
	}
	return newSchemas(ret)
}

// Compose composes the supplied schemas into a singular instance.
func Compose(schemas ...*Schemas) *Schemas {
	out := map[Key]*ScopedAttributeSchema{}
	for _, s := range schemas {
		for k, v := range s.s {
			_, seen := out[k]
			if seen {
				log.Printf("multiple schemas found for %v", k)
				continue
			}
			out[k] = v
		}
	}
	return newSchemas(out)
}
