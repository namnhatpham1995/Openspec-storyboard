// Command storyboard runs the Storyboard server: a local, portable board
// for viewing and managing OpenSpec projects.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"storyboard/internal/registry"
	"storyboard/internal/server"
)

// version is stamped at release build time via -ldflags (see design D9 / task 9.3).
// It stays "dev" for local `go run` / `go build` invocations.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run contains main's logic without calling os.Exit, so it can be unit
// tested directly. It returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runContext(ctx, args, stdout, stderr)
}

func runContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("storyboard", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print the version and exit")
	projectRoot := fs.String("project", "", "OpenSpec project folder to register on startup")
	configPath := fs.String("config", "", "project registry file (defaults to the OS user config directory)")
	address := fs.String("addr", "127.0.0.1:8080", "loopback address to listen on")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "storyboard", version)
		return 0
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	if *configPath == "" {
		var err error
		*configPath, err = registry.DefaultPath()
		if err != nil {
			logger.Error("locating project registry", "error", err)
			return 1
		}
	}
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		logger.Error("starting server", "address", *address, "error", err)
		return 1
	}

	app, err := server.NewPersistent(*configPath, *projectRoot, logger)
	if err != nil {
		listener.Close()
		logger.Error("loading project registry", "error", err)
		return 1
	}
	httpServer := &http.Server{
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- httpServer.Serve(listener)
	}()
	go func() {
		if err := app.WatchProjects(ctx); err != nil && ctx.Err() == nil {
			logger.Error("live updates stopped", "error", err)
		}
	}()
	logger.Info("storyboard server started", "version", version, "address", listener.Addr(), "project", *projectRoot, "config", *configPath)

	select {
	case err := <-serveErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			return 1
		}
	case <-ctx.Done():
		logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			return 1
		}
		if err := <-serveErrors; !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			return 1
		}
	}

	return 0
}
