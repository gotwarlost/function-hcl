package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/creachadair/jrpc2"
	rpch "github.com/creachadair/jrpc2/handler"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/session"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/logging"
	"github.com/pkg/errors"
)

const requestCancelled jrpc2.Code = -32800

func (svc *service) getDispatchTable(s *session.Lifecycle) (rpch.Map, error) {
	dispatchTo := func(to any) func(_ context.Context, _ *jrpc2.Request) (any, error) {
		return func(ctx context.Context, req *jrpc2.Request) (any, error) {
			err := s.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			return handle(ctx, req, to)
		}
	}
	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := s.Initialize(req)
			if err != nil {
				return nil, err
			}
			return handle(ctx, req, svc.initialize)
		},
		"initialized": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := s.ConfirmInitialization(req)
			if err != nil {
				return nil, err
			}
			return handle(ctx, req, svc.initialized)
		},
		"shutdown": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := s.Shutdown(req)
			if err != nil {
				return nil, err
			}
			return handle(ctx, req, shutdown)
		},
		"exit": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := s.Exit()
			if err != nil {
				return nil, err
			}
			svc.stopSession()
			return nil, nil
		},

		"textDocument/didChange":              dispatchTo(svc.textDocumentDidChange),
		"textDocument/didOpen":                dispatchTo(svc.textDocumentDidOpen),
		"textDocument/didClose":               dispatchTo(svc.textDocumentDidClose),
		"textDocument/documentSymbol":         dispatchTo(svc.textDocumentSymbol),
		"textDocument/documentLink":           dispatchTo(svc.textDocumentLink),
		"textDocument/declaration":            dispatchTo(svc.textDocumentGoToDeclaration),
		"textDocument/definition":             dispatchTo(svc.textDocumentGoToDefinition),
		"textDocument/references":             dispatchTo(svc.textDocumentReferences),
		"textDocument/completion":             dispatchTo(svc.textDocumentCompletion),
		"textDocument/hover":                  dispatchTo(svc.textDocumentHover),
		"textDocument/codeLens":               dispatchTo(svc.textDocumentCodeLens),
		"textDocument/formatting":             dispatchTo(svc.textDocumentFormatting),
		"textDocument/signatureHelp":          dispatchTo(svc.textDocumentSignatureHelp),
		"textDocument/semanticTokens/full":    dispatchTo(svc.textDocumentSemanticTokensFull),
		"textDocument/foldingRange":           dispatchTo(svc.textDocumentFoldingRange),
		"textDocument/didSave":                dispatchTo(svc.textDocumentDidSave),
		"workspace/didChangeWorkspaceFolders": dispatchTo(svc.workspaceDidChangeWorkspaceFolders),
		"workspace/didChangeWatchedFiles":     dispatchTo(svc.workspaceDidChangeWatchedFiles),
		"workspace/symbol":                    dispatchTo(svc.workspaceSymbol),
		"$/cancelRequest":                     dispatchTo(cancelRequest),
	}
	return convertMap(m), nil
}

// convertMap is a helper function allowing us to omit the jrpc2.Func
// signature from the method definitions
func convertMap(m map[string]rpch.Func) rpch.Map {
	hm := make(rpch.Map, len(m))
	for method, fun := range m {
		hm[method] = rpch.New(fun)
	}
	return hm
}

// handle calls a jrpc2.Func compatible function
func handle(ctx context.Context, req *jrpc2.Request, fn interface{}) (interface{}, error) {
	if logging.PerfLogger != nil {
		start := time.Now()
		defer func() {
			logging.PerfLogger.Printf("req: %s::%s [%s]", req.Method(), req.ID(), time.Since(start))
		}()
	}
	result, err := rpch.New(fn)(ctx, req)
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
		err = fmt.Errorf("%w: %s", requestCancelled.Err(), err)
	}
	return result, err
}
