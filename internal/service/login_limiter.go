package service

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxLoginAttempts = 5
	loginWindow      = time.Minute
	cleanupInterval  = 5 * time.Minute
)

type loginAttemptWindow struct {
	Count     int
	ExpiresAt time.Time
}

type LoginRateLimiter struct {
	mu      sync.Mutex
	windows map[string]loginAttemptWindow
	stopCh  chan struct{}
}

func NewLoginRateLimiter() *LoginRateLimiter {
	l := &LoginRateLimiter{
		windows: make(map[string]loginAttemptWindow),
		stopCh:  make(chan struct{}),
	}
	go l.startCleanup()
	return l
}

func (l *LoginRateLimiter) Shutdown() {
	close(l.stopCh)
}

func (l *LoginRateLimiter) Record(key string) (string, bool) {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.windows[key]
	if attempt.ExpiresAt.IsZero() || !now.Before(attempt.ExpiresAt) {
		attempt = loginAttemptWindow{ExpiresAt: now.Add(loginWindow)}
	}

	if attempt.Count >= maxLoginAttempts {
		retryAfter := int(time.Until(attempt.ExpiresAt).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		l.windows[key] = attempt
		return strconv.Itoa(retryAfter), true
	}

	attempt.Count++
	l.windows[key] = attempt
	return "", false
}

func (l *LoginRateLimiter) ForgetSuccessful(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.windows[key]
	if attempt.Count <= 1 {
		delete(l.windows, key)
		return
	}
	attempt.Count--
	l.windows[key] = attempt
}

func (l *LoginRateLimiter) Cleanup() {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	for key, attempt := range l.windows {
		if !now.Before(attempt.ExpiresAt) {
			delete(l.windows, key)
		}
	}
}

func (l *LoginRateLimiter) startCleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.Cleanup()
		case <-l.stopCh:
			return
		}
	}
}

func NormalizeLoginLimiterUsername(username string) string {
	username = strings.ToLower(strings.TrimSpace(username))

	var builder strings.Builder
	builder.Grow(len(username))
	for i := 0; i < len(username) && builder.Len() < 30; i++ {
		ch := username[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteByte(ch)
		}
	}
	if builder.Len() == 0 {
		return "-"
	}
	return builder.String()
}
