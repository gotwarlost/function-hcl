package store

import (
	"sync"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/resource"
	types "github.com/crossplane-contrib/function-hcl/function-hcl-ls/types/v1"
)

// sourceInfo tracks the set of files and the schema for a single CRD source
type sourceInfo struct {
	sourcePath    string
	source        *types.CRDSourceRuntime
	expandedFiles map[string]bool
	schema        *resource.Schemas
}

func (s *sourceInfo) copy() *sourceInfo {
	if s == nil {
		return nil
	}
	m := make(map[string]bool)
	for k, v := range s.expandedFiles {
		m[k] = v
	}
	c := *s.source
	return &sourceInfo{
		sourcePath:    s.sourcePath,
		source:        &c,
		expandedFiles: m,
		schema:        s.schema,
	}
}

type sourceStore struct {
	l       sync.RWMutex
	sources map[string]*sourceInfo
}

func newSourceStore() *sourceStore {
	return &sourceStore{
		sources: map[string]*sourceInfo{},
	}
}

func (s *sourceStore) get(path string) *sourceInfo {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.sources[path].copy()
}

func (s *sourceStore) put(ss *sourceInfo) {
	s.l.Lock()
	defer s.l.Unlock()
	s.sources[ss.sourcePath] = ss.copy()
}

func (s *sourceStore) list() []string {
	s.l.RLock()
	defer s.l.RUnlock()
	var stores []string
	for k := range s.sources {
		stores = append(stores, k)
	}
	return stores
}
