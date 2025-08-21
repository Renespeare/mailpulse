package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQLStorage implements Storage interface with PostgreSQL
type PostgreSQLStorage struct {
	db *sql.DB
}

// NewPostgreSQLStorage creates a new PostgreSQL storage instance
func NewPostgreSQLStorage(databaseURL string) (*PostgreSQLStorage, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	
	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	
	storage := &PostgreSQLStorage{db: db}
	
	// Initialize tables if needed
	if err := storage.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}
	
	return storage, nil
}

// initTables creates necessary tables if they don't exist
func (s *PostgreSQLStorage) initTables() error {
	// This is a basic table creation - in production you'd use migrations
	
	// First, add missing columns to existing tables
	migrationQueries := []string{
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS smtp_password_enc TEXT`,
	}
	
	// Run migration queries first (ignore errors for columns that already exist)
	for _, query := range migrationQueries {
		if _, err := s.db.Exec(query); err != nil {
			// Log but don't fail - column might already exist
			log.Printf("Migration query result (may be expected): %v", err)
		}
	}
	
	queries := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			api_key VARCHAR(255) UNIQUE NOT NULL,
			password_hash TEXT,
			smtp_host VARCHAR(255),
			smtp_port INTEGER,
			smtp_user VARCHAR(255),
			smtp_password_enc TEXT,
			quota_daily INTEGER NOT NULL DEFAULT 500,
			quota_per_minute INTEGER NOT NULL DEFAULT 10,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			user_id VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			last_used_at TIMESTAMP WITH TIME ZONE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_api_key ON projects(api_key)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id)`,
		
		`CREATE TABLE IF NOT EXISTS emails (
			id VARCHAR(255) PRIMARY KEY,
			message_id VARCHAR(255) UNIQUE NOT NULL,
			project_id VARCHAR(255) NOT NULL,
			from_email VARCHAR(255) NOT NULL,
			to_emails TEXT[] NOT NULL,
			subject TEXT NOT NULL,
			content_enc BYTEA,
			size INTEGER NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'sent',
			error_msg TEXT,
			attempts INTEGER NOT NULL DEFAULT 1,
			sent_at TIMESTAMP WITH TIME ZONE NOT NULL,
			opened_at TIMESTAMP WITH TIME ZONE,
			clicked_at TIMESTAMP WITH TIME ZONE,
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_project_id ON emails(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_sent_at ON emails(sent_at)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_status ON emails(status)`,
		
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id VARCHAR(255) PRIMARY KEY,
			project_id VARCHAR(255),
			user_id VARCHAR(255),
			action VARCHAR(100) NOT NULL,
			ip_address INET NOT NULL,
			user_agent TEXT,
			details JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_project_id ON audit_logs(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs(ip_address)`,
	}
	
	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}
	
	return nil
}

// Ping checks database connectivity
func (s *PostgreSQLStorage) Ping() error {
	return s.db.Ping()
}

// Close closes the database connection
func (s *PostgreSQLStorage) Close() error {
	return s.db.Close()
}

// Helper functions for PostgreSQL array handling
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, str := range strs[1:] {
		result += sep + str
	}
	return result
}

func parseArrayString(s string) []string {
	// Simplified array parsing - in production use proper PostgreSQL array parsing
	if s == "" || s == "{}" {
		return []string{}
	}
	// Remove braces and split by comma
	s = s[1 : len(s)-1] // Remove { and }
	return []string{s}   // Simplified - just return as single element
}