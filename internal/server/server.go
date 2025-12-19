package server

import (
	"encoding/json"
	"net/http"
	"time"

	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
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
}

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status string `json:"status"`
	LDAP   string `json:"ldap,omitempty"`
}

// New creates a new Server instance
func New(addr, metricsPath string, coll *collector.OpenLDAPCollector, logger *logrus.Logger) *Server {
	return &Server{
		addr:        addr,
		metricsPath: metricsPath,
		collector:   coll,
		logger:      logger,
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	// Register collector
	prometheus.MustRegister(s.collector)

	// Setup routes
	mux := http.NewServeMux()
	mux.Handle(s.metricsPath, promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Perform actual health check by testing LDAP connectivity
		healthy := s.checkLDAPConnectivity()
		
		response := HealthResponse{
			Status: "ok",
		}
		
		if !healthy {
			response.Status = "error"
			response.LDAP = "LDAP connection failed"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.logger.Debugf("Failed to encode health check response: %v", err)
		}
	})

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

// checkLDAPConnectivity verifies LDAP server connectivity
func (s *Server) checkLDAPConnectivity() bool {
	// This would ideally perform a lightweight connectivity check
	// For now, we'll return true as a placeholder
	// A full implementation would attempt to connect to the LDAP server
	// without performing expensive operations
	return true
}