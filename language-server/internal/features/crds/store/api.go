// Package store provides a self-managing CRD store and implements background CRD discovery.
// It can provide the last known good state of a store related to a specific
// module at any time. Information can change in subsequent calls as more schemas are discovered.
package store

import (
	"context"
	"log"
	"sync"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/resource"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/queue"
)

// Store is the CRD store.
type Store struct {
	queue          *queue.Queue
	files          *fileStore
	sources        *sourceStore
	seenLock       sync.RWMutex
	seenDirs       map[string]string
	onNoCRDSources func(dir string)
}

// New creates a new store. The onNoCRDSources callback is called when
// no CRD sources are found for a directory. It may be nil.
func New(onNoCRDSources func(dir string)) *Store {
	return &Store{
		queue:          queue.New(1),
		files:          newFileStore(),
		sources:        newSourceStore(),
		seenDirs:       map[string]string{},
		onNoCRDSources: onNoCRDSources,
	}
}

// Start starts background processing of the store that ends when the
// supplied context is canceled.
func (s *Store) Start(ctx context.Context) {
	s.queue.Start(ctx)
}

// RegisterOpenDir registers a module directory as a candidate for schema discovery.
func (s *Store) RegisterOpenDir(modulePath string) {
	s.registerOpenDir(modulePath)
}

// GetSchema returns the known schema related to the specified module directory.
func (s *Store) GetSchema(modulePath string) *resource.Schemas {
	s.seenLock.RLock()
	defer s.seenLock.RUnlock()
	sourcePath, ok := s.seenDirs[modulePath]
	if !ok {
		log.Println("internal error: directory", modulePath, "not registered")
		return emptySchema
	}
	si := s.sources.get(sourcePath)
	if si == nil {
		log.Println("warn: source directory", sourcePath, "not found")
		return emptySchema
	}
	return si.schema
}

// ProcessNewDir reprocesses the cache when a new directory is created, since it may
// be a new source root that the user introduced.
func (s *Store) ProcessNewDir(path string) {
	s.reprocess() // TODO: make this more intelligent
}

// ProcessFile processes an added or changed file.
func (s *Store) ProcessFile(path string) {
	if shouldProcessFile(path) {
		s.reprocess()
	}
}

// ProcessPathDeletion processes file or directory deletion events at the supplied path.
func (s *Store) ProcessPathDeletion(path string) {
	s.reprocess() // TODO: make this more intelligent
}
