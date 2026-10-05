package mcpserver

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Status is the live MCP server state shown in Preferences.
type Status struct {
	Running bool   `json:"running"`
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
	Token   string `json:"token"`
	Error   string `json:"error,omitempty"`
}

// Server owns the localhost MCP HTTP listener.
type Server struct {
	mu     sync.Mutex
	host   Host
	cfg    Config
	dir    string
	version string
	http   *http.Server
	ln     net.Listener
	err    string
}

// New builds a stopped MCP server bound to the given host implementation.
func New(host Host, dataDir, version string) *Server {
	return &Server{
		host:    host,
		dir:     dataDir,
		version: version,
		cfg:     LoadConfig(dataDir),
	}
}

// Config returns a copy of the current config.
func (s *Server) Config() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// Status reports whether the listener is up.
func (s *Server) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{
		Running: s.http != nil && s.ln != nil,
		Enabled: s.cfg.Enabled,
		Port:    s.cfg.Port,
		Token:   s.cfg.Token,
		Error:   s.err,
	}
	if st.Port > 0 {
		st.URL = fmt.Sprintf("http://127.0.0.1:%d/mcp", st.Port)
	}
	return st
}

// Apply saves config and starts or stops the listener to match Enabled.
func (s *Server) Apply(cfg Config) error {
	cfg = normalize(cfg)
	if cfg.Token == "" {
		cfg.Token = newToken()
	}
	if err := SaveConfig(s.dir, cfg); err != nil {
		return err
	}

	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()

	if !cfg.Enabled {
		return s.Stop()
	}
	// Always bounce the listener so a rotated token or port change takes effect.
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// Start launches the streamable HTTP MCP server on 127.0.0.1.
// Safe to call when already running with the same port (no-op).
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.http != nil && s.ln != nil {
		// Restart if the port changed while running.
		addr := s.ln.Addr().String()
		want := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
		if strings.HasSuffix(addr, fmt.Sprintf(":%d", s.cfg.Port)) || addr == want {
			s.err = ""
			return nil
		}
		_ = s.stopLocked()
	}

	if s.cfg.Port <= 0 {
		s.cfg.Port = DefaultPort
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "jumpstart",
		Version: s.version,
	}, nil)
	registerTools(mcpServer, s.host)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.cfg.Port))
	if err != nil {
		s.err = err.Error()
		return err
	}

	srv := &http.Server{
		Handler:           withBearerAuth(s.cfg.Token, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.http = srv
	s.ln = ln
	s.err = ""

	go func() {
		err := srv.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			s.mu.Lock()
			s.err = err.Error()
			s.http = nil
			s.ln = nil
			s.mu.Unlock()
		}
	}()
	return nil
}

// Stop shuts down the listener if it is running.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked()
}

func (s *Server) stopLocked() error {
	if s.http == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.http.Shutdown(ctx)
	s.http = nil
	s.ln = nil
	if err != nil {
		s.err = err.Error()
	}
	return err
}

// SyncFromDisk reloads config and aligns the listener (used on app startup).
func (s *Server) SyncFromDisk() error {
	cfg := LoadConfig(s.dir)
	return s.Apply(cfg)
}

func withBearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.Error(w, "mcp token not configured", http.StatusUnauthorized)
			return
		}
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		ok := false
		if strings.HasPrefix(auth, prefix) {
			ok = secureTokenEq(auth[len(prefix):], token)
		}
		if !ok {
			ok = secureTokenEq(r.URL.Query().Get("token"), token)
		}
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func secureTokenEq(got, want string) bool {
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
