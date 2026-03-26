package handlers

import (
	"context"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/eventbus"
	ilsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/lsp"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
)

func (svc *service) startDiagnosticsPublisher(ctx context.Context) {
	diagEvents := svc.eventBus.SubscribeToDiagnosticsEvents("handlers.diagnostics")
	go func() {
		for {
			select {
			case event := <-diagEvents:
				svc.publishDiagnostics(ctx, event)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (svc *service) publishDiagnostics(ctx context.Context, event eventbus.DiagnosticsEvent) {
	uri := lsp.DocumentURI(event.Doc.FullURI())
	diags := ilsp.HCLDiagsToLSP(event.Diags, "function-hcl")
	err := svc.server.Notify(ctx, "textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
	if err != nil {
		svc.logger.Printf("failed to publish diagnostics: %s", err)
	}
}
