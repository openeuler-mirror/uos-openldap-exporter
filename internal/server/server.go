package server

import (
	"encoding/json"
	"net/http"
	"time"

	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server for the exporter
type Server struct {
	addr        string
	metricsPath string
	collector   *collector.OpenLDAPCollector
	logger      *logrus.Logger
	ldapConfig  *config.LDAPConfig // Store LDAP config for health checks
}

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status string `json:"status"`
	LDAP   string `json:"ldap,omitempty"`
}

// New creates a new Server instance
func New(addr, metricsPath string, coll *collector.OpenLDAPCollector, logger *logrus.Logger) *Server {
	// Extract LDAP config from collector for health checks
	return &Server{
		addr:        addr,
		metricsPath: metricsPath,
		collector:   coll,
		logger:      logger,
		ldapConfig:  coll.GetLDAPConfig(),
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	// Register collector
	if err := prometheus.Register(s.collector); err != nil {
		return fmt.Errorf("failed to register collector: %w", err)
	}

	// Setup routes with middleware
	mux := http.NewServeMux()

	// Wrap promhttp handler with logging middleware
	metricsHandler := s.loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	}))

	mux.Handle(s.metricsPath, metricsHandler)

	// Health check endpoint with logging middleware
	healthzHandler := s.loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handleHealthCheck(w, r)
	}))

	mux.Handle("/healthz", healthzHandler)

	s.logger.Infof("Starting server on %s", s.addr)

	// Create server with timeouts to prevent potential Slowloris attacks
	server := &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// loggingMiddleware provides basic request logging
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		s.logger.Debugf("HTTP %s %s - %d (%v)",
			r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// handleHealthCheck handles the health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Perform actual health check by testing LDAP connectivity
	healthy, errMsg := collector.CheckLDAPHealth(s.ldapConfig, s.logger)

	response := HealthResponse{
		Status: "ok",
	}

	if !healthy {
		response.Status = "error"
		response.LDAP = errMsg
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Debugf("Failed to encode health check response: %v", err)
	}
}
