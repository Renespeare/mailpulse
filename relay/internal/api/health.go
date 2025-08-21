package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

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