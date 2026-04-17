// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langhcl/lang"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestToCompletionList(t *testing.T) {
	tests := []struct {
		name       string
		candidates lang.Candidates
		caps       lsp.TextDocumentClientCapabilities
		expected   lsp.CompletionList
	}{
		{
			name: "empty candidates",
			candidates: lang.Candidates{
				List:       []lang.Candidate{},
				IsComplete: true,
			},
			caps: lsp.TextDocumentClientCapabilities{},
			expected: lsp.CompletionList{
				Items:        []lsp.CompletionItem{},
				IsIncomplete: false,
			},
		},
		{
			name: "incomplete candidates list",
			candidates: lang.Candidates{
				List:       []lang.Candidate{},
				IsComplete: false,
			},
			caps: lsp.TextDocumentClientCapabilities{},
			expected: lsp.CompletionList{
				Items:        []lsp.CompletionItem{},
				IsIncomplete: true,
			},
		},
		{
			name: "single attribute candidate",
			candidates: lang.Candidates{
				List: []lang.Candidate{
					{
						Label:       "test_attribute",
						Description: lang.PlainText("Test attribute description"),
						Detail:      "string",
						Kind:        lang.AttributeCandidateKind,
						TextEdit: lang.TextEdit{
							Range: hcl.Range{
								Start: hcl.Pos{Line: 1, Column: 1},
								End:   hcl.Pos{Line: 1, Column: 1},
							},
							NewText: "test_attribute",
							Snippet: "test_attribute = ${1:value}",
						},
					},
				},
				IsComplete: true,
			},
			caps: lsp.TextDocumentClientCapabilities{
				Completion: lsp.CompletionClientCapabilities{
					CompletionItem: lsp.PCompletionItemPCompletion{
						SnippetSupport: true,
					},
				},
			},
			expected: lsp.CompletionList{
				Items: []lsp.CompletionItem{
					{
						Label:            "test_attribute",
						Kind:             lsp.PropertyCompletion,
						Detail:           "string",
						Documentation:    "Test attribute description",
						InsertTextFormat: lsp.SnippetTextFormat,
					},
				},
				IsIncomplete: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCompletionList(tt.candidates, tt.caps)
			assert.Equal(t, tt.expected.IsIncomplete, result.IsIncomplete)
			assert.Len(t, result.Items, len(tt.expected.Items))
			for i := range tt.expected.Items {
				assert.Equal(t, tt.expected.Items[i].Label, result.Items[i].Label)
				assert.Equal(t, tt.expected.Items[i].Kind, result.Items[i].Kind)
				assert.Equal(t, tt.expected.Items[i].Detail, result.Items[i].Detail)
				assert.Equal(t, tt.expected.Items[i].Documentation, result.Items[i].Documentation)
			}
		})
	}
}

func TestToCompletionItem_AllCandidateKinds(t *testing.T) {
	tests := []struct {
		name          string
		candidateKind lang.CandidateKind
		expectedKind  lsp.CompletionItemKind
	}{
		{name: "attribute", candidateKind: lang.AttributeCandidateKind, expectedKind: lsp.PropertyCompletion},
		{name: "block", candidateKind: lang.BlockCandidateKind, expectedKind: lsp.ClassCompletion},
		{name: "label", candidateKind: lang.LabelCandidateKind, expectedKind: lsp.FieldCompletion},
		{name: "bool", candidateKind: lang.BoolCandidateKind, expectedKind: lsp.EnumMemberCompletion},
		{name: "string", candidateKind: lang.StringCandidateKind, expectedKind: lsp.TextCompletion},
		{name: "number", candidateKind: lang.NumberCandidateKind, expectedKind: lsp.ValueCompletion},
		{name: "keyword", candidateKind: lang.KeywordCandidateKind, expectedKind: lsp.KeywordCompletion},
		{name: "list", candidateKind: lang.ListCandidateKind, expectedKind: lsp.EnumCompletion},
		{name: "set", candidateKind: lang.SetCandidateKind, expectedKind: lsp.EnumCompletion},
		{name: "tuple", candidateKind: lang.TupleCandidateKind, expectedKind: lsp.EnumCompletion},
		{name: "map", candidateKind: lang.MapCandidateKind, expectedKind: lsp.StructCompletion},
		{name: "object", candidateKind: lang.ObjectCandidateKind, expectedKind: lsp.StructCompletion},
		{name: "reference", candidateKind: lang.ReferenceCandidateKind, expectedKind: lsp.VariableCompletion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := lang.Candidate{
				Label: "test",
				Kind:  tt.candidateKind,
			}
			caps := lsp.CompletionClientCapabilities{}

			item := toCompletionItem(candidate, caps)
			assert.Equal(t, tt.expectedKind, item.Kind)
		})
	}
}

func TestToCompletionItem_TriggerSuggest(t *testing.T) {
	tests := []struct {
		name           string
		triggerSuggest bool
		snippetSupport bool
		expectCommand  bool
	}{
		{
			name:           "trigger suggest enabled with snippet support",
			triggerSuggest: true,
			snippetSupport: true,
			expectCommand:  true,
		},
		{
			name:           "trigger suggest enabled without snippet support",
			triggerSuggest: true,
			snippetSupport: false,
			expectCommand:  false,
		},
		{
			name:           "trigger suggest disabled with snippet support",
			triggerSuggest: false,
			snippetSupport: true,
			expectCommand:  false,
		},
		{
			name:           "trigger suggest disabled without snippet support",
			triggerSuggest: false,
			snippetSupport: false,
			expectCommand:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := lang.Candidate{
				Label:          "test",
				TriggerSuggest: tt.triggerSuggest,
			}
			caps := lsp.CompletionClientCapabilities{
				CompletionItem: lsp.PCompletionItemPCompletion{
					SnippetSupport: tt.snippetSupport,
				},
			}

			item := toCompletionItem(candidate, caps)
			if tt.expectCommand {
				assert.NotNil(t, item.Command)
				assert.Equal(t, "editor.action.triggerSuggest", item.Command.Command)
				assert.Equal(t, "Suggest", item.Command.Title)
			} else {
				assert.Nil(t, item.Command)
			}
		})
	}
}

