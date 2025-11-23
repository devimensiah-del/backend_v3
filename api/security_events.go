package api

import (
	"time"

	"github.com/rs/zerolog"
)

// SecurityEvent represents a security-relevant event for audit logging
type SecurityEvent struct {
	Timestamp     time.Time
	EventType     string // "auth_failure", "authz_failure", "admin_action", "suspicious_activity"
	UserID        string
	IP            string
	UserAgent     string
	Resource      string
	Action        string
	Success       bool
	FailureReason string
	Metadata      map[string]interface{}
}

// SecurityEventLogger provides structured logging for security events
type SecurityEventLogger struct {
	logger zerolog.Logger
}

// NewSecurityEventLogger creates a new security event logger
func NewSecurityEventLogger(logger zerolog.Logger) *SecurityEventLogger {
	return &SecurityEventLogger{
		logger: logger,
	}
}

// LogAuthFailure logs a failed authentication attempt
func (s *SecurityEventLogger) LogAuthFailure(ip, userAgent, resource, reason string) {
	s.logger.Warn().
		Time("timestamp", time.Now()).
		Str("event_type", "auth_failure").
		Str("ip", ip).
		Str("user_agent", userAgent).
		Str("resource", resource).
		Str("action", "authenticate").
		Bool("success", false).
		Str("failure_reason", reason).
		Msg("SECURITY: Authentication failure")
}

// LogAuthSuccess logs a successful authentication
func (s *SecurityEventLogger) LogAuthSuccess(userID, ip, userAgent, resource string) {
	s.logger.Info().
		Time("timestamp", time.Now()).
		Str("event_type", "auth_success").
		Str("user_id", userID).
		Str("ip", ip).
		Str("user_agent", userAgent).
		Str("resource", resource).
		Str("action", "authenticate").
		Bool("success", true).
		Msg("SECURITY: Authentication success")
}

// LogAuthzFailure logs a failed authorization attempt (insufficient permissions)
func (s *SecurityEventLogger) LogAuthzFailure(userID, role, ip, resource, action, reason string) {
	s.logger.Warn().
		Time("timestamp", time.Now()).
		Str("event_type", "authz_failure").
		Str("user_id", userID).
		Str("role", role).
		Str("ip", ip).
		Str("resource", resource).
		Str("action", action).
		Bool("success", false).
		Str("failure_reason", reason).
		Msg("SECURITY: Authorization failure")
}

// LogAdminAction logs administrative actions for audit trail
func (s *SecurityEventLogger) LogAdminAction(userID, role, ip, resource, action string, metadata map[string]interface{}) {
	event := s.logger.Info().
		Time("timestamp", time.Now()).
		Str("event_type", "admin_action").
		Str("user_id", userID).
		Str("role", role).
		Str("ip", ip).
		Str("resource", resource).
		Str("action", action).
		Bool("success", true)

	// Add metadata fields
	if metadata != nil {
		for key, value := range metadata {
			event = event.Interface(key, value)
		}
	}

	event.Msg("SECURITY: Admin action performed")
}

// LogSuspiciousActivity logs potentially malicious behavior
func (s *SecurityEventLogger) LogSuspiciousActivity(ip, userAgent, resource, reason string, metadata map[string]interface{}) {
	event := s.logger.Warn().
		Time("timestamp", time.Now()).
		Str("event_type", "suspicious_activity").
		Str("ip", ip).
		Str("user_agent", userAgent).
		Str("resource", resource).
		Str("reason", reason)

	// Add metadata fields
	if metadata != nil {
		for key, value := range metadata {
			event = event.Interface(key, value)
		}
	}

	event.Msg("SECURITY: Suspicious activity detected")
}

// LogRateLimitExceeded logs rate limit violations
func (s *SecurityEventLogger) LogRateLimitExceeded(ip, userAgent, endpoint string, attemptCount int) {
	s.logger.Warn().
		Time("timestamp", time.Now()).
		Str("event_type", "rate_limit_exceeded").
		Str("ip", ip).
		Str("user_agent", userAgent).
		Str("endpoint", endpoint).
		Int("attempt_count", attemptCount).
		Msg("SECURITY: Rate limit exceeded")
}

// LogAccountLockout logs when an account is locked due to failed attempts
func (s *SecurityEventLogger) LogAccountLockout(identifier, ip, reason string, lockoutDuration time.Duration) {
	s.logger.Warn().
		Time("timestamp", time.Now()).
		Str("event_type", "account_lockout").
		Str("identifier", identifier).
		Str("ip", ip).
		Str("reason", reason).
		Dur("lockout_duration", lockoutDuration).
		Msg("SECURITY: Account locked out")
}

// LogPasswordResetRequest logs password reset attempts
func (s *SecurityEventLogger) LogPasswordResetRequest(email, ip, userAgent string, exists bool) {
	s.logger.Info().
		Time("timestamp", time.Now()).
		Str("event_type", "password_reset_request").
		Str("email", email).
		Str("ip", ip).
		Str("user_agent", userAgent).
		Bool("account_exists", exists).
		Msg("SECURITY: Password reset requested")
}

// LogSessionEvent logs session-related events (login, logout, token refresh)
func (s *SecurityEventLogger) LogSessionEvent(eventType, userID, sessionID, ip, userAgent string) {
	s.logger.Info().
		Time("timestamp", time.Now()).
		Str("event_type", eventType).
		Str("user_id", userID).
		Str("session_id", sessionID).
		Str("ip", ip).
		Str("user_agent", userAgent).
		Msg("SECURITY: Session event")
}
