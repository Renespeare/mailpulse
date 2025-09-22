package storage

import (
	"fmt"
)

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
		if err.Error() == "sql: no rows in result set" {
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
		SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
		       e.size, e.status, e.error_msg, e.attempts, e.sent_at
		FROM emails e
		INNER JOIN projects p ON e.project_id = p.id
		WHERE e.project_id = $1 AND p.status != 'deleted'
		ORDER BY e.sent_at DESC 
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
		SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
		       e.size, e.status, e.error_msg, e.attempts, e.sent_at
		FROM emails e
		INNER JOIN projects p ON e.project_id = p.id
		WHERE p.status != 'deleted'
		ORDER BY e.sent_at DESC 
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

// SearchEmails searches emails for a project with pagination
func (s *PostgreSQLStorage) SearchEmails(projectID string, searchQuery string, limit, offset int) ([]*Email, int, error) {
	var countQuery, emailQuery string
	var args []interface{}
	
	if searchQuery == "" {
		// No search - return all emails for project
		countQuery = `SELECT COUNT(*) FROM emails e INNER JOIN projects p ON e.project_id = p.id WHERE e.project_id = $1 AND p.status != 'deleted'`
		emailQuery = `
			SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
			       e.size, e.status, e.error_msg, e.attempts, e.sent_at
			FROM emails e
			INNER JOIN projects p ON e.project_id = p.id
			WHERE e.project_id = $1 AND p.status != 'deleted'
			ORDER BY e.sent_at DESC
			LIMIT $2 OFFSET $3`
		args = []interface{}{projectID, limit, offset}
	} else {
		// Search in from_email, to_emails, and subject
		searchPattern := "%" + searchQuery + "%"
		countQuery = `SELECT COUNT(*) FROM emails e INNER JOIN projects p ON e.project_id = p.id 
		              WHERE e.project_id = $1 AND p.status != 'deleted' 
		              AND (e.from_email ILIKE $2 OR e.subject ILIKE $2 OR array_to_string(e.to_emails, ',') ILIKE $2)`
		emailQuery = `
			SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
			       e.size, e.status, e.error_msg, e.attempts, e.sent_at
			FROM emails e
			INNER JOIN projects p ON e.project_id = p.id
			WHERE e.project_id = $1 AND p.status != 'deleted' 
			AND (e.from_email ILIKE $2 OR e.subject ILIKE $2 OR array_to_string(e.to_emails, ',') ILIKE $2)
			ORDER BY e.sent_at DESC
			LIMIT $3 OFFSET $4`
		args = []interface{}{projectID, searchPattern, limit, offset}
	}
	
	// Get total count
	var totalCount int
	err := s.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}
	
	// Get emails
	rows, err := s.db.Query(emailQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query emails: %w", err)
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
			return nil, 0, fmt.Errorf("failed to scan email: %w", err)
		}
		
		email.To = parseArrayString(toEmails)
		emails = append(emails, email)
	}
	
	return emails, totalCount, nil
}

