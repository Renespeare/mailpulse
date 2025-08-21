package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Renespeare/mailpulse/relay/internal/crypto"
	"github.com/Renespeare/mailpulse/relay/internal/storage"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

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

	// Record audit log for project creation
	s.recordAuditLog(r, "project_created", &project.ID, map[string]interface{}{
		"project_name":     project.Name,
		"quota_daily":      project.QuotaDaily,
		"quota_per_minute": project.QuotaPerMinute,
		"has_smtp_config":  project.SMTPHost != nil,
	})

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

	// Record audit log for project update
	auditDetails := map[string]interface{}{
		"project_name": project.Name,
	}

	// Add specific fields that were updated
	for key, value := range updates {
		switch key {
		case "name", "description", "status":
			auditDetails["updated_"+key] = value
		case "quotaDaily", "quotaPerMinute":
			auditDetails["updated_"+key] = value
		case "smtpHost", "smtpPort", "smtpUser":
			auditDetails["updated_smtp_config"] = true
		case "smtpPassword":
			auditDetails["updated_smtp_password"] = true
		}
	}

	s.recordAuditLog(r, "project_updated", &projectID, auditDetails)

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

	// Get project details before deletion for audit log
	project, err := s.storage.GetProject(projectID)
	if err != nil {
		log.Printf("Failed to get project %s for deletion: %v", projectID, err)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Delete from storage (soft delete - sets status to 'deleted')
	if err := s.storage.DeleteProject(projectID); err != nil {
		log.Printf("Failed to delete project %s: %v", projectID, err)
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	// Record audit log for project deletion
	s.recordAuditLog(r, "project_deleted", &projectID, map[string]interface{}{
		"project_name": project.Name,
		"was_active":   project.Status == "active",
	})

	response := map[string]interface{}{
		"success": true,
		"message": "Project deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

