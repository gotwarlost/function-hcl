// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/utils/mdplain"
)

func ToSignatureHelp(signature *lang.FunctionSignature) *lsp.SignatureHelp {
	if signature == nil {
		return nil
	}

	parameters := make([]lsp.ParameterInformation, 0, len(signature.Parameters))
	for _, p := range signature.Parameters {
		parameters = append(parameters, lsp.ParameterInformation{
			Label: p.Name,
			// TODO: Support markdown per https://github.com/hashicorp/terraform-ls/issues/1212
			Documentation: mdplain.Clean(p.Description.Value()),
		})
	}

	return &lsp.SignatureHelp{
		Signatures: []lsp.SignatureInformation{
			{
				Label: signature.Name,
				// TODO: Support markdown per https://github.com/hashicorp/terraform-ls/issues/1212
				Documentation: mdplain.Clean(signature.Description.Value()),
				Parameters:    parameters,
			},
		},
		ActiveParameter: signature.ActiveParameter,
		ActiveSignature: 0,
	}
}
