// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lang

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

type CompletionHook struct {
	Name string
}

type ResolveHook struct {
	Name string `json:"resolve_hook,omitempty"`
	Path string `json:"path,omitempty"`
}

type CompletionHooks []CompletionHook

func (chs CompletionHooks) Copy() CompletionHooks {
	if chs == nil {
		return nil
	}

	hooksCopy := make(CompletionHooks, len(chs))
	copy(hooksCopy, chs)
	return hooksCopy
}

// HookCandidate represents a completion candidate created and returned from a
// completion hook.
type HookCandidate struct {
	// Label represents a human-readable name of the candidate
	// if one exists (otherwise Value is used)
	Label string
	// Detail represents a human-readable string with additional
	// information about this candidate, like symbol information.
	Detail string
	Kind   CandidateKind
	// Description represents human-readable description
	// of the candidate
	Description MarkupContent
	// IsDeprecated indicates whether the candidate is deprecated
	IsDeprecated bool
	// RawInsertText represents the final text which is used to build the
	// TextEdit for completion. It should contain quotes when completing
	// strings.
	RawInsertText string
	// ResolveHook represents a resolve hook to call
	// and any arguments to pass to it
	ResolveHook *ResolveHook
	// SortText is an optional string that will be used when comparing this
	// candidate with other candidates
	SortText string
}

// ExpressionCandidate is a simplified version of HookCandidate and the preferred
// way to create completion candidates from completion hooks for attributes
// values (expressions). One can use ExpressionCompletionCandidate to convert
// those into candidates.
type ExpressionCandidate struct {
	// Value represents the value to be inserted
	Value cty.Value

	// Detail represents a human-readable string with additional
	// information about this candidate, like symbol information.
	Detail string

	// Description represents human-readable description
	// of the candidate
	Description MarkupContent

	// IsDeprecated indicates whether the candidate is deprecated
	IsDeprecated bool
}

// ExpressionCompletionCandidate converts a simplified ExpressionCandidate
// into a HookCandidate while taking care of populating fields and quoting strings
func ExpressionCompletionCandidate(c ExpressionCandidate) HookCandidate {
	// We're adding quotes to the string here, as we're always
	// replacing the whole edit range for attribute expressions
	text := fmt.Sprintf("%q", c.Value.AsString())

	return HookCandidate{
		Label:         text,
		Detail:        c.Detail,
		Kind:          candidateKindForType(c.Value.Type()),
		Description:   c.Description,
		IsDeprecated:  c.IsDeprecated,
		RawInsertText: text,
	}
}

func candidateKindForType(t cty.Type) CandidateKind {
	if t == cty.Bool {
		return BoolCandidateKind
	}
	if t == cty.String {
		return StringCandidateKind
	}
	if t == cty.Number {
		return NumberCandidateKind
	}
	if t.IsListType() {
		return ListCandidateKind
	}
	if t.IsSetType() {
		return SetCandidateKind
	}
	if t.IsTupleType() {
		return TupleCandidateKind
	}
	if t.IsMapType() {
		return MapCandidateKind
	}
	if t.IsObjectType() {
		return ObjectCandidateKind
	}

	return NilCandidateKind
}
