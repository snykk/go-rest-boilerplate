package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Config configures Run and Serve.
type Config struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	// ShutdownTimeout bounds how long requests in flight may run once the context ends.
	ShutdownTimeout time.Duration
}

// Run listens on config.Addr and serves handler as Serve does. It returns the listen error when config.Addr cannot be
// used.
func Run(ctx context.Context, config Config, handler http.Handler, log *slog.Logger) error {
	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", config.Addr, err)
	}
	return Serve(ctx, listener, config, handler, log)
}

// Serve serves handler on listener until ctx ends. It then stops accepting connections and waits up to
// config.ShutdownTimeout for the requests in flight. Requests that outlive the timeout have their connections
// closed, which cancels their contexts. Serve returns nil after a graceful shutdown, the shutdown error when
// requests outlived the timeout, and the serve error when serving fails first. Serve closes listener.
func Serve(ctx context.Context, listener net.Listener, config Config, handler http.Handler, log *slog.Logger) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	log.InfoContext(ctx, "listening", slog.String("addr", listener.Addr().String()))

	select {
	case err := <-served:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.InfoContext(ctx, "shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), config.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shut down: %w", err)
	}
	return nil
}
