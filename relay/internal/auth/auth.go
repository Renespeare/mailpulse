package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Project represents a project with API credentials
type Project struct {
	ID               string
	Name             string
	APIKey           string
	APIKeyHash       string
	SMTPHost         string
	SMTPPort         int
	SMTPUsername     string
	SMTPPasswordEnc  string
	QuotaDaily       int
	QuotaPerMinute   int
	Status           string
	RequireIPAllow   bool
	AllowedIPs       []string
	UserID           string
	CreatedAt        time.Time
	LastUsedAt       *time.Time
}

// StorageProject represents a project from storage layer
type StorageProject struct {
	ID             string
	Name           string
	APIKey         string
	PasswordHash   *string
	QuotaDaily     int
	QuotaPerMinute int
	Status         string
}

// ProjectStorage interface for loading projects
type ProjectStorage interface {
	ListAllProjects() ([]*StorageProject, error)
}

// AuthManager handles authentication and authorization
type AuthManager interface {
	ValidateAPIKey(username, password string) (*Project, error)
	CheckRateLimit(projectID string) error
	IsIPAllowed(projectID string, ip string) bool
	RecordAuthAttempt(ip string, success bool)
	GenerateAPIKey(prefix string) (string, string, error) // key, hash, error
	ReloadProjects() error
}

// InMemoryAuthManager is a basic implementation for testing
type InMemoryAuthManager struct {
	projects     map[string]*Project
	authAttempts map[string][]time.Time
	storage      ProjectStorage
}

// DatabaseAuthManager uses the database for authentication
type DatabaseAuthManager struct {
	storage interface {
		GetProjectByAPIKey(apiKey string) (*Project, error)
	}
	authAttempts map[string][]time.Time
}

// NewInMemoryAuthManager creates a new in-memory auth manager
func NewInMemoryAuthManager(storage ProjectStorage) *InMemoryAuthManager {
	return &InMemoryAuthManager{
		projects:     make(map[string]*Project),
		authAttempts: make(map[string][]time.Time),
		storage:      storage,
	}
}

// LoadProjectFromDB adds a project to the in-memory store from database data
func (m *InMemoryAuthManager) LoadProjectFromDB(id, name, apiKey, passwordHash, status string) {
	project := &Project{
		ID:             id,
		Name:           name,
		APIKey:         apiKey,
		APIKeyHash:     passwordHash, // Store password hash in APIKeyHash field
		Status:         status,
		SMTPHost:       "smtp.gmail.com", // Default values
		SMTPPort:       587,
		QuotaDaily:     500,
		QuotaPerMinute: 10,
		RequireIPAllow: false,
		CreatedAt:      time.Now(),
	}
	m.projects[id] = project
}

// GenerateAPIKey generates a new API key and its bcrypt hash
func (m *InMemoryAuthManager) GenerateAPIKey(prefix string) (string, string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Create API key with prefix
	apiKey := fmt.Sprintf("%s_%s", prefix, base64.URLEncoding.EncodeToString(bytes)[:32])
	
	// Generate bcrypt hash
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash API key: %w", err)
	}
	
	return apiKey, string(hash), nil
}

// ValidateAPIKey validates username (API key) and password
func (m *InMemoryAuthManager) ValidateAPIKey(username, password string) (*Project, error) {
	// Find project by matching the API key directly
	for _, project := range m.projects {
		// Compare the provided username with the stored API key
		if strings.EqualFold(project.APIKey, username) {
			// If project has a password hash, verify the password
			if project.APIKeyHash != "" {
				// Convert password to lowercase for comparison (SMTP servers often uppercase)
				lowercasePassword := strings.ToLower(password)
				err := bcrypt.CompareHashAndPassword([]byte(project.APIKeyHash), []byte(lowercasePassword))
				if err != nil {
					return nil, errors.New("invalid password")
				}
			}
			
			// Check if project is active
			if project.Status != "active" {
				return nil, errors.New("project is not active")
			}
			
			// Update last used time
			now := time.Now()
			project.LastUsedAt = &now
			return project, nil
		}
	}
	
	return nil, errors.New("invalid API credentials")
}

// CheckRateLimit checks if project has exceeded rate limits
func (m *InMemoryAuthManager) CheckRateLimit(projectID string) error {
	// Basic rate limiting implementation
	// This would use Redis in production
	
	project, exists := m.projects[projectID]
	if !exists {
		return errors.New("project not found")
	}
	
	// Simple per-minute check (would be more sophisticated in Redis)
	now := time.Now()
	attempts := 0
	
	// Count recent attempts (last minute)
	for ip, times := range m.authAttempts {
		if strings.HasPrefix(ip, projectID) {
			for _, t := range times {
				if now.Sub(t) < time.Minute {
					attempts++
				}
			}
		}
	}
	
	if attempts >= project.QuotaPerMinute {
		return fmt.Errorf("rate limit exceeded: %d requests per minute", project.QuotaPerMinute)
	}
	
	return nil
}

// IsIPAllowed checks if IP is in project's allowlist
func (m *InMemoryAuthManager) IsIPAllowed(projectID string, ip string) bool {
	project, exists := m.projects[projectID]
	if !exists {
		return false
	}
	
	// If no IP restrictions, allow all
	if !project.RequireIPAllow || len(project.AllowedIPs) == 0 {
		return true
	}
	
	// Check if IP is in allowlist
	for _, allowedIP := range project.AllowedIPs {
		if ip == allowedIP {
			return true
		}
	}
	
	return false
}

// RecordAuthAttempt records an authentication attempt for rate limiting
func (m *InMemoryAuthManager) RecordAuthAttempt(ip string, success bool) {
	now := time.Now()
	
	// Clean old attempts (older than 1 hour)
	cleanTime := now.Add(-time.Hour)
	for key, times := range m.authAttempts {
		var cleanTimes []time.Time
		for _, t := range times {
			if t.After(cleanTime) {
				cleanTimes = append(cleanTimes, t)
			}
		}
		m.authAttempts[key] = cleanTimes
	}
	
	// Record new attempt
	m.authAttempts[ip] = append(m.authAttempts[ip], now)
}

// AddProject adds a project for testing
func (m *InMemoryAuthManager) AddProject(project *Project) {
	m.projects[project.ID] = project
}

// ReloadProjects reloads projects from storage
func (m *InMemoryAuthManager) ReloadProjects() error {
	if m.storage == nil {
		return errors.New("no storage configured")
	}
	
	projects, err := m.storage.ListAllProjects()
	if err != nil {
		return fmt.Errorf("failed to load projects from storage: %w", err)
	}
	
	// Clear existing projects and reload
	m.projects = make(map[string]*Project)
	
	for _, project := range projects {
		passwordHash := ""
		if project.PasswordHash != nil {
			passwordHash = *project.PasswordHash
		}
		m.LoadProjectFromDB(project.ID, project.Name, project.APIKey, passwordHash, project.Status)
	}
	
	return nil
}