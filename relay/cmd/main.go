package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Renespeare/mailpulse/relay/internal/api"
	"github.com/Renespeare/mailpulse/relay/internal/auth"
	"github.com/Renespeare/mailpulse/relay/internal/security"
	"github.com/Renespeare/mailpulse/relay/internal/smtp"
	"github.com/Renespeare/mailpulse/relay/internal/storage"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	log.Println("🚀 MailPulse Relay Server starting...")
	log.Println("⚠️  SECURITY: This is NOT an open relay - all connections require authentication")
	
	// Initialize storage
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	
	store, err := storage.NewPostgreSQLStorage(databaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()
	
	log.Println("✅ Database connection established")
	
	// Initialize simple in-memory rate limiter
	rateLimiter := security.NewInMemoryRateLimiter()
	log.Println("✅ Using in-memory rate limiter")
	
	// Initialize authentication manager with storage adapter
	storageAdapter := api.NewStorageAdapter(store)
	authManager := auth.NewInMemoryAuthManager(storageAdapter)
	
	// Load existing projects from database
	log.Println("🔍 Loading projects from database...")
	if err := authManager.ReloadProjects(); err != nil {
		log.Printf("⚠️  Could not load projects from database: %v", err)
	}
	
	
	// Get ports
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "2525"
	}
	
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	
	// Initialize HTTP API server
	apiServer := api.NewServer(authManager, store, rateLimiter)
	
	// Start HTTP API server in background
	go func() {
		if err := apiServer.Start(fmt.Sprintf(":%s", httpPort)); err != nil {
			log.Fatalf("HTTP API server failed: %v", err)
		}
	}()
	
	// Initialize email forwarder
	emailForwarder := smtp.NewEmailForwarder(authManager, store)
	
	// Initialize SMTP server
	smtpConfig := smtp.Config{
		Address:     fmt.Sprintf(":%s", smtpPort),
		AuthManager: authManager,
		Storage:     store,
		RateLimiter: rateLimiter,
		Forwarder:   emailForwarder,
		RequireAuth: true,
		RequireTLS:  false, // Disable TLS for development
	}
	
	smtpServer := smtp.NewServer(smtpConfig)
	
	log.Printf("🔐 Starting SMTP server on port %s (AUTH REQUIRED)", smtpPort)
	log.Println("📧 Ready to accept authenticated email connections")
	
	// Start the SMTP server (blocking)
	if err := smtpServer.Start(); err != nil {
		log.Fatalf("SMTP server failed: %v", err)
	}
}