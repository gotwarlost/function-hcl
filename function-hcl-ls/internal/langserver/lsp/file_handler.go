// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
)

func HandleFromDocumentURI(docUri lsp.DocumentURI) document.Handle {
	return document.HandleFromURI(string(docUri))
}
