// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
)

func Command(cmd lang.Command) (lsp.Command, error) {
	lspCmd := lsp.Command{
		Title:   cmd.Title,
		Command: cmd.ID,
	}

	for _, arg := range cmd.Arguments {
		jsonArg, err := arg.MarshalJSON()
		if err != nil {
			return lspCmd, err
		}
		lspCmd.Arguments = append(lspCmd.Arguments, jsonArg)
	}

	return lspCmd, nil
}
