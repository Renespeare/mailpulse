package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// listAuditLogsHandler returns all audit logs
func (s *Server) listAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); l != 1 || err != nil {
			limit = 50
		}
		if limit > 100 {
			limit = 100 // max limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := fmt.Sscanf(offsetStr, "%d", &offset); o != 1 || err != nil {
			offset = 0
		}
	}
	
	// Get audit logs from storage
	logs, err := s.storage.GetAuditLogs(nil, limit, offset)
	if err != nil {
		log.Printf("Failed to get audit logs: %v", err)
		http.Error(w, "Failed to get audit logs", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// listProjectAuditLogsHandler returns audit logs for a specific project
func (s *Server) listProjectAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]
	
	if projectID == "" {
		http.Error(w, "Project ID required", http.StatusBadRequest)
		return
	}
	
	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); l != 1 || err != nil {
			limit = 50
		}
		if limit > 100 {
			limit = 100 // max limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := fmt.Sscanf(offsetStr, "%d", &offset); o != 1 || err != nil {
			offset = 0
		}
	}
	
	// Get audit logs for project from storage
	logs, err := s.storage.GetAuditLogs(&projectID, limit, offset)
	if err != nil {
		log.Printf("Failed to get audit logs for project %s: %v", projectID, err)
		http.Error(w, "Failed to get audit logs", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}