func TestToCompletionItem_Deprecated(t *testing.T) {
	tests := []struct {
		name              string
		isDeprecated      bool
		deprecatedSupport bool
		tagSupport        []lsp.CompletionItemTag
		expectDeprecated  bool
		expectTags        bool
	}{
		{
			name:              "deprecated with support",
			isDeprecated:      true,
			deprecatedSupport: true,
			tagSupport:        []lsp.CompletionItemTag{lsp.ComplDeprecated},
			expectDeprecated:  true,
			expectTags:        true,
		},
		{
			name:              "deprecated without deprecated support",
			isDeprecated:      true,
			deprecatedSupport: false,
			tagSupport:        []lsp.CompletionItemTag{lsp.ComplDeprecated},
			expectDeprecated:  false,
			expectTags:        true,
		},
		{
			name:              "deprecated without tag support",
			isDeprecated:      true,
			deprecatedSupport: true,
			tagSupport:        []lsp.CompletionItemTag{},
			expectDeprecated:  true,
			expectTags:        false,
		},
		{
			name:              "not deprecated",
			isDeprecated:      false,
			deprecatedSupport: true,
			tagSupport:        []lsp.CompletionItemTag{lsp.ComplDeprecated},
			expectDeprecated:  false,
			expectTags:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := lang.Candidate{
				Label:        "test",
				IsDeprecated: tt.isDeprecated,
			}
			caps := lsp.CompletionClientCapabilities{
				CompletionItem: lsp.PCompletionItemPCompletion{
					DeprecatedSupport: tt.deprecatedSupport,
					TagSupport: lsp.FTagSupportPCompletionItem{
						ValueSet: tt.tagSupport,
					},
				},
			}

			item := toCompletionItem(candidate, caps)
			assert.Equal(t, tt.expectDeprecated, item.Deprecated)
			if tt.expectTags {
				assert.Equal(t, []lsp.CompletionItemTag{lsp.ComplDeprecated}, item.Tags)
			} else {
				assert.Empty(t, item.Tags)
			}
		})
	}
}

func TestToCompletionItem_ResolveHook(t *testing.T) {
	hook := &lang.ResolveHook{}
	candidate := lang.Candidate{
		Label:       "test",
		ResolveHook: hook,
	}
	caps := lsp.CompletionClientCapabilities{}

	item := toCompletionItem(candidate, caps)
	assert.Equal(t, hook, item.Data)
}

func TestToCompletionItem_ResolveHookNil(t *testing.T) {
	candidate := lang.Candidate{
		Label:       "test",
		ResolveHook: nil,
	}
	caps := lsp.CompletionClientCapabilities{}

	item := toCompletionItem(candidate, caps)
	assert.Nil(t, item.Data)
}

func TestToCompletionItem_MarkdownCleaning(t *testing.T) {
	candidate := lang.Candidate{
		Label:       "test",
		Description: lang.Markdown("# Heading\n\nSome **bold** text with `code`"),
	}
	caps := lsp.CompletionClientCapabilities{}

	item := toCompletionItem(candidate, caps)
	// mdplain.Clean should remove markdown formatting
	assert.NotContains(t, item.Documentation, "**")
	assert.NotContains(t, item.Documentation, "`")
	assert.NotContains(t, item.Documentation, "#")
}

func TestToCompletionItem_AdditionalTextEdits(t *testing.T) {
	candidate := lang.Candidate{
		Label: "test",
		AdditionalTextEdits: []lang.TextEdit{
			{
				Range: hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1},
					End:   hcl.Pos{Line: 1, Column: 5},
				},
				NewText: "additional",
				Snippet: "additional ${1:text}",
			},
		},
	}
	caps := lsp.CompletionClientCapabilities{
		CompletionItem: lsp.PCompletionItemPCompletion{
			SnippetSupport: true,
		},
	}

	item := toCompletionItem(candidate, caps)
	assert.Len(t, item.AdditionalTextEdits, 1)
	assert.Equal(t, "additional ${1:text}", item.AdditionalTextEdits[0].NewText)
}

func TestToCompletionItem_SortText(t *testing.T) {
	candidate := lang.Candidate{
		Label:    "test",
		SortText: "00_test",
	}
	caps := lsp.CompletionClientCapabilities{}

	item := toCompletionItem(candidate, caps)
	assert.Equal(t, "00_test", item.SortText)
}

func TestTagSliceContains(t *testing.T) {
	tests := []struct {
		name      string
		supported []lsp.CompletionItemTag
		tag       lsp.CompletionItemTag
		expected  bool
	}{
		{
			name:      "empty slice",
			supported: []lsp.CompletionItemTag{},
			tag:       lsp.ComplDeprecated,
			expected:  false,
		},
		{
			name:      "tag present",
			supported: []lsp.CompletionItemTag{lsp.ComplDeprecated},
			tag:       lsp.ComplDeprecated,
			expected:  true,
		},
		{
			name:      "tag not present",
			supported: []lsp.CompletionItemTag{lsp.ComplDeprecated},
			tag:       lsp.CompletionItemTag(99),
			expected:  false,
		},
		{
			name:      "multiple tags, tag present",
			supported: []lsp.CompletionItemTag{lsp.ComplDeprecated, lsp.CompletionItemTag(2)},
			tag:       lsp.CompletionItemTag(2),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tagSliceContains(tt.supported, tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}
