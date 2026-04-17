// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package session

// state represents state of the language server
// workspace ("session") with respect to the LSP
type state int

const (
	stateEmpty                  state = -1 // before session starts
	statePrepared               state = 0  // after session starts, before any request
	stateInitializedUnconfirmed state = 1  // after "initialize", before "initialized"
	stateInitializedConfirmed   state = 2  // after "initialized"
	stateDown                   state = 3  // after "shutdown"
)

func (ss state) String() string {
	switch ss {
	case stateEmpty:
		return "<empty>"
	case statePrepared:
		return "prepared"
	case stateInitializedUnconfirmed:
		return "initialized (unconfirmed)"
	case stateInitializedConfirmed:
		return "initialized (confirmed)"
	case stateDown:
		return "down"
	}
	return "<unknown>"
}
