// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package langserver implements the language server endpoints.
package langserver

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/server"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/handlers"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/session"
	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/utils/logging"
)

type Options struct {
	ServerVersion string
	Concurrency   int
	Factory       session.Factory
}

type LangServer struct {
	version    string
	srvCtx     context.Context
	logger     *log.Logger
	srvOptions *jrpc2.ServerOptions
	newSession session.Factory
}

func (o *Options) setDefaults() {
	if o.ServerVersion == "" {
		o.ServerVersion = "1.0"
	}
	if o.Factory == nil {
		o.Factory = handlers.NewSession
	}
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency()
	}
}

func New(srvCtx context.Context, opts Options) *LangServer {
	opts.setDefaults()
	logger := logging.LoggerFor(logging.ModuleLangServer)
	rpcOpts := &jrpc2.ServerOptions{
		AllowPush:   true,
		Concurrency: opts.Concurrency,
		Logger:      jrpc2.StdLogger(logger),
		RPCLog:      &rpcLogger{logger},
	}
	return &LangServer{
		version:    opts.ServerVersion,
		srvCtx:     srvCtx,
		logger:     logger,
		srvOptions: rpcOpts,
		newSession: opts.Factory,
	}
}

func DefaultConcurrency() int {
	cpu := runtime.NumCPU()
	// Cap concurrency on powerful machines
	// to leave some capacity for module ops
	// and other application
	if cpu >= 4 {
		return cpu / 2
	}
	return cpu
}

func (ls *LangServer) newService() server.Service {
	return ls.newSession(ls.srvCtx, ls.version)
}

func (ls *LangServer) startServer(reader io.Reader, writer io.WriteCloser) (*singleServer, error) {
	srv, err := getServer(ls.newService(), ls.srvOptions)
	if err != nil {
		return nil, err
	}
	srv.Start(channel.LSP(reader, writer))

	return srv, nil
}

func (ls *LangServer) StartAndWait(reader io.Reader, writer io.WriteCloser) error {
	srv, err := ls.startServer(reader, writer)
	if err != nil {
		return err
	}
	ls.logger.Printf("Starting server (pid %d; concurrency: %d) ...",
		os.Getpid(), ls.srvOptions.Concurrency)

	// Wrap waiter with a context so that we can cancel it here
	// after the service is cancelled (and srv.Wait returns)
	ctx, cancelFunc := context.WithCancel(ls.srvCtx)
	go func() {
		srv.Wait()
		cancelFunc()
	}()

	<-ctx.Done()
	ls.logger.Printf("Stopping server (pid %d) ...", os.Getpid())
	srv.Stop()
	ls.logger.Printf("Server (pid %d) stopped.", os.Getpid())
	return nil
}

func (ls *LangServer) StartTCP(address string) error {
	ls.logger.Printf("Starting TCP server (pid %d; concurrency: %d) at %q ...",
		os.Getpid(), ls.srvOptions.Concurrency, address)
	lst, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("TCP Server failed to start: %s", err)
	}
	ls.logger.Printf("TCP server running at %q", lst.Addr())

	accepter := server.NetAccepter(lst, channel.LSP)

	go func() {
		ls.logger.Println("Starting loop server ...")
		err = server.Loop(context.TODO(), accepter, ls.newService, &server.LoopOptions{
			ServerOptions: ls.srvOptions,
		})
		if err != nil {
			ls.logger.Printf("Loop server failed to start: %s", err)
		}
	}()

	<-ls.srvCtx.Done()
	ls.logger.Printf("Stopping TCP server (pid %d) ...", os.Getpid())
	err = lst.Close()
	if err != nil {
		ls.logger.Printf("TCP server (pid %d) failed to stop: %s", os.Getpid(), err)
		return err
	}
	ls.logger.Printf("TCP server (pid %d) stopped.", os.Getpid())
	return nil
}

// singleServer is a wrapper around jrpc2.NewServer providing support
// for server.Service (Assigner/Finish interface)
type singleServer struct {
	srv        *jrpc2.Server
	finishFunc func(jrpc2.ServerStatus)
}

func getServer(svc server.Service, opts *jrpc2.ServerOptions) (*singleServer, error) {
	assigner, err := svc.Assigner()
	if err != nil {
		return nil, err
	}
	return &singleServer{
		srv: jrpc2.NewServer(assigner, opts),
		finishFunc: func(status jrpc2.ServerStatus) {
			svc.Finish(assigner, status)
		},
	}, nil
}

func (ss *singleServer) Start(ch channel.Channel) {
	ss.srv = ss.srv.Start(ch)
}

func (ss *singleServer) StartAndWait(ch channel.Channel) {
	ss.Start(ch)
	ss.Wait()
}

func (ss *singleServer) Wait() {
	status := ss.srv.WaitStatus()
	ss.finishFunc(status)
}

func (ss *singleServer) Stop() {
	ss.srv.Stop()
}
