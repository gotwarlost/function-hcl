// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package protocol

import "github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"

type CompletionItemWithResolveHook struct {
	CompletionItem

	ResolveHook *lang.ResolveHook `json:"data,omitempty"`
}
