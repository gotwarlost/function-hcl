// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"encoding/json"
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCommandArgument is a test implementation of lang.CommandArgument
type mockCommandArgument struct {
	value string
}

func (m mockCommandArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.value)
}

// errorCommandArgument is a test implementation that returns an error
type errorCommandArgument struct{}

func (e errorCommandArgument) MarshalJSON() ([]byte, error) {
	return nil, assert.AnError
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       lang.Command
		expected    lsp.Command
		expectError bool
	}{
		{
			name: "command without arguments",
			input: lang.Command{
				Title:     "Test Command",
				ID:        "test.command",
				Arguments: []lang.CommandArgument{},
			},
			expected: lsp.Command{
				Title:     "Test Command",
				Command:   "test.command",
				Arguments: []json.RawMessage{},
			},
			expectError: false,
		},
		{
			name: "command with single argument",
			input: lang.Command{
				Title: "Format",
				ID:    "format.document",
				Arguments: []lang.CommandArgument{
					mockCommandArgument{value: "arg1"},
				},
			},
			expected: lsp.Command{
				Title:   "Format",
				Command: "format.document",
				Arguments: []json.RawMessage{
					json.RawMessage(`"arg1"`),
				},
			},
			expectError: false,
		},
		{
			name: "command with multiple arguments",
			input: lang.Command{
				Title: "Execute",
				ID:    "execute.command",
				Arguments: []lang.CommandArgument{
					mockCommandArgument{value: "first"},
					mockCommandArgument{value: "second"},
					mockCommandArgument{value: "third"},
				},
			},
			expected: lsp.Command{
				Title:   "Execute",
				Command: "execute.command",
				Arguments: []json.RawMessage{
					json.RawMessage(`"first"`),
					json.RawMessage(`"second"`),
					json.RawMessage(`"third"`),
				},
			},
			expectError: false,
		},
		{
			name: "command with argument that fails to marshal",
			input: lang.Command{
				Title: "Error Command",
				ID:    "error.command",
				Arguments: []lang.CommandArgument{
					errorCommandArgument{},
				},
			},
			expected: lsp.Command{
				Title:   "Error Command",
				Command: "error.command",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Command(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.Title, result.Title)
				assert.Equal(t, tt.expected.Command, result.Command)
				assert.Equal(t, len(tt.expected.Arguments), len(result.Arguments))
				for i, expectedArg := range tt.expected.Arguments {
					assert.JSONEq(t, string(expectedArg), string(result.Arguments[i]))
				}
			}
		})
	}
}
