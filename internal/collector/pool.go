package collector

import (
	"fmt"
	"sync"
	"time"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/sirupsen/logrus"
)

// LDAPClientPool manages a pool of LDAP client connections
type LDAPClientPool struct {
	config        *config.LDAPConfig
	logger        *logrus.Logger
	clients       chan *LDAPClient
	maxPoolSize   int
	mu            sync.Mutex
	creator       func(*config.LDAPConfig, *logrus.Logger) (*LDAPClient, error)
}

// NewLDAPClientPool creates a new LDAP client connection pool
func NewLDAPClientPool(cfg *config.LDAPConfig, logger *logrus.Logger, maxPoolSize int) *LDAPClientPool {
	if maxPoolSize <= 0 {
		maxPoolSize = 5 // Default pool size
	}

	return &LDAPClientPool{
		config:      cfg,
		logger:      logger,
		clients:     make(chan *LDAPClient, maxPoolSize),
		maxPoolSize: maxPoolSize,
		creator: func(cfg *config.LDAPConfig, logger *logrus.Logger) (*LDAPClient, error) {
			return NewLDAPClient(cfg, logger)
		},
	}
}

// Get retrieves a client from the pool or creates a new one if needed
func (p *LDAPClientPool) Get() (*LDAPClient, error) {
	select {
	case client := <-p.clients:
		// Verify the connection is still alive
		if ok, _ := client.CheckHealth(); ok {
			return client, nil
		}
		// Connection is dead, close it and create a new one
		client.Close()
		return p.createNewClient()
	default:
		// Pool is empty, create a new client
		return p.createNewClient()
	}
}

// Put returns a client back to the pool
func (p *LDAPClientPool) Put(client *LDAPClient) {
	if client == nil {
		return
	}

	select {
	case p.clients <- client:
		// Successfully returned to pool
	default:
		// Pool is full, close the connection
		client.Close()
	}
}

// createNewClient creates a new LDAP client
func (p *LDAPClientPool) createNewClient() (*LDAPClient, error) {
	client, err := p.creator(p.config, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create LDAP client: %w", err)
	}
	return client, nil
}

// Close closes all connections in the pool
func (p *LDAPClientPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.clients)
	for client := range p.clients {
		client.Close()
	}
}

// Stats returns pool statistics
func (p *LDAPClientPool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"max_pool_size": p.maxPoolSize,
		"active_clients": len(p.clients),
	}
}
