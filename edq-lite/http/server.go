package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	edqlite "github.com/dustland/agentok/edq-lite"
)

// ServerConfig configures the HTTP server.
type ServerConfig struct {
	Addr       string // default ":8900"
	BrokerOpts []edqlite.BrokerOption
}

// Server wraps the broker and HTTP handler.
type Server struct {
	broker  *edqlite.Broker
	handler *Handler
	httpSrv *http.Server
	cfg     ServerConfig
}

// NewServer creates a new EDQ Lite HTTP server.
func NewServer(cfg ServerConfig) *Server {
	if cfg.Addr == "" {
		cfg.Addr = ":8900"
	}
	broker := edqlite.NewBroker(cfg.BrokerOpts...)
	handler := NewHandler(broker)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	wrapped := chain(mux, recoveryMiddleware, loggingMiddleware, requestIDMiddleware)

	return &Server{
		broker:  broker,
		handler: handler,
		httpSrv: &http.Server{
			Addr:    cfg.Addr,
			Handler: wrapped,
		},
		cfg: cfg,
	}
}

// ListenAndServe starts the broker and HTTP server.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if err := s.broker.Start(ctx); err != nil {
		return fmt.Errorf("server: broker start failed: %w", err)
	}

	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return fmt.Errorf("server: listen failed: %w", err)
	}

	go func() {
		<-ctx.Done()
		s.httpSrv.Close()
	}()

	if err := s.httpSrv.Serve(ln); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.handler.Close()
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return err
	}
	return s.broker.Shutdown(ctx)
}

// Broker returns the underlying broker for direct access.
func (s *Server) Broker() *edqlite.Broker {
	return s.broker
}
