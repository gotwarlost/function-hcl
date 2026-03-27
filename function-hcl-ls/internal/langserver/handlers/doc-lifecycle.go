// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/eventbus"
	ilsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/lsp"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/utils/uri"
)

func (svc *service) textDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	docURI := string(params.TextDocument.URI)

	// URIs are always checked during initialize request, but
	// we still allow single-file mode, therefore invalid URIs
	// can still land here, so we check for those.
	if !uri.IsURIValid(docURI) {
		_ = jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("Ignoring workspace folder (unsupport or invalid URI) %s."+
				" This is most likely bug, please report it.", docURI),
		})
		return fmt.Errorf("invalid URI: %s", docURI)
	}

	dh := document.HandleFromURI(docURI)
	err := svc.docStore.Open(dh, params.TextDocument.LanguageID,
		int(params.TextDocument.Version), []byte(params.TextDocument.Text))
	if err != nil {
		return err
	}

	svc.eventBus.PublishOpenEvent(eventbus.OpenEvent{
		Doc:        dh,
		LanguageID: params.TextDocument.LanguageID,
	})
	return nil
}

func (svc *service) textDocumentDidChange(ctx context.Context, params lsp.DidChangeTextDocumentParams) error {
	p := lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: params.TextDocument.URI,
			},
			Version: params.TextDocument.Version,
		},
		ContentChanges: params.ContentChanges,
	}

	dh := ilsp.HandleFromDocumentURI(p.TextDocument.URI)
	doc, err := svc.docStore.Get(dh)
	if err != nil {
		svc.logger.Println("GetDocument error:", err)
		return err
	}

	newVersion := int(p.TextDocument.Version)
	// Versions don't have to be consecutive, but they must be increasing
	if newVersion <= doc.Version {
		svc.logger.Printf("Old document version (%d) received, current version is %d. "+
			"Ignoring this update for %s. This is likely a client bug, please report it.",
			newVersion, doc.Version, p.TextDocument.URI)
		return nil
	}

	changes := ilsp.DocumentChanges(params.ContentChanges)
	newText, err := document.ApplyChanges(doc.Text, changes)
	if err != nil {
		svc.logger.Println("ApplyChanges error:", err)
		return err
	}
	err = svc.docStore.Update(dh, newText, newVersion)
	if err != nil {
		svc.logger.Println("updateDocument error:", err)
		return err
	}

	svc.eventBus.PublishEditEvent(eventbus.EditEvent{
		Doc:        dh,
		LanguageID: doc.LanguageID,
	})
	return nil
}

func (svc *service) textDocumentDidSave(ctx context.Context, params lsp.DidSaveTextDocumentParams) error {
	// TODO: maybe implement validate on save
	return nil
}

func (svc *service) textDocumentDidClose(ctx context.Context, params lsp.DidCloseTextDocumentParams) error {
	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	return svc.docStore.Close(dh)
}
