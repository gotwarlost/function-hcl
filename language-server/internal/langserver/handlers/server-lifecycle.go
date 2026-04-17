package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/creachadair/jrpc2"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/funchcl/decoder"
	ilsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/lsp"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/hashicorp/go-uuid"
)

func (svc *service) initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
	b, _ := json.MarshalIndent(params, "", "  ")
	log.Println("Initializing:\n", string(b))
	decoder.SetBehavior(behaviorFromClientInfo(params.ClientInfo))
	serverCaps := initializeResult(svc.version)
	svc.server = jrpc2.ServerFromContext(ctx)
	svc.cc = &params.Capabilities
	err := svc.configureSessionDependencies()
	if err != nil {
		return serverCaps, err
	}
	return serverCaps, nil
}

func initializeResult(serverVersion string) lsp.InitializeResult {
	serverCaps := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: lsp.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    lsp.Incremental,
			},
			CompletionProvider: lsp.CompletionOptions{
				ResolveProvider:   false,
				TriggerCharacters: []string{".", "["},
			},
			//CodeActionProvider: lsp.CodeActionOptions{
			//	CodeActionKinds: ilsp.SupportedCodeActions.AsSlice(),
			//	ResolveProvider: false,
			//},
			DeclarationProvider: true,
			DefinitionProvider:  true,
			// define this explicitly to be an empty command list because
			// intellij does not like nil values for this attribute.
			ExecuteCommandProvider: lsp.ExecuteCommandOptions{
				Commands:                []string{},
				WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{WorkDoneProgress: false},
			},
			// CodeLensProvider:           &lsp.CodeLensOptions{},
			ReferencesProvider:         true,
			HoverProvider:              true,
			DocumentFormattingProvider: true,
			DocumentSymbolProvider:     true,
			WorkspaceSymbolProvider:    true,
			Workspace: lsp.Workspace6Gn{
				WorkspaceFolders: lsp.WorkspaceFolders5Gn{
					Supported:           true,
					ChangeNotifications: "workspace/didChangeWorkspaceFolders",
				},
			},
			SemanticTokensProvider: lsp.SemanticTokensOptions{
				Full: true,
				Legend: lsp.SemanticTokensLegend{
					TokenTypes:     ilsp.TokenTypesLegend().AsStrings(),
					TokenModifiers: ilsp.TokenModifiersLegend().AsStrings(),
				},
			},
			SignatureHelpProvider: lsp.SignatureHelpOptions{
				TriggerCharacters: []string{"(", ","},
			},
			FoldingRangeProvider: true,
		},
	}
	serverCaps.ServerInfo.Name = "function-hcl-ls"
	serverCaps.ServerInfo.Version = serverVersion
	return serverCaps
}

func (svc *service) initialized(ctx context.Context, params lsp.InitializedParams) error {
	return svc.setupWatchedFiles(ctx, svc.cc.Workspace.DidChangeWatchedFiles)
}

func (svc *service) setupWatchedFiles(ctx context.Context, caps lsp.DidChangeWatchedFilesClientCapabilities) error {
	if !caps.DynamicRegistration {
		svc.logger.Printf("Client doesn't support dynamic watched files registration, " +
			"provider and module changes may not be reflected at runtime")
		return nil
	}

	id, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	srv := jrpc2.ServerFromContext(ctx)
	_, err = srv.Callback(ctx, "client/registerCapability", lsp.RegistrationParams{
		Registrations: []lsp.Registration{
			{
				ID:     id,
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: lsp.DidChangeWatchedFilesRegistrationOptions{
					Watchers: []lsp.FileSystemWatcher{
						{
							GlobPattern: "**/*",
							Kind:        lsp.WatchCreate | lsp.WatchDelete | lsp.WatchChange,
						},
					},
				},
			},
		},
	})
	if err != nil {
		svc.logger.Printf("failed to register watched files: %s", err)
	} else {
		svc.logger.Printf("registered watched files: %s", id)
	}
	return nil
}

func shutdown(ctx context.Context, _ interface{}) error {
	return nil
}

func cancelRequest(ctx context.Context, params lsp.CancelParams) error {
	id, err := decodeRequestID(params.ID)
	if err != nil {
		return err
	}
	jrpc2.ServerFromContext(ctx).CancelRequest(id)
	return nil
}

func behaviorFromClientInfo(clientInfo lsp.Msg_XInitializeParams_clientInfo) decoder.LangServerBehavior {
	log.Printf("Client info: name=%q version=%q", clientInfo.Name, clientInfo.Version)
	var ret decoder.LangServerBehavior
	if clientInfo.Name == "function-hcl-intellij" {
		ret.MaxCompletionItems = 1000
		ret.InnerBraceRangesForFolding = true
		ret.IndentMultiLineProposals = true
	}
	return ret
}

func decodeRequestID(v interface{}) (string, error) {
	if val, ok := v.(string); ok {
		return val, nil
	}
	if val, ok := v.(float64); ok {
		return fmt.Sprintf("%d", int64(val)), nil
	}
	return "", fmt.Errorf("unable to decode request ID: %#v", v)
}
