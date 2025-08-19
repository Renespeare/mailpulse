package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Renespeare/mailpulse/relay/internal/auth"
	"github.com/Renespeare/mailpulse/relay/internal/crypto"
	"github.com/Renespeare/mailpulse/relay/internal/security"
	"github.com/Renespeare/mailpulse/relay/internal/smtp"
	"github.com/Renespeare/mailpulse/relay/internal/storage"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
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
	
	// CORS middleware
	s.router.Use(s.corsMiddleware)
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			log.Printf("CORS preflight request from %s for %s", r.Header.Get("Origin"), r.URL.Path)
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// healthHandler returns server health status
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbStatus := "healthy"
	dbError := ""
	if err := s.storage.Ping(); err != nil {
		dbStatus = "unhealthy"
		dbError = err.Error()
		log.Printf("Database health check failed: %v", err)
	}
	
	// Overall status is healthy only if all components are healthy
	overallStatus := "healthy"
	if dbStatus != "healthy" {
		overallStatus = "unhealthy"
	}
	
	response := map[string]interface{}{
		"status":   overallStatus,
		"service":  "mailpulse-relay",
		"message":  "SMTP relay is running (AUTH REQUIRED - NOT AN OPEN RELAY)",
		"database": map[string]interface{}{
			"status": dbStatus,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	// Add error details if database is unhealthy
	if dbError != "" {
		response["database"].(map[string]interface{})["error"] = dbError
	}
	
	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// quotaUsageHandler returns quota usage for a project
func (s *Server) quotaUsageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	
	if projectID == "" {
		http.Error(w, "Project ID required", http.StatusBadRequest)
		return
	}
	
	// Get quota usage from storage
	usage, err := s.storage.GetQuotaUsage(projectID)
	if err != nil {
		log.Printf("Failed to get quota usage for project %s: %v", projectID, err)
		http.Error(w, "Failed to get quota usage", http.StatusInternalServerError)
		return
	}
	
	// Calculate usage percentages
	minutePercent := 0.0
	if usage.MinuteLimit > 0 {
		minutePercent = float64(usage.MinuteUsed) / float64(usage.MinuteLimit) * 100
	}
	
	dailyPercent := 0.0
	if usage.DailyLimit > 0 {
		dailyPercent = float64(usage.DailyUsed) / float64(usage.DailyLimit) * 100
	}
	
	response := map[string]interface{}{
		"projectId":           usage.ProjectID,
		"dailyUsed":          usage.DailyUsed,
		"dailyLimit":         usage.DailyLimit,
		"dailyRemaining":     usage.DailyRemaining,
		"minuteUsed":         usage.MinuteUsed,
		"minuteLimit":        usage.MinuteLimit,
		"minuteRemaining":    usage.MinuteRemaining,
		"dailyUsagePercent":  dailyPercent,
		"minuteUsagePercent": minutePercent,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// emailStatsHandler returns email statistics for a project
func (s *Server) emailStatsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	
	if projectID == "" {
		http.Error(w, "Project ID required", http.StatusBadRequest)
		return
	}
	
	// Get emails for this project
	emails, err := s.storage.ListEmails(projectID, 1000, 0) // Get up to 1000 recent emails
	if err != nil {
		log.Printf("Failed to get emails for project %s: %v", projectID, err)
		http.Error(w, "Failed to get email statistics", http.StatusInternalServerError)
		return
	}
	
	// Calculate statistics
	stats := map[string]interface{}{
		"projectId":     projectID,
		"totalEmails":   len(emails),
		"sentEmails":    0,
		"failedEmails":  0,
		"queuedEmails":  0,
		"totalSize":     0,
	}
	
	for _, email := range emails {
		switch email.Status {
		case "delivered", "processed":
			stats["sentEmails"] = stats["sentEmails"].(int) + 1
		case "failed":
			stats["failedEmails"] = stats["failedEmails"].(int) + 1
		case "queued":
			stats["queuedEmails"] = stats["queuedEmails"].(int) + 1
		}
		stats["totalSize"] = stats["totalSize"].(int) + email.Size
	}
	
	// Calculate success rate
	totalProcessed := stats["sentEmails"].(int) + stats["failedEmails"].(int)
	successRate := 0.0
	if totalProcessed > 0 {
		successRate = float64(stats["sentEmails"].(int)) / float64(totalProcessed) * 100
	}
	stats["successRate"] = successRate
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// resendEmailHandler resends a failed email
func (s *Server) resendEmailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["emailId"]
	
	if emailID == "" {
		http.Error(w, "Email ID required", http.StatusBadRequest)
		return
	}
	
	// Get the email from storage
	email, err := s.storage.GetEmail(emailID)
	if err != nil {
		log.Printf("Failed to get email %s for resend: %v", emailID, err)
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}
	
	// Check if email can be resent (not already sent successfully)
	if email.Status == "delivered" {
		http.Error(w, "Email already sent successfully", http.StatusBadRequest)
		return
	}
	
	// Update email status to queued for resend
	err = s.storage.UpdateEmailStatus(emailID, "queued", nil)
	if err != nil {
		log.Printf("Failed to update email status for resend: %v", err)
		http.Error(w, "Failed to queue email for resend", http.StatusInternalServerError)
		return
	}
	
	// Actually forward the email using SMTP
	go func() {
		// Simulate processing time
		time.Sleep(1 * time.Second)
		
		// Use the email forwarder to actually resend the email
		err := s.forwarder.ForwardEmail(email, email.ProjectID)
		
		if err == nil {
			// Success - mark as sent
			s.storage.UpdateEmailStatus(emailID, "delivered", nil)
			log.Printf("✅ Email %s resent successfully via SMTP", emailID)
		} else {
			// Failed - mark as failed with error
			errorMsg := fmt.Sprintf("SMTP forwarding failed: %s", err.Error())
			s.storage.UpdateEmailStatus(emailID, "failed", &errorMsg)
			log.Printf("❌ Email %s resend failed: %s", emailID, err.Error())
		}
	}()
	
	response := map[string]interface{}{
		"success": true,
		"message": "Email queued for resend",
		"emailId": emailID,
	}
	
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listProjectsHandler returns all projects
func (s *Server) listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := s.storage.ListAllProjects()
	if err != nil {
		log.Printf("Failed to list projects: %v", err)
		http.Error(w, "Failed to list projects", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// createProjectHandler creates a new project
func (s *Server) createProjectHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Password     string `json:"password"`
		SMTPHost     string `json:"smtpHost,omitempty"`
		SMTPPort     int    `json:"smtpPort,omitempty"`
		SMTPUser     string `json:"smtpUser,omitempty"`
		SMTPPassword string `json:"smtpPassword,omitempty"`
		QuotaPerMinute int  `json:"quotaPerMinute"`
		QuotaDaily     int  `json:"quotaDaily"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Name == "" {
		http.Error(w, "Project name is required", http.StatusBadRequest)
		return
	}
	
	if req.Password == "" {
		http.Error(w, "SMTP password is required", http.StatusBadRequest)
		return
	}
	
	// Generate unique project ID and API key
	projectID := generateID()
	apiKey := generateAPIKey()
	
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(req.Password)), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}
	
	// Set default quotas if not provided
	quotaPerMinute := req.QuotaPerMinute
	if quotaPerMinute == 0 {
		quotaPerMinute = 10
	}
	quotaDaily := req.QuotaDaily
	if quotaDaily == 0 {
		quotaDaily = 500
	}
	
	// Encrypt SMTP password if provided
	var smtpPasswordEnc *string
	if req.SMTPPassword != "" {
		encrypted, err := crypto.EncryptSMTPPassword(req.SMTPPassword)
		if err != nil {
			log.Printf("Failed to encrypt SMTP password: %v", err)
			http.Error(w, "Failed to encrypt SMTP password", http.StatusInternalServerError)
			return
		}
		smtpPasswordEnc = &encrypted
	}
	
	// Create project
	project := &storage.Project{
		ID:             projectID,
		Name:           req.Name,
		Description:    req.Description,
		APIKey:         apiKey,
		PasswordHash:   stringPtr(string(hashedPassword)),
		SMTPHost:       stringPtrFromString(req.SMTPHost),
		SMTPPort:       intPtrFromInt(req.SMTPPort),
		SMTPUser:       stringPtrFromString(req.SMTPUser),
		SMTPPasswordEnc: smtpPasswordEnc,
		QuotaDaily:     quotaDaily,
		QuotaPerMinute: quotaPerMinute,
		Status:         "active",
		UserID:         nil,
		CreatedAt:      time.Now(),
		LastUsedAt:     nil,
	}
	
	// Save to database
	if err := s.storage.CreateProject(project); err != nil {
		log.Printf("Failed to create project: %v", err)
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}
	
	// Reload auth manager projects so new project is available immediately
	if err := s.authManager.ReloadProjects(); err != nil {
		log.Printf("⚠️  Failed to reload projects in auth manager: %v", err)
	} else {
		log.Printf("✅ Reloaded projects in auth manager")
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// getProjectHandler returns a specific project
func (s *Server) getProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	
	project, err := s.storage.GetProject(projectID)
	if err != nil {
		log.Printf("Failed to get project %s: %v", projectID, err)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// updateProjectHandler updates a project
func (s *Server) updateProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Get existing project
	project, err := s.storage.GetProject(projectID)
	if err != nil {
		log.Printf("Failed to get project %s: %v", projectID, err)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	
	// Apply updates
	if name, ok := updates["name"].(string); ok && name != "" {
		project.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		project.Description = desc
	}
	if status, ok := updates["status"].(string); ok {
		project.Status = status
	}
	
	// SMTP Configuration updates
	if smtpHost, ok := updates["smtpHost"].(string); ok {
		project.SMTPHost = stringPtrFromString(smtpHost)
	}
	if smtpPort, ok := updates["smtpPort"].(float64); ok {
		project.SMTPPort = intPtrFromInt(int(smtpPort))
	}
	if smtpUser, ok := updates["smtpUser"].(string); ok {
		project.SMTPUser = stringPtrFromString(smtpUser)
	}
	if smtpPassword, ok := updates["smtpPassword"].(string); ok && smtpPassword != "" {
		// Encrypt the SMTP password before storing
		encryptedPassword, err := crypto.EncryptSMTPPassword(smtpPassword)
		if err != nil {
			log.Printf("Failed to encrypt SMTP password: %v", err)
			http.Error(w, "Failed to encrypt SMTP password", http.StatusInternalServerError)
			return
		}
		project.SMTPPasswordEnc = &encryptedPassword
	}
	
	// Quota updates
	if quotaDaily, ok := updates["quotaDaily"].(float64); ok && quotaDaily >= 0 {
		project.QuotaDaily = int(quotaDaily)
	}
	if quotaPerMinute, ok := updates["quotaPerMinute"].(float64); ok && quotaPerMinute >= 0 {
		project.QuotaPerMinute = int(quotaPerMinute)
	}
	
	// Update in database
	if err := s.storage.UpdateProject(projectID, project); err != nil {
		log.Printf("Failed to update project %s: %v", projectID, err)
		http.Error(w, "Failed to update project", http.StatusInternalServerError)
		return
	}
	
	// Reload auth manager projects to reflect status changes
	if err := s.authManager.ReloadProjects(); err != nil {
		log.Printf("⚠️  Failed to reload projects in auth manager: %v", err)
	} else {
		log.Printf("✅ Reloaded projects in auth manager after update")
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// deleteProjectHandler deletes a project
func (s *Server) deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	
	// Delete from storage (soft delete - sets status to 'deleted')
	if err := s.storage.DeleteProject(projectID); err != nil {
		log.Printf("Failed to delete project %s: %v", projectID, err)
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"message": "Project deleted successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listEmailsHandler returns emails (with optional project filter)
func (s *Server) listEmailsHandler(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	
	var emails []*storage.Email
	var err error
	
	if projectID != "" {
		emails, err = s.storage.ListEmails(projectID, 50, 0)
	} else {
		emails, err = s.storage.ListAllEmails(50, 0)
	}
	
	if err != nil {
		log.Printf("Failed to list emails: %v", err)
		http.Error(w, "Failed to list emails", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(emails)
}

// handleOptions handles preflight OPTIONS requests
func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	log.Printf("Explicit OPTIONS handler called for %s", r.URL.Path)
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "86400")
	
	w.WriteHeader(http.StatusOK)
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

// Start starts the HTTP API server
func (s *Server) Start(addr string) error {
	log.Printf("🌐 Starting HTTP API server on %s", addr)
	log.Printf("📊 API Endpoints:")
	log.Printf("   GET %s/health - Server health check", addr)
	log.Printf("   GET %s/api/quota/{projectId} - Quota usage", addr)
	log.Printf("   GET %s/api/emails/stats/{projectId} - Email statistics", addr)
	
	return http.ListenAndServe(addr, s.router)
}