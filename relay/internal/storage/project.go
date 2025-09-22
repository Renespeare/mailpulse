package storage

import (
	"fmt"
)

// GetProject retrieves a project by ID
func (s *PostgreSQLStorage) GetProject(id string) (*Project, error) {
	query := `
		SELECT id, name, description, api_key_enc, password_hash, smtp_host, smtp_port, smtp_user, 
		       smtp_password_enc, quota_daily, quota_per_minute, status, user_id, created_at, last_used_at
		FROM projects
		WHERE id = $1
	`
	
	project := &Project{}
	err := s.db.QueryRow(query, id).Scan(
		&project.ID, &project.Name, &project.Description, &project.APIKeyEnc,
		&project.PasswordHash, &project.SMTPHost, &project.SMTPPort, &project.SMTPUser,
		&project.SMTPPasswordEnc, &project.QuotaDaily, &project.QuotaPerMinute, &project.Status,
		&project.UserID, &project.CreatedAt, &project.LastUsedAt,
	)
	
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	
	return project, nil
}

// ListAllProjects retrieves all projects
func (s *PostgreSQLStorage) ListAllProjects() ([]*Project, error) {
	query := `
		SELECT id, name, description, api_key_enc, password_hash, smtp_host, smtp_port, smtp_user, 
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
			&project.ID, &project.Name, &project.Description, &project.APIKeyEnc,
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
		INSERT INTO projects (id, name, description, api_key_enc, password_hash, smtp_host, smtp_port, smtp_user, 
		                     smtp_password_enc, quota_daily, quota_per_minute, status, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	
	_, err := s.db.Exec(query,
		project.ID, project.Name, project.Description, project.APIKeyEnc, project.PasswordHash,
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

// GetQuotaUsage retrieves current quota usage for a project
func (s *PostgreSQLStorage) GetQuotaUsage(projectID string) (*QuotaUsage, error) {
	// First get project limits
	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	
	// Count emails sent in the last 24 hours
	dailyQuery := `
	
	SELECT COUNT(*) FROM emails 
		WHERE project_id = $1 AND sent_at > NOW() - INTERVAL '24 hours'
	`
	var dailyUsed int
	if err := s.db.QueryRow(dailyQuery, projectID).Scan(&dailyUsed); err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}
	
	// Count emails sent in the last minute
	minuteQuery := `
		SELECT COUNT(*) FROM emails 
		WHERE project_id = $1 AND sent_at > NOW() - INTERVAL '1 minute'
	`
	var minuteUsed int
	if err := s.db.QueryRow(minuteQuery, projectID).Scan(&minuteUsed); err != nil {
		return nil, fmt.Errorf("failed to get minute usage: %w", err)
	}
	
	quota := &QuotaUsage{
		ProjectID:       projectID,
		DailyUsed:      dailyUsed,
		DailyLimit:     project.QuotaDaily,
		MinuteUsed:     minuteUsed,
		MinuteLimit:    project.QuotaPerMinute,
		DailyRemaining: project.QuotaDaily - dailyUsed,
		MinuteRemaining: project.QuotaPerMinute - minuteUsed,
	}
	
	// Ensure remaining counts don't go negative
	if quota.DailyRemaining < 0 {
		quota.DailyRemaining = 0
	}
	if quota.MinuteRemaining < 0 {
		quota.MinuteRemaining = 0
	}
	
	return quota, nil
}

// CheckQuotaLimits checks if a project has exceeded its quotas
func (s *PostgreSQLStorage) CheckQuotaLimits(projectID string) error {
	quota, err := s.GetQuotaUsage(projectID)
	if err != nil {
		return fmt.Errorf("failed to check quotas: %w", err)
	}
	
	if quota.DailyRemaining <= 0 {
		return fmt.Errorf("daily quota exceeded: %d/%d emails used", quota.DailyUsed, quota.DailyLimit)
	}
	
	if quota.MinuteRemaining <= 0 {
		return fmt.Errorf("per-minute quota exceeded: %d/%d emails used", quota.MinuteUsed, quota.MinuteLimit)
	}
	
	return nil
}