package security

import (
	"fmt"
	"time"
)

// RateLimiter interface defines rate limiting operations
type RateLimiter interface {
	CheckAuthAttempt(ip string) error
	CheckEmailQuota(projectID string, quotaPerMinute, quotaDaily int) error
	RecordEmailSent(projectID string) error
	GetQuotaUsage(projectID string) (*QuotaUsage, error)
	Close() error
}

// QuotaUsage represents current quota usage statistics
type QuotaUsage struct {
	ProjectID       string
	EmailsThisHour  int
	EmailsToday     int
	AuthAttemptsIP  int
	LastEmailSent   *time.Time
	QuotaPerMinute  int
	QuotaDaily      int
}

// InMemoryRateLimiter provides a simple in-memory rate limiter
type InMemoryRateLimiter struct {
	authAttempts map[string][]time.Time
	emailCounts  map[string][]time.Time
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter() *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		authAttempts: make(map[string][]time.Time),
		emailCounts:  make(map[string][]time.Time),
	}
}

// CheckAuthAttempt checks auth attempts for in-memory limiter
func (m *InMemoryRateLimiter) CheckAuthAttempt(ip string) error {
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	
	// Clean old attempts
	var recentAttempts []time.Time
	for _, attempt := range m.authAttempts[ip] {
		if attempt.After(cutoff) {
			recentAttempts = append(recentAttempts, attempt)
		}
	}
	m.authAttempts[ip] = recentAttempts
	
	// Check limit
	if len(recentAttempts) >= 5 {
		return fmt.Errorf("too many authentication attempts from IP %s", ip)
	}
	
	// Record attempt
	m.authAttempts[ip] = append(m.authAttempts[ip], now)
	return nil
}

// CheckEmailQuota checks email quota for in-memory limiter
func (m *InMemoryRateLimiter) CheckEmailQuota(projectID string, quotaPerMinute, quotaDaily int) error {
	now := time.Now()
	
	// Clean old entries and count recent ones
	var recentEmails []time.Time
	minuteCutoff := now.Add(-time.Minute)
	dayCutoff := now.Add(-24 * time.Hour)
	
	emailsThisMinute := 0
	emailsToday := 0
	
	for _, emailTime := range m.emailCounts[projectID] {
		if emailTime.After(dayCutoff) {
			recentEmails = append(recentEmails, emailTime)
			emailsToday++
			
			if emailTime.After(minuteCutoff) {
				emailsThisMinute++
			}
		}
	}
	
	m.emailCounts[projectID] = recentEmails
	
	// Check quotas
	if emailsThisMinute >= quotaPerMinute {
		return fmt.Errorf("project %s exceeded per-minute quota: %d/%d", 
			projectID, emailsThisMinute, quotaPerMinute)
	}
	
	if emailsToday >= quotaDaily {
		return fmt.Errorf("project %s exceeded daily quota: %d/%d", 
			projectID, emailsToday, quotaDaily)
	}
	
	return nil
}

// RecordEmailSent records email for in-memory limiter
func (m *InMemoryRateLimiter) RecordEmailSent(projectID string) error {
	m.emailCounts[projectID] = append(m.emailCounts[projectID], time.Now())
	return nil
}

// GetQuotaUsage gets usage for in-memory limiter
func (m *InMemoryRateLimiter) GetQuotaUsage(projectID string) (*QuotaUsage, error) {
	now := time.Now()
	minuteCutoff := now.Add(-time.Minute)
	dayCutoff := now.Add(-24 * time.Hour)
	
	emailsThisMinute := 0
	emailsToday := 0
	var lastEmailSent *time.Time
	
	for _, emailTime := range m.emailCounts[projectID] {
		if emailTime.After(dayCutoff) {
			emailsToday++
			
			if emailTime.After(minuteCutoff) {
				emailsThisMinute++
			}
			
			if lastEmailSent == nil || emailTime.After(*lastEmailSent) {
				lastEmailSent = &emailTime
			}
		}
	}
	
	return &QuotaUsage{
		ProjectID:      projectID,
		EmailsThisHour: emailsThisMinute,
		EmailsToday:    emailsToday,
		LastEmailSent:  lastEmailSent,
	}, nil
}

// Close is a no-op for in-memory limiter
func (m *InMemoryRateLimiter) Close() error {
	return nil
}