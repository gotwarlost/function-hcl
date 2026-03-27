// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/decoder/symbols"
	ilsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/lsp"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
)

func (svc *service) workspaceSymbol(ctx context.Context, params lsp.WorkspaceSymbolParams) ([]lsp.SymbolInformation, error) {
	syms, err := symbols.WorkspaceSymbols(svc.features.modules, params.Query)
	if err != nil {
		return nil, err
	}
	return ilsp.WorkspaceSymbols(syms, svc.cc.Workspace.Symbol), nil
}
