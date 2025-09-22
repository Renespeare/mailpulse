package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Renespeare/mailpulse/relay/internal/storage"
	"github.com/gorilla/mux"
)

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

// allEmailStatsHandler returns email statistics across all projects
func (s *Server) allEmailStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Get emails for all projects
	emails, err := s.storage.ListAllEmails(10000, 0) // Get up to 10000 recent emails for stats
	if err != nil {
		log.Printf("Failed to get all emails for stats: %v", err)
		http.Error(w, "Failed to get email statistics", http.StatusInternalServerError)
		return
	}
	
	// Calculate statistics
	stats := map[string]interface{}{
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
	
	// Record audit log for email resend request
	s.recordAuditLog(r, "email_resend_requested", &email.ProjectID, map[string]interface{}{
		"email_id":   emailID,
		"message_id": email.MessageID,
		"from":       email.From,
		"to":         email.To,
		"subject":    email.Subject,
	})
	
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

// listEmailsHandler returns emails with pagination, search, and status filtering
func (s *Server) listEmailsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	projectID := r.URL.Query().Get("project")
	searchQuery := r.URL.Query().Get("search")
	statusFilter := r.URL.Query().Get("status")
	
	// Parse pagination parameters
	limit := 20 // default page size
	offset := 0 // default offset
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	
	// Get emails with search, status filtering, and pagination
	var emails []*storage.Email
	var totalCount int
	var err error
	
	if projectID != "" {
		emails, totalCount, err = s.storage.SearchEmailsWithStatus(projectID, searchQuery, statusFilter, limit, offset)
	} else {
		emails, totalCount, err = s.storage.SearchAllEmailsWithStatus(searchQuery, statusFilter, limit, offset)
	}
	
	if err != nil {
		log.Printf("Failed to search emails: %v", err)
		http.Error(w, "Failed to search emails", http.StatusInternalServerError)
		return
	}
	
	// Create paginated response
	response := map[string]interface{}{
		"emails":     emails,
		"totalCount": totalCount,
		"limit":      limit,
		"offset":     offset,
		"hasMore":    offset+len(emails) < totalCount,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}