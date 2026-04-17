// Package handlers provides the core handler implementation that is responsible for managing internal features,
// and dispatching requests to these.
package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/session"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/logging"
)

func NewSession(srvCtx context.Context, serverVersion string) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	return &service{
		version:     serverVersion,
		logger:      logging.LoggerFor(logging.ModuleHandlers),
		sessCtx:     sessCtx,
		stopSession: stopSession,
	}
}

// Assigner builds out the jrpc2.Map according to the LSP protocol
// and passes related dependencies to handlers via context
func (svc *service) Assigner() (jrpc2.Assigner, error) {
	svc.logger.Println("Preparing new session ...")
	s := session.NewLifecycle(svc.stopSession)
	err := s.Prepare()
	if err != nil {
		return nil, fmt.Errorf("unable to prepare session: %w", err)
	}
	return svc.getDispatchTable(s)
}

func (svc *service) Finish(_ jrpc2.Assigner, status jrpc2.ServerStatus) {
	if status.Closed || status.Err != nil {
		svc.logger.Printf("session stopped unexpectedly (err: %v)", status.Err)
	}
	svc.stopSession()
}
