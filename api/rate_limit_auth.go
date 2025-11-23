package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthRateLimiter implements rate limiting specifically for authentication endpoints
// to prevent credential stuffing, brute force attacks, and account enumeration
type AuthRateLimiter struct {
	attempts map[string]*authAttempt
	mu       sync.RWMutex

	// Configuration
	maxAttempts    int           // Maximum attempts allowed
	window         time.Duration // Time window for attempts
	lockoutTime    time.Duration // How long to lock out after max attempts
}

type authAttempt struct {
	count       int
	firstAttempt time.Time
	lockedUntil  time.Time
	mu          sync.Mutex
}

// NewAuthRateLimiter creates a rate limiter for authentication endpoints
func NewAuthRateLimiter() *AuthRateLimiter {
	limiter := &AuthRateLimiter{
		attempts:    make(map[string]*authAttempt),
		maxAttempts: 5,              // 5 attempts
		window:      15 * time.Minute, // per 15 minutes
		lockoutTime: 15 * time.Minute, // 15 minute lockout
	}

	// Start cleanup goroutine to prevent memory leak
	go limiter.cleanup()

	return limiter
}

// cleanup removes old entries to prevent memory leak
func (a *AuthRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		a.mu.Lock()
		now := time.Now()
		for key, attempt := range a.attempts {
			attempt.mu.Lock()
			// Remove if lockout expired and attempt window expired
			if now.After(attempt.lockedUntil) &&
			   now.Sub(attempt.firstAttempt) > a.window {
				delete(a.attempts, key)
			}
			attempt.mu.Unlock()
		}
		a.mu.Unlock()
	}
}

// Check returns true if the request should be allowed, false if rate limited
func (a *AuthRateLimiter) Check(identifier string) (allowed bool, retryAfter time.Duration) {
	now := time.Now()

	// Get or create attempt tracker
	a.mu.Lock()
	attempt, exists := a.attempts[identifier]
	if !exists {
		attempt = &authAttempt{
			count:        0,
			firstAttempt: now,
			lockedUntil:  time.Time{},
		}
		a.attempts[identifier] = attempt
	}
	a.mu.Unlock()

	attempt.mu.Lock()
	defer attempt.mu.Unlock()

	// Check if currently locked out
	if now.Before(attempt.lockedUntil) {
		retryAfter = attempt.lockedUntil.Sub(now)
		return false, retryAfter
	}

	// Reset if window expired
	if now.Sub(attempt.firstAttempt) > a.window {
		attempt.count = 0
		attempt.firstAttempt = now
		attempt.lockedUntil = time.Time{}
	}

	// Increment attempt counter
	attempt.count++

	// Check if exceeded max attempts
	if attempt.count > a.maxAttempts {
		attempt.lockedUntil = now.Add(a.lockoutTime)
		retryAfter = a.lockoutTime
		return false, retryAfter
	}

	return true, 0
}

// RecordFailure records a failed authentication attempt
func (a *AuthRateLimiter) RecordFailure(identifier string) {
	// Check already increments the counter, so this is a no-op
	// Kept for semantic clarity in handlers
}

// Reset clears the rate limit for an identifier (e.g., after successful login)
func (a *AuthRateLimiter) Reset(identifier string) {
	a.mu.Lock()
	delete(a.attempts, identifier)
	a.mu.Unlock()
}

// AuthRateLimitMiddleware creates a middleware for authentication endpoints
func AuthRateLimitMiddleware(limiter *AuthRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address as primary identifier
		ip := c.ClientIP()

		// Check rate limit
		allowed, retryAfter := limiter.Check(ip)

		if !allowed {
			// Set rate limit headers
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.maxAttempts))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(retryAfter).Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "Too many requests",
				Message: fmt.Sprintf("Too many authentication attempts. Please try again in %v", retryAfter.Round(time.Second)),
			})
			c.Abort()
			return
		}

		// Set rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.maxAttempts))
		// Note: We can't easily get remaining count without race conditions
		// So we don't set X-RateLimit-Remaining here

		c.Next()
	}
}
