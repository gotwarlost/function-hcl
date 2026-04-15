package v1

import "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource"

const (
	StandardSourcesFile = ".crd-sources.yaml"
	DefaultSourcesDir   = ".crds"
)

// Scope represents a resource scope for processing CRDs.
type Scope = resource.Scope

const (
	ScopeNamespaced Scope = resource.ScopeNamespaced
	ScopeCluster    Scope = resource.ScopeCluster
	ScopeBoth       Scope = resource.ScopeBoth
)

// CRDSource provides a source for loading CRDs.
type CRDSource struct {
	Scope Scope    `json:"scope,omitempty"`
	Paths []string `json:"paths"`
}
