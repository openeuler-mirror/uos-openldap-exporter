package server

import (
	"net/http"

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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	s.logger.Infof("Starting server on %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}
