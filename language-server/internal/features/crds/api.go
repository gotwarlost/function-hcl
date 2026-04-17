// Package crds provides schema information for CRDs and XRDs that are used in function-hcl compositions.
// For each open document, it tries and discovers CRD information that the user has captured in a set of files
// and provides dynamic schemas for these types.
package crds

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/eventbus"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/features/crds/store"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/resource"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/logging"
)

// Config configures the feature.
type Config struct {
	EventBus *eventbus.EventBus
}

// CRDs provides schemas for CRDs and XRDs.
type CRDs struct {
	bus     *eventbus.EventBus
	store   *store.Store
	logger  *log.Logger
	schemas atomic.Pointer[resource.Schemas]
}

// New creates an instance of the CRD discovery feature.
func New(c Config) *CRDs {
	ret := &CRDs{
		bus:    c.EventBus,
		logger: logging.LoggerFor(logging.ModuleCRDs),
	}
	ret.store = store.New(func(dir string) {
		c.EventBus.PublishNoCRDSourcesEvent(eventbus.NoCRDSourcesEvent{Dir: dir})
	})
	ret.schemas.Store(resource.ToSchemas())
	return ret
}

// Start starts background event processing. The background routine terminates
// when the context is canceled.
func (c *CRDs) Start(ctx context.Context) {
	c.start(ctx)
}

// DynamicSchemas returns the schemas loaded for the supplied module directory path.
func (c *CRDs) DynamicSchemas(path string) *resource.Schemas {
	return c.store.GetSchema(path)
}
