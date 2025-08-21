package api

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Renespeare/mailpulse/relay/internal/auth"
	"github.com/Renespeare/mailpulse/relay/internal/security"
	"github.com/Renespeare/mailpulse/relay/internal/smtp"
	"github.com/Renespeare/mailpulse/relay/internal/storage"
	"github.com/gorilla/mux"
)

// Server represents the HTTP API server
type Server struct {
	authManager auth.AuthManager
	storage     storage.Storage
	rateLimiter security.RateLimiter
	forwarder   *smtp.EmailForwarder
	router      *mux.Router
}

// NewServer creates a new API server
func NewServer(authManager auth.AuthManager, storage storage.Storage, rateLimiter security.RateLimiter) *Server {
	s := &Server{
		authManager: authManager,
		storage:     storage,
		rateLimiter: rateLimiter,
		forwarder:   smtp.NewEmailForwarder(authManager, storage),
		router:      mux.NewRouter(),
	}
	
	s.setupRoutes()
	return s
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	s.router.HandleFunc("/health", s.handleOptions).Methods("OPTIONS")
	
	// Quota usage
	s.router.HandleFunc("/api/quota/{projectId}", s.quotaUsageHandler).Methods("GET")
	s.router.HandleFunc("/api/quota/{projectId}", s.handleOptions).Methods("OPTIONS")
	
	// Email stats  
	s.router.HandleFunc("/api/emails/stats/{projectId}", s.emailStatsHandler).Methods("GET")
	s.router.HandleFunc("/api/emails/stats/{projectId}", s.handleOptions).Methods("OPTIONS")
	
	// Email resend
	s.router.HandleFunc("/api/emails/{emailId}/resend", s.resendEmailHandler).Methods("POST")
	s.router.HandleFunc("/api/emails/{emailId}/resend", s.handleOptions).Methods("OPTIONS")
	
	// Projects
	s.router.HandleFunc("/api/projects", s.listProjectsHandler).Methods("GET")
	s.router.HandleFunc("/api/projects", s.createProjectHandler).Methods("POST")
	s.router.HandleFunc("/api/projects", s.handleOptions).Methods("OPTIONS")
	
	s.router.HandleFunc("/api/projects/{projectId}", s.getProjectHandler).Methods("GET")
	s.router.HandleFunc("/api/projects/{projectId}", s.updateProjectHandler).Methods("PATCH")
	s.router.HandleFunc("/api/projects/{projectId}", s.deleteProjectHandler).Methods("DELETE")
	s.router.HandleFunc("/api/projects/{projectId}", s.handleOptions).Methods("OPTIONS")
	
	// Emails
	s.router.HandleFunc("/api/emails", s.listEmailsHandler).Methods("GET")
	s.router.HandleFunc("/api/emails", s.handleOptions).Methods("OPTIONS")
	
	// Audit Logs
	s.router.HandleFunc("/api/audit", s.listAuditLogsHandler).Methods("GET")
	s.router.HandleFunc("/api/audit", s.handleOptions).Methods("OPTIONS")
	s.router.HandleFunc("/api/audit/{projectId}", s.listProjectAuditLogsHandler).Methods("GET")
	s.router.HandleFunc("/api/audit/{projectId}", s.handleOptions).Methods("OPTIONS")
	
	// CORS middleware
	s.router.Use(s.corsMiddleware)
}

// Helper functions
func generateID() string {
	// Generate a random ID similar to what Prisma uses
	bytes := make([]byte, 12)
	rand.Read(bytes)
	return "cmd" + hex.EncodeToString(bytes)[:22] // Similar to Prisma cuid format
}

func generateAPIKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "mp_live_" + hex.EncodeToString(bytes)
}

func stringPtr(s string) *string {
	return &s
}

func stringPtrFromString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtrFromInt(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// StorageAdapter adapts storage.Storage to auth.ProjectStorage  
type StorageAdapter struct {
	storage storage.Storage
}

func NewStorageAdapter(s storage.Storage) *StorageAdapter {
	return &StorageAdapter{storage: s}
}

func (a *StorageAdapter) ListAllProjects() ([]*auth.StorageProject, error) {
	projects, err := a.storage.ListAllProjects()
	if err != nil {
		return nil, err
	}
	
	var authProjects []*auth.StorageProject
	for _, p := range projects {
		authProjects = append(authProjects, &auth.StorageProject{
			ID:             p.ID,
			Name:           p.Name,
			APIKey:         p.APIKey,
			PasswordHash:   p.PasswordHash,
			QuotaDaily:     p.QuotaDaily,
			QuotaPerMinute: p.QuotaPerMinute,
			Status:         p.Status,
		})
	}
	
	return authProjects, nil
}

// recordAuditLog records an audit log entry for API operations
func (s *Server) recordAuditLog(r *http.Request, action string, projectID *string, details map[string]interface{}) {
	// Generate unique audit log ID
	auditID := generateAuditID()
	
	// Extract client IP
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = strings.Split(forwarded, ",")[0]
	}
	
	// Clean up IP address for PostgreSQL INET type
	// Handle IPv6 format like [::1]:port or IPv4 like 127.0.0.1:port
	if strings.Contains(clientIP, ":") {
		if strings.HasPrefix(clientIP, "[") {
			// IPv6 format [::1]:port
			if closeBracket := strings.Index(clientIP, "]"); closeBracket != -1 {
				clientIP = clientIP[1:closeBracket]
			}
		} else {
			// IPv4 format 127.0.0.1:port
			clientIP = strings.Split(clientIP, ":")[0]
		}
	}
	
	// Extract user agent
	userAgent := r.Header.Get("User-Agent")
	var userAgentPtr *string
	if userAgent != "" {
		userAgentPtr = &userAgent
	}
	
	auditLog := &storage.AuditLog{
		ID:        auditID,
		ProjectID: projectID,
		UserID:    nil, // No user concept in current API
		Action:    action,
		IPAddress: clientIP,
		UserAgent: userAgentPtr,
		Details:   details,
		CreatedAt: time.Now(),
	}
	
	// Store audit log (non-blocking)
	go func() {
		if err := s.storage.RecordAuditLog(auditLog); err != nil {
			log.Printf("⚠️  Failed to record audit log: %v", err)
		}
	}()
}

// generateAuditID generates a unique audit log ID for API operations
func generateAuditID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "audit_" + hex.EncodeToString(bytes)
}

// Start starts the HTTP API server
func (s *Server) Start(addr string) error {
	log.Printf("🌐 Starting HTTP API server on %s", addr)
	log.Printf("📊 API Endpoints:")
	log.Printf("   GET %s/health - Server health check", addr)
	log.Printf("   GET %s/api/quota/{projectId} - Quota usage", addr)
	log.Printf("   GET %s/api/emails/stats/{projectId} - Email statistics", addr)
	log.Printf("   GET %s/api/audit - All audit logs", addr)
	log.Printf("   GET %s/api/audit/{projectId} - Project audit logs", addr)
	
	return http.ListenAndServe(addr, s.router)
}