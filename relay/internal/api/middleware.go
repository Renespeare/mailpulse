package api

import (
	"log"
	"net/http"
)

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

// handleOptions handles preflight OPTIONS requests
func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	log.Printf("Explicit OPTIONS handler called for %s", r.URL.Path)
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "86400")
	
	w.WriteHeader(http.StatusOK)
}