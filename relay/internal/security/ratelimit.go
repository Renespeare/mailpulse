package security

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
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

// RedisRateLimiter implements rate limiting using Redis
type RedisRateLimiter struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(redisURL string) (*RedisRateLimiter, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	
	client := redis.NewClient(opt)
	ctx := context.Background()
	
	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	return &RedisRateLimiter{
		client: client,
		ctx:    ctx,
	}, nil
}

// CheckAuthAttempt checks if IP has exceeded auth attempt limits
func (r *RedisRateLimiter) CheckAuthAttempt(ip string) error {
	key := fmt.Sprintf("auth_attempts:%s", ip)
	
	// Get current attempts count
	attempts, err := r.client.Get(r.ctx, key).Int()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get auth attempts: %w", err)
	}
	
	// Allow up to 5 attempts per minute
	if attempts >= 5 {
		return fmt.Errorf("too many authentication attempts from IP %s", ip)
	}
	
	// Increment and set expiry
	pipe := r.client.Pipeline()
	pipe.Incr(r.ctx, key)
	pipe.Expire(r.ctx, key, time.Minute)
	
	if _, err := pipe.Exec(r.ctx); err != nil {
		return fmt.Errorf("failed to record auth attempt: %w", err)
	}
	
	return nil
}

// CheckEmailQuota checks if project has exceeded email quotas
func (r *RedisRateLimiter) CheckEmailQuota(projectID string, quotaPerMinute, quotaDaily int) error {
	now := time.Now()
	
	// Check per-minute quota
	minuteKey := fmt.Sprintf("emails:minute:%s:%d", projectID, now.Unix()/60)
	minuteCount, err := r.client.Get(r.ctx, minuteKey).Int()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get minute quota: %w", err)
	}
	
	if minuteCount >= quotaPerMinute {
		return fmt.Errorf("project %s exceeded per-minute quota: %d/%d", 
			projectID, minuteCount, quotaPerMinute)
	}
	
	// Check daily quota
	dayKey := fmt.Sprintf("emails:daily:%s:%s", projectID, now.Format("2006-01-02"))
	dayCount, err := r.client.Get(r.ctx, dayKey).Int()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get daily quota: %w", err)
	}
	
	if dayCount >= quotaDaily {
		return fmt.Errorf("project %s exceeded daily quota: %d/%d", 
			projectID, dayCount, quotaDaily)
	}
	
	return nil
}

// RecordEmailSent records an email being sent and updates quotas
func (r *RedisRateLimiter) RecordEmailSent(projectID string) error {
	now := time.Now()
	
	// Update counters with pipeline for atomicity
	pipe := r.client.Pipeline()
	
	// Per-minute counter
	minuteKey := fmt.Sprintf("emails:minute:%s:%d", projectID, now.Unix()/60)
	pipe.Incr(r.ctx, minuteKey)
	pipe.Expire(r.ctx, minuteKey, time.Minute*2) // Keep for 2 minutes
	
	// Daily counter
	dayKey := fmt.Sprintf("emails:daily:%s:%s", projectID, now.Format("2006-01-02"))
	pipe.Incr(r.ctx, dayKey)
	pipe.Expire(r.ctx, dayKey, time.Hour*25) // Keep for 25 hours
	
	// Last email timestamp
	lastEmailKey := fmt.Sprintf("last_email:%s", projectID)
	pipe.Set(r.ctx, lastEmailKey, now.Unix(), time.Hour*24)
	
	if _, err := pipe.Exec(r.ctx); err != nil {
		return fmt.Errorf("failed to record email sent: %w", err)
	}
	
	return nil
}

// GetQuotaUsage retrieves current quota usage for a project
func (r *RedisRateLimiter) GetQuotaUsage(projectID string) (*QuotaUsage, error) {
	now := time.Now()
	
	// Get current counters
	minuteKey := fmt.Sprintf("emails:minute:%s:%d", projectID, now.Unix()/60)
	dayKey := fmt.Sprintf("emails:daily:%s:%s", projectID, now.Format("2006-01-02"))
	lastEmailKey := fmt.Sprintf("last_email:%s", projectID)
	
	pipe := r.client.Pipeline()
	minuteCmd := pipe.Get(r.ctx, minuteKey)
	dayCmd := pipe.Get(r.ctx, dayKey)
	lastEmailCmd := pipe.Get(r.ctx, lastEmailKey)
	
	_, err := pipe.Exec(r.ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get quota usage: %w", err)
	}
	
	// Parse results
	emailsThisMinute := 0
	if minuteCmd.Err() == nil {
		emailsThisMinute, _ = strconv.Atoi(minuteCmd.Val())
	}
	
	emailsToday := 0
	if dayCmd.Err() == nil {
		emailsToday, _ = strconv.Atoi(dayCmd.Val())
	}
	
	var lastEmailSent *time.Time
	if lastEmailCmd.Err() == nil {
		if timestamp, err := strconv.ParseInt(lastEmailCmd.Val(), 10, 64); err == nil {
			t := time.Unix(timestamp, 0)
			lastEmailSent = &t
		}
	}
	
	return &QuotaUsage{
		ProjectID:      projectID,
		EmailsThisHour: emailsThisMinute, // Using minute counter as approximate
		EmailsToday:    emailsToday,
		LastEmailSent:  lastEmailSent,
	}, nil
}

// Close closes the Redis connection
func (r *RedisRateLimiter) Close() error {
	return r.client.Close()
}

// InMemoryRateLimiter provides a simple in-memory rate limiter for development
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