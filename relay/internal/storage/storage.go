package storage

import (
	"time"
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

// QuotaUsage represents quota usage statistics
type QuotaUsage struct {
	ProjectID       string
	DailyUsed      int
	DailyLimit     int
	MinuteUsed     int
	MinuteLimit    int
	DailyRemaining int
	MinuteRemaining int
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
	
	// Quota operations
	GetQuotaUsage(projectID string) (*QuotaUsage, error)
	CheckQuotaLimits(projectID string) error
	
	// Audit operations
	RecordAuditLog(log *AuditLog) error
	GetAuditLogs(projectID *string, limit, offset int) ([]*AuditLog, error)
	
	// Health check
	Ping() error
	Close() error
}