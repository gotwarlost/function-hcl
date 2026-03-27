// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/utils/mdplain"
)

func markupContent(content lang.MarkupContent, mdSupported bool) lsp.MarkupContent {
	value := content.Value()

	kind := lsp.PlainText
	if content.Kind() == lang.MarkdownKind {
		if mdSupported {
			kind = lsp.Markdown
		} else {
			value = mdplain.Clean(value)
		}
	}

	return lsp.MarkupContent{
		Kind:  kind,
		Value: value,
	}
}
