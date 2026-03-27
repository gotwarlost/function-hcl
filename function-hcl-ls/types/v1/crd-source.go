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

// CRDSourceRuntime is used by the language server at runtime to
// load schemas from filesystem paths.
type CRDSourceRuntime struct {
	Scope Scope    `json:"scope,omitempty"`
	Paths []string `json:"paths"`
}

// CRDSourceOffline is the section used by tooling to download
// CRDs from remote packages to the local filesystem.
type CRDSourceOffline struct {
	CacheDir string   `json:"cache-dir"`
	Images   []string `json:"images"`
}

// CRDSource provides a source for loading CRDs.
type CRDSource struct {
	Runtime CRDSourceRuntime `json:"runtime"`
	Offline CRDSourceOffline `json:"offline"`
}
