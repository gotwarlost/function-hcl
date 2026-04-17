// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"os"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/eventbus"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/uri"
)

func (svc *service) workspaceDidChangeWatchedFiles(_ context.Context, params lsp.DidChangeWatchedFilesParams) error {
	for _, change := range params.Changes {
		svc.logger.Printf("received change event for %q: %s", change.Type, change.URI)
		rawURI := string(change.URI)

		rawPath, err := uri.PathFromURI(rawURI)
		if err != nil {
			svc.logger.Printf("error parsing %q: %s", rawURI, err)
			continue
		}
		isDir := false

		switch change.Type {
		case lsp.Changed, lsp.Created:
			fi, err := os.Stat(rawPath)
			if err != nil {
				svc.logger.Printf("error checking existence (%q changed): %s", rawPath, err)
				continue
			}
			isDir = fi.IsDir()
		}

		svc.eventBus.PublishChangeWatchEvent(eventbus.ChangeWatchEvent{
			RawPath:    rawPath,
			IsDir:      isDir,
			ChangeType: change.Type,
		})
	}
	return nil
}

func (svc *service) workspaceDidChangeWorkspaceFolders(ctx context.Context, params lsp.DidChangeWorkspaceFoldersParams) error {
	return nil
}
