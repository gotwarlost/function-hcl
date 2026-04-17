// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/stretchr/testify/assert"
)

func TestHandleFromDocumentURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      lsp.DocumentURI
		expected document.Handle
	}{
		{
			name:     "file URI unix path",
			uri:      "file:///path/to/test.hcl",
			expected: document.HandleFromURI("file:///path/to/test.hcl"),
		},
		{
			name:     "file URI with spaces",
			uri:      "file:///path/with%20spaces/test.hcl",
			expected: document.HandleFromURI("file:///path/with%20spaces/test.hcl"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HandleFromDocumentURI(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}
