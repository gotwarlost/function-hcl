// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLanguageID_String(t *testing.T) {
	tests := []struct {
		name     string
		langID   LanguageID
		expected string
	}{
		{
			name:     "HCL language ID",
			langID:   HCL,
			expected: "hcl",
		},
		{
			name:     "custom language ID",
			langID:   LanguageID("custom"),
			expected: "custom",
		},
		{
			name:     "empty language ID",
			langID:   LanguageID(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.langID.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
