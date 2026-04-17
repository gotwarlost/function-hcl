// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
)

func HandleFromDocumentURI(docUri lsp.DocumentURI) document.Handle {
	return document.HandleFromURI(string(docUri))
}
