// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package session

import (
	"context"

	"github.com/creachadair/jrpc2"
)

type Session interface {
	Assigner() (jrpc2.Assigner, error)
	Finish(jrpc2.Assigner, jrpc2.ServerStatus)
}

type ClientNotifier interface {
	Notify(ctx context.Context, method string, params interface{}) error
}

type Factory func(parentContext context.Context, serverVersion string) Session

type ClientCaller interface {
	Callback(ctx context.Context, method string, params interface{}) (*jrpc2.Response, error)
}

type Server interface {
	ClientNotifier
	ClientCaller
}
