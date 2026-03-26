// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package session provides session lifecycle tracking for a language server.
package session

import (
	"context"
	"fmt"
	"time"

	"github.com/creachadair/jrpc2"
)

type Lifecycle struct {
	initializeReq     *jrpc2.Request
	initializeReqTime time.Time

	initializedReq     *jrpc2.Request
	initializedReqTime time.Time

	downReq     *jrpc2.Request
	downReqTime time.Time

	state    state
	exitFunc context.CancelFunc
}

func (s *Lifecycle) isPrepared() bool {
	return s.state == statePrepared
}

func (s *Lifecycle) Prepare() error {
	if s.state != stateEmpty {
		return &unexpectedState{
			want: stateInitializedConfirmed,
			have: s.state,
		}
	}
	s.state = statePrepared
	return nil
}

func (s *Lifecycle) IsInitializedUnconfirmed() bool {
	return s.state == stateInitializedUnconfirmed
}

func (s *Lifecycle) Initialize(req *jrpc2.Request) error {
	if s.state != statePrepared {
		if s.IsInitializedUnconfirmed() {
			return alreadyInitializedErr(s.initializeReq.ID())
		}
		return fmt.Errorf("session is not ready to be initialized. state: %s",
			s.state)
	}

	s.initializeReq = req
	s.initializeReqTime = time.Now()
	s.state = stateInitializedUnconfirmed

	return nil
}

func (s *Lifecycle) isInitializationConfirmed() bool {
	return s.state == stateInitializedConfirmed
}

func (s *Lifecycle) CheckInitializationIsConfirmed() error {
	if !s.isInitializationConfirmed() {
		return notInitializedErr(s.state)
	}
	return nil
}

func (s *Lifecycle) ConfirmInitialization(req *jrpc2.Request) error {
	if s.state != stateInitializedUnconfirmed {
		if s.isInitializationConfirmed() {
			return fmt.Errorf("session was already confirmed as initalized at %s via request %s",
				s.initializedReqTime, s.initializedReq.ID())
		}
		return fmt.Errorf("session is not ready to be confirmed as initialized (%s)",
			s.state)
	}
	s.initializedReq = req
	s.initializedReqTime = time.Now()
	s.state = stateInitializedConfirmed

	return nil
}

func (s *Lifecycle) Shutdown(req *jrpc2.Request) error {
	if s.isDown() {
		return alreadyDownErr(s.downReq.ID())
	}
	s.downReq = req
	s.downReqTime = time.Now()
	s.state = stateDown
	return nil
}

func (s *Lifecycle) Exit() error {
	if !s.isExitable() {
		return fmt.Errorf("cannot exit as session is %s", s.state)
	}
	s.exitFunc()
	return nil
}

func (s *Lifecycle) isExitable() bool {
	return s.isDown() || s.isPrepared()
}

func (s *Lifecycle) isDown() bool {
	return s.state == stateDown
}

func NewLifecycle(exitFunc context.CancelFunc) *Lifecycle {
	return &Lifecycle{
		state:    stateEmpty,
		exitFunc: exitFunc,
	}
}