// SearchAllEmails searches all emails across projects with pagination
func (s *PostgreSQLStorage) SearchAllEmails(searchQuery string, limit, offset int) ([]*Email, int, error) {
	var countQuery, emailQuery string
	var args []interface{}
	
	if searchQuery == "" {
		// No search - return all emails
		countQuery = `SELECT COUNT(*) FROM emails e INNER JOIN projects p ON e.project_id = p.id WHERE p.status != 'deleted'`
		emailQuery = `
			SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
			       e.size, e.status, e.error_msg, e.attempts, e.sent_at
			FROM emails e
			INNER JOIN projects p ON e.project_id = p.id
			WHERE p.status != 'deleted'
			ORDER BY e.sent_at DESC
			LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	} else {
		// Search in from_email, to_emails, and subject
		searchPattern := "%" + searchQuery + "%"
		countQuery = `SELECT COUNT(*) FROM emails e INNER JOIN projects p ON e.project_id = p.id 
		              WHERE p.status != 'deleted' 
		              AND (e.from_email ILIKE $1 OR e.subject ILIKE $1 OR array_to_string(e.to_emails, ',') ILIKE $1)`
		emailQuery = `
			SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
			       e.size, e.status, e.error_msg, e.attempts, e.sent_at
			FROM emails e
			INNER JOIN projects p ON e.project_id = p.id
			WHERE p.status != 'deleted' 
			AND (e.from_email ILIKE $1 OR e.subject ILIKE $1 OR array_to_string(e.to_emails, ',') ILIKE $1)
			ORDER BY e.sent_at DESC
			LIMIT $2 OFFSET $3`
		args = []interface{}{searchPattern, limit, offset}
	}
	
	// Get total count
	var totalCount int
	countArgs := args
	if searchQuery != "" {
		countArgs = args[:1] // Only search pattern for count query
	} else {
		countArgs = []interface{}{} // No args for count query when no search
	}
	
	err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}
	
	// Get emails
	rows, err := s.db.Query(emailQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query emails: %w", err)
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
			return nil, 0, fmt.Errorf("failed to scan email: %w", err)
		}
		
		email.To = parseArrayString(toEmails)
		emails = append(emails, email)
	}
	
	return emails, totalCount, nil
}

// SearchEmailsWithStatus searches emails for a project with pagination and status filtering
func (s *PostgreSQLStorage) SearchEmailsWithStatus(projectID string, searchQuery string, statusFilter string, limit, offset int) ([]*Email, int, error) {
	var countQuery, emailQuery string
	var args []interface{}
	
	// Build base conditions
	baseCondition := "e.project_id = $1 AND p.status != 'deleted'"
	argIndex := 2
	
	// Add search condition if provided
	var searchCondition string
	if searchQuery != "" {
		searchPattern := "%" + searchQuery + "%"
		searchCondition = fmt.Sprintf(" AND (e.from_email ILIKE $%d OR e.subject ILIKE $%d OR array_to_string(e.to_emails, ',') ILIKE $%d)", argIndex, argIndex, argIndex)
		args = append(args, projectID, searchPattern)
		argIndex++
	} else {
		args = append(args, projectID)
	}
	
	// Add status condition if provided
	var statusCondition string
	if statusFilter != "" && statusFilter != "all" {
		statusCondition = fmt.Sprintf(" AND e.status = $%d", argIndex)
		args = append(args, statusFilter)
		argIndex++
	}
	
	// Build count query
	countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM emails e INNER JOIN projects p ON e.project_id = p.id 
	                          WHERE %s%s%s`, baseCondition, searchCondition, statusCondition)
	
	// Build email query
	emailQuery = fmt.Sprintf(`
		SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
		       e.size, e.status, e.error_msg, e.attempts, e.sent_at
		FROM emails e
		INNER JOIN projects p ON e.project_id = p.id
		WHERE %s%s%s
		ORDER BY e.sent_at DESC
		LIMIT $%d OFFSET $%d`, baseCondition, searchCondition, statusCondition, argIndex, argIndex+1)
	
	// Add limit and offset to args
	args = append(args, limit, offset)
	
	// Get total count (exclude limit and offset)
	var totalCount int
	countArgs := args[:len(args)-2]
	err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}
	
	// Get emails
	rows, err := s.db.Query(emailQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query emails: %w", err)
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
			return nil, 0, fmt.Errorf("failed to scan email: %w", err)
		}
		
		email.To = parseArrayString(toEmails)
		emails = append(emails, email)
	}
	
	return emails, totalCount, nil
}

// SearchAllEmailsWithStatus searches all emails across projects with pagination and status filtering
func (s *PostgreSQLStorage) SearchAllEmailsWithStatus(searchQuery string, statusFilter string, limit, offset int) ([]*Email, int, error) {
	var countQuery, emailQuery string
	var args []interface{}
	
	// Build base conditions
	baseCondition := "p.status != 'deleted'"
	argIndex := 1
	
	// Add search condition if provided
	var searchCondition string
	if searchQuery != "" {
		searchPattern := "%" + searchQuery + "%"
		searchCondition = fmt.Sprintf(" AND (e.from_email ILIKE $%d OR e.subject ILIKE $%d OR array_to_string(e.to_emails, ',') ILIKE $%d)", argIndex, argIndex, argIndex)
		args = append(args, searchPattern)
		argIndex++
	}
	
	// Add status condition if provided
	var statusCondition string
	if statusFilter != "" && statusFilter != "all" {
		statusCondition = fmt.Sprintf(" AND e.status = $%d", argIndex)
		args = append(args, statusFilter)
		argIndex++
	}
	
	// Build count query
	countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM emails e INNER JOIN projects p ON e.project_id = p.id 
	                          WHERE %s%s%s`, baseCondition, searchCondition, statusCondition)
	
	// Build email query
	emailQuery = fmt.Sprintf(`
		SELECT e.id, e.message_id, e.project_id, e.from_email, e.to_emails, e.subject, e.content_enc,
		       e.size, e.status, e.error_msg, e.attempts, e.sent_at
		FROM emails e
		INNER JOIN projects p ON e.project_id = p.id
		WHERE %s%s%s
		ORDER BY e.sent_at DESC
		LIMIT $%d OFFSET $%d`, baseCondition, searchCondition, statusCondition, argIndex, argIndex+1)
	
	// Add limit and offset to args
	args = append(args, limit, offset)
	
	// Get total count (exclude limit and offset)
	var totalCount int
	countArgs := args
	if len(args) > 2 {
		countArgs = args[:len(args)-2]
	} else {
		countArgs = []interface{}{}
	}
	
	err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}
	
	// Get emails
	rows, err := s.db.Query(emailQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query emails: %w", err)
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
			return nil, 0, fmt.Errorf("failed to scan email: %w", err)
		}
		
		email.To = parseArrayString(toEmails)
		emails = append(emails, email)
	}
	
	return emails, totalCount, nil
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