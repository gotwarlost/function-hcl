// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package protocol

import "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang"

type CompletionItemWithResolveHook struct {
	CompletionItem

	ResolveHook *lang.ResolveHook `json:"data,omitempty"`
}
