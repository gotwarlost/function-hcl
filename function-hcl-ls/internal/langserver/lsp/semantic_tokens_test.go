// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langhcl/lang/semtok"
	"github.com/stretchr/testify/assert"
)

func TestTokenTypesLegend(t *testing.T) {
	legend := TokenTypesLegend()

	assert.NotNil(t, legend)
	assert.Greater(t, len(legend), 0)

	// Verify expected token types are present
	expectedTypes := []semtok.TokenType{
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

	for _, expected := range expectedTypes {
		found := false
		for _, actual := range legend {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected token type %s not found in legend", expected)
	}
}

func TestTokenModifiersLegend(t *testing.T) {
	legend := TokenModifiersLegend()

	assert.NotNil(t, legend)
	assert.Greater(t, len(legend), 0)

	// Verify expected token modifiers are present
	expectedModifiers := []semtok.TokenModifier{
		semtok.TokenModifierDeclaration,
	}

	for _, expected := range expectedModifiers {
		found := false
		for _, actual := range legend {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected token modifier %s not found in legend", expected)
	}
}

func TestServerTokenTypes(t *testing.T) {
	assert.Equal(t, serverTokenTypes, TokenTypesLegend())
}

func TestServerTokenModifiers(t *testing.T) {
	assert.Equal(t, serverTokenModifiers, TokenModifiersLegend())
}
