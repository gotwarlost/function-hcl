// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package semtok

import (
	"github.com/hashicorp/hcl/v2"
)

type SemanticToken struct {
	Type      TokenType
	Modifiers TokenModifiers
	Range     hcl.Range
}
