package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

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