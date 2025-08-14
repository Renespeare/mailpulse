package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Email represents an email record in the database
type Email struct {
	ID          string
	MessageID   string
	ProjectID   string
	From        string
	To          []string
	Subject     string
	ContentEnc  []byte
	Size        int
	Status      string
	Error       *string
	Attempts    int
	SentAt      time.Time
	OpenedAt    *time.Time
	ClickedAt   *time.Time
	Metadata    map[string]interface{}
}

// Project represents a project configuration
type Project struct {
	ID               string
	Name             string
	Description      string
	APIKey           string
	PasswordHash     *string
	SMTPHost         *string
	SMTPPort         *int
	SMTPUser         *string
	SMTPPasswordEnc  *string  // Encrypted SMTP provider password
	QuotaDaily       int
	QuotaPerMinute   int
	Status           string
	UserID           *string
	CreatedAt        time.Time
	LastUsedAt       *time.Time
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string
	ProjectID *string
	UserID    *string
	Action    string
	IPAddress string
	UserAgent *string
	Details   map[string]interface{}
	CreatedAt time.Time
}

// Storage interface defines database operations
type Storage interface {
	// Email operations
	StoreEmail(email *Email) error
	GetEmail(id string) (*Email, error)
	ListEmails(projectID string, limit, offset int) ([]*Email, error)
	ListAllEmails(limit, offset int) ([]*Email, error)
	UpdateEmailStatus(id string, status string, error *string) error
	
	// Project operations
	CreateProject(project *Project) error
	GetProject(id string) (*Project, error)
	UpdateProject(id string, project *Project) error
	DeleteProject(id string) error
	ListAllProjects() ([]*Project, error)
	
	// Audit operations
	RecordAuditLog(log *AuditLog) error
	GetAuditLogs(projectID *string, limit, offset int) ([]*AuditLog, error)
	
	// Health check
	Ping() error
	Close() error
}

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

