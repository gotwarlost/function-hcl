// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"log"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document"
	docstore "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document/store"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/eventbus"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/features/crds"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/features/modules"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/filesystem"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	ilsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/lsp"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/session"
)

type features struct {
	modules *modules.Modules
	crds    *crds.CRDs
}

type service struct {
	version string
	cc      *lsp.ClientCapabilities
	logger  *log.Logger

	sessCtx     context.Context
	stopSession context.CancelFunc

	docStore *docstore.Store
	fs       *filesystem.Filesystem
	server   session.Server
	eventBus *eventbus.EventBus
	features *features
}

func (svc *service) configureSessionDependencies() error {
	if svc.docStore == nil {
		svc.docStore = docstore.New()
	}

	if svc.fs == nil {
		svc.fs = filesystem.New(svc.docStore)
	}

	if svc.eventBus == nil {
		svc.eventBus = eventbus.New()
	}

	if svc.features == nil {
		c := crds.New(crds.Config{
			EventBus: svc.eventBus,
		})
		c.Start(svc.sessCtx)

		m, err := modules.New(modules.Config{
			EventBus: svc.eventBus,
			DocStore: svc.docStore,
			FS:       svc.fs,
			Provider: func(path string) modules.DynamicSchemas {
				return c.DynamicSchemas(path)
			},
		})
		if err != nil {
			return err
		}
		m.Start(svc.sessCtx)

		svc.features = &features{
			modules: m,
			crds:    c,
		}
	}
	svc.startDiagnosticsPublisher(svc.sessCtx)
	svc.startCRDNotificationHandler(svc.sessCtx)
	return nil
}

func (svc *service) standardInit(_ context.Context, uri lsp.DocumentURI) (_ *document.Document, p lang.Path, _ error) {
	dh := ilsp.HandleFromDocumentURI(uri)
	doc, err := svc.docStore.Get(dh)
	if err != nil {
		return nil, p, err
	}
	svc.features.modules.WaitUntilProcessed(dh.Dir.Path())
	p = lang.Path{
		Path:       doc.Dir.Path(),
		LanguageID: doc.LanguageID,
	}
	return doc, p, nil
}
