package storage

import (
	"fmt"
)

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