// StoreEmail stores an email record in the database
func (s *PostgreSQLStorage) StoreEmail(email *Email) error {
	query := `
		INSERT INTO emails (id, message_id, project_id, from_email, to_emails, subject, 
		                   content_enc, size, status, error_msg, attempts, sent_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	
	// Convert []string to pq.Array for PostgreSQL
	_, err := s.db.Exec(query,
		email.ID, email.MessageID, email.ProjectID, email.From, 
		fmt.Sprintf("{%s}", joinStrings(email.To, ",")), // Simple array conversion
		email.Subject, email.ContentEnc, email.Size, email.Status,
		email.Error, email.Attempts, email.SentAt, nil) // metadata as nil for now
	
	if err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}
	
	return nil
}

// GetEmail retrieves an email by ID
func (s *PostgreSQLStorage) GetEmail(id string) (*Email, error) {
	query := `
		SELECT id, message_id, project_id, from_email, to_emails, subject,
		       content_enc, size, status, error_msg, attempts, sent_at
		FROM emails WHERE id = $1
	`
	
	row := s.db.QueryRow(query, id)
	
	email := &Email{}
	var toEmails string
	
	err := row.Scan(
		&email.ID, &email.MessageID, &email.ProjectID, &email.From,
		&toEmails, &email.Subject, &email.ContentEnc, &email.Size,
		&email.Status, &email.Error, &email.Attempts, &email.SentAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("email not found")
		}
		return nil, fmt.Errorf("failed to get email: %w", err)
	}
	
	// Parse array string back to slice (simplified)
	email.To = parseArrayString(toEmails)
	
	return email, nil
}

// ListEmails retrieves emails for a project with pagination
func (s *PostgreSQLStorage) ListEmails(projectID string, limit, offset int) ([]*Email, error) {
	query := `
		SELECT id, message_id, project_id, from_email, to_emails, subject, content_enc,
		       size, status, error_msg, attempts, sent_at
		FROM emails 
		WHERE project_id = $1 
		ORDER BY sent_at DESC 
		LIMIT $2 OFFSET $3
	`
	
	rows, err := s.db.Query(query, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()
	
	var emails []*Email
	for rows.Next() {
		email := &Email{}
		var toEmails string
		
		err := rows.Scan(
			&email.ID, &email.MessageID, &email.ProjectID, &email.From,
			&toEmails, &email.Subject, &email.ContentEnc, &email.Size, &email.Status,
			&email.Error, &email.Attempts, &email.SentAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}
		
		email.To = parseArrayString(toEmails)
		emails = append(emails, email)
	}
	
	return emails, nil
}

// ListAllEmails retrieves all emails across projects with pagination
func (s *PostgreSQLStorage) ListAllEmails(limit, offset int) ([]*Email, error) {
	query := `
		SELECT id, message_id, project_id, from_email, to_emails, subject, content_enc,
		       size, status, error_msg, attempts, sent_at
		FROM emails 
		ORDER BY sent_at DESC 
		LIMIT $1 OFFSET $2
	`
	
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list all emails: %w", err)
	}
	defer rows.Close()
	
	var emails []*Email
	for rows.Next() {
		email := &Email{}
		var toEmails string
		
		err := rows.Scan(
			&email.ID, &email.MessageID, &email.ProjectID, &email.From,
			&toEmails, &email.Subject, &email.ContentEnc, &email.Size, &email.Status,
			&email.Error, &email.Attempts, &email.SentAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}
		
		email.To = parseArrayString(toEmails)
		emails = append(emails, email)
	}
	
	return emails, nil
}

// UpdateEmailStatus updates an email's status
func (s *PostgreSQLStorage) UpdateEmailStatus(id string, status string, errorMsg *string) error {
	query := `UPDATE emails SET status = $1, error_msg = $2, attempts = attempts + 1 WHERE id = $3`
	
	_, err := s.db.Exec(query, status, errorMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}
	
	return nil
}

// RecordAuditLog stores an audit log entry
func (s *PostgreSQLStorage) RecordAuditLog(log *AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, project_id, user_id, action, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := s.db.Exec(query, log.ID, log.ProjectID, log.UserID, 
		log.Action, log.IPAddress, log.UserAgent, nil) // details as nil for now
	
	if err != nil {
		return fmt.Errorf("failed to record audit log: %w", err)
	}
	
	return nil
}

// GetAuditLogs retrieves audit logs with pagination
func (s *PostgreSQLStorage) GetAuditLogs(projectID *string, limit, offset int) ([]*AuditLog, error) {
	var query string
	var args []interface{}
	
	if projectID != nil {
		query = `
			SELECT id, project_id, user_id, action, ip_address, user_agent, created_at
			FROM audit_logs 
			WHERE project_id = $1 
			ORDER BY created_at DESC 
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{*projectID, limit, offset}
	} else {
		query = `
			SELECT id, project_id, user_id, action, ip_address, user_agent, created_at
			FROM audit_logs 
			ORDER BY created_at DESC 
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()
	
	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		
		err := rows.Scan(
			&log.ID, &log.ProjectID, &log.UserID, &log.Action,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		
		logs = append(logs, log)
	}
	
	return logs, nil
}

// GetProject retrieves a project by ID
func (s *PostgreSQLStorage) GetProject(id string) (*Project, error) {
	query := `
		SELECT id, name, description, api_key, password_hash, smtp_host, smtp_port, smtp_user, 
		       smtp_password_enc, quota_daily, quota_per_minute, status, user_id, created_at, last_used_at
		FROM projects
		WHERE id = $1
	`
	
	project := &Project{}
	err := s.db.QueryRow(query, id).Scan(
		&project.ID, &project.Name, &project.Description, &project.APIKey,
		&project.PasswordHash, &project.SMTPHost, &project.SMTPPort, &project.SMTPUser,
		&project.SMTPPasswordEnc, &project.QuotaDaily, &project.QuotaPerMinute, &project.Status,
		&project.UserID, &project.CreatedAt, &project.LastUsedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	
	return project, nil
}

// ListAllProjects retrieves all projects
func (s *PostgreSQLStorage) ListAllProjects() ([]*Project, error) {
	query := `
		SELECT id, name, description, api_key, password_hash, smtp_host, smtp_port, smtp_user, 
		       smtp_password_enc, quota_daily, quota_per_minute, status, user_id, created_at, last_used_at
		FROM projects
		WHERE status != 'deleted'
		ORDER BY created_at DESC
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()
	
	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID, &project.Name, &project.Description, &project.APIKey,
			&project.PasswordHash, &project.SMTPHost, &project.SMTPPort, &project.SMTPUser,
			&project.SMTPPasswordEnc, &project.QuotaDaily, &project.QuotaPerMinute, &project.Status,
			&project.UserID, &project.CreatedAt, &project.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}
	
	// Ensure we always return an empty slice instead of nil
	if projects == nil {
		projects = []*Project{}
	}
	
	return projects, nil
}

// CreateProject creates a new project
func (s *PostgreSQLStorage) CreateProject(project *Project) error {
	query := `
		INSERT INTO projects (id, name, description, api_key, password_hash, smtp_host, smtp_port, smtp_user, 
		                     smtp_password_enc, quota_daily, quota_per_minute, status, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	
	_, err := s.db.Exec(query,
		project.ID, project.Name, project.Description, project.APIKey, project.PasswordHash,
		project.SMTPHost, project.SMTPPort, project.SMTPUser, project.SMTPPasswordEnc,
		project.QuotaDaily, project.QuotaPerMinute, project.Status, project.UserID, project.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	
	return nil
}

// UpdateProject updates an existing project
func (s *PostgreSQLStorage) UpdateProject(id string, project *Project) error {
	query := `
		UPDATE projects 
		SET name = $1, description = $2, password_hash = $3, smtp_host = $4, smtp_port = $5, 
		    smtp_user = $6, smtp_password_enc = $7, quota_daily = $8, quota_per_minute = $9, 
		    status = $10, last_used_at = $11
		WHERE id = $12
	`
	
	_, err := s.db.Exec(query,
		project.Name, project.Description, project.PasswordHash, project.SMTPHost, 
		project.SMTPPort, project.SMTPUser, project.SMTPPasswordEnc, project.QuotaDaily, 
		project.QuotaPerMinute, project.Status, project.LastUsedAt, id)
	
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}
	
	return nil
}

// DeleteProject deletes a project by ID
func (s *PostgreSQLStorage) DeleteProject(id string) error {
	query := `UPDATE projects SET status = 'deleted' WHERE id = $1`
	
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
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

// Helper functions
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