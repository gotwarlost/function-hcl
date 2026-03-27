// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package session

import (
	"fmt"

	"github.com/creachadair/jrpc2"
)

const NotInitialized jrpc2.Code = -32002

type unexpectedState struct {
	want state
	have state
}

func (e *unexpectedState) Error() string {
	return fmt.Sprintf("session is not %s, current state: %s",
		e.want, e.have)
}

func notInitializedErr(state state) error {
	uss := &unexpectedState{
		want: stateInitializedConfirmed,
		have: state,
	}
	if state < stateInitializedConfirmed {
		return fmt.Errorf("%w: %s", NotInitialized.Err(), uss)
	}
	if state == stateDown {
		return fmt.Errorf("%w: %s", jrpc2.InvalidRequest.Err(), uss)
	}
	return uss
}

func alreadyInitializedErr(reqID string) error {
	return fmt.Errorf("%w: session was already initialized via request ID %s",
		jrpc2.SystemError.Err(), reqID)
}

func alreadyDownErr(reqID string) error {
	return fmt.Errorf("%w: session was already shut down via request %s",
		jrpc2.InvalidRequest.Err(), reqID)
}
