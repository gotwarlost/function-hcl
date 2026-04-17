// Package cmd provides sub-command implementations.
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/langserver"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/handlers"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/logging"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type serveParams struct {
	port           int
	logFilePath    string
	logModules     string
	cpuProfile     string
	memProfile     string
	reqConcurrency int
	stdio          bool
}

// Version tracks the version of the command.
var Version string

// AddServeCommand adds the serve sub-command.
func AddServeCommand(root *cobra.Command) {
	var p serveParams
	c := &cobra.Command{
		Use:   "serve",
		Short: "run the language server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return serve(p)
		},
	}
	root.AddCommand(c)

	f := c.Flags()
	f.IntVar(&p.port, "port", 0, "port number to listen on (turns server into TCP mode)")
	f.StringVar(&p.logFilePath, "log-file", "", "path to a file to log into with support "+
		"for variables (e.g. timestamp, pid, ppid) via Go template syntax {{varName}}")
	f.StringVar(&p.logModules, "log-modules", "", "comma-separated list of modules to enable logging for "+
		"(langserver,handlers,eventbus,modules,crds,filesystem,docstore,perf) or 'all'")
	f.StringVar(&p.cpuProfile, "cpu-profile", "", "file into which to write CPU profile")
	f.StringVar(&p.memProfile, "mem-profile", "", "file into which to write memory profile")
	f.IntVar(&p.reqConcurrency, "concurrency", 0, fmt.Sprintf("number of RPC requests to process concurrently,"+
		" defaults to %d, concurrency lower than 2 is not recommended", langserver.DefaultConcurrency()))
	f.BoolVar(&p.stdio, "stdio", true, "use stdio")
}

func serve(c serveParams) error {
	if c.cpuProfile != "" {
		stop, err := writeCpuProfileInto(c.cpuProfile)
		if err != nil {
			return errors.Wrap(err, "write CPU profile")
		}
		if stop != nil {
			defer func() { _ = stop() }()
		}
	}

	if c.memProfile != "" {
		defer func() {
			_ = writeMemoryProfileInto(c.memProfile)
		}()
	}

	// Parse enabled modules
	enabledModules, err := logging.ParseModules(c.logModules)
	if err != nil {
		return errors.Wrap(err, "parse log modules")
	}

	// Set up logging output and initialize registry
	var logOutput *logging.FileLogger
	if c.logFilePath != "" {
		var err error
		logOutput, err = logging.NewFileLogger(c.logFilePath)
		if err != nil {
			return errors.Wrap(err, "open log file")
		}
		defer func() { _ = logOutput.Close() }()
		logging.Init(logOutput.Writer(), enabledModules)
	} else {
		logging.Init(nil, enabledModules)
	}

	// Get a logger for startup messages
	logger := logging.LoggerFor(logging.ModuleLangServer)

	ctx, cancelFunc := withSignalCancel(context.Background(), logger, os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	logger.Printf("Starting function-hcl-ls %s", Version)

	srv := langserver.New(ctx, langserver.Options{
		ServerVersion: Version,
		Concurrency:   c.reqConcurrency,
		Factory:       handlers.NewSession,
	})

	if c.port != 0 {
		err := srv.StartTCP(fmt.Sprintf("localhost:%d", c.port))
		if err != nil {
			return errors.Wrap(err, "start tcp server")
		}
		return nil
	}

	return srv.StartAndWait(os.Stdin, os.Stdout)
}

type stopFunc func() error

func withSignalCancel(ctx context.Context, l *log.Logger, sigs ...os.Signal) (
	context.Context, context.CancelFunc,
) {
	ctx, cancelFunc := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, sigs...)

	go func() {
		select {
		case sig := <-sigChan:
			l.Printf("Cancellation signal (%s) received", sig)
			cancelFunc()
		case <-ctx.Done():
		}
	}()

	f := func() {
		signal.Stop(sigChan)
		cancelFunc()
	}

	return ctx, f
}

func writeCpuProfileInto(rawPath string) (stopFunc, error) {
	path, err := logging.ParseRawPath("cpuprofile-path", rawPath)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("could not create CPU profile: %s", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		return f.Close, fmt.Errorf("could not start CPU profile: %s", err)
	}
	return func() error {
		pprof.StopCPUProfile()
		return f.Close()
	}, nil
}

func writeMemoryProfileInto(rawPath string) error {
	path, err := logging.ParseRawPath("memprofile-path", rawPath)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create memory profile: %s", err)
	}
	defer func() { _ = f.Close() }()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("could not write memory profile: %s", err)
	}
	return nil
}
