// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang/semtok"
)

// Registering types which are actually in use
var (
	serverTokenTypes = semtok.TokenTypes{
		semtok.TokenTypeNamespace,
		semtok.TokenTypeClass,
		semtok.TokenTypeEnumMember,
		semtok.TokenTypeFunction,
		semtok.TokenTypeKeyword,
		semtok.TokenTypeNumber,
		semtok.TokenTypeProperty,
		semtok.TokenTypeString,
		semtok.TokenTypeVariable,
		semtok.TokenTypeOperator,
	}
	serverTokenModifiers = semtok.TokenModifiers{
		semtok.TokenModifierDeclaration,
		semtok.TokenModifierDefinition,
	}
)

func TokenTypesLegend() semtok.TokenTypes {
	return serverTokenTypes
}

func TokenModifiersLegend() semtok.TokenModifiers {
	return serverTokenModifiers
}
