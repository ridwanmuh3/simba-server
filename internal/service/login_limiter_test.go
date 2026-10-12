package service

import (
	"strconv"
	"testing"
	"time"
)

// ─── Positive: basic rate limiting ──────────────────────────────────────

func TestLoginRateLimiter_Basic(t *testing.T) {
	rl := NewLoginRateLimiter()
	defer rl.Shutdown()

	key := "127.0.0.1|testuser"

	for i := 0; i < 5; i++ {
		_, limited := rl.Record(key)
		if limited {
			t.Fatalf("attempt %d: should not be limited", i+1)
		}
	}

	_, limited := rl.Record(key)
	if !limited {
		t.Fatal("6th attempt should be rate limited")
	}
}

// ─── Positive: forget decrements count ──────────────────────────────────

func TestLoginRateLimiter_ForgetSuccessful(t *testing.T) {
	rl := NewLoginRateLimiter()
	defer rl.Shutdown()

	key := "127.0.0.1|testuser2"

	for i := 0; i < 3; i++ {
		_, limited := rl.Record(key)
		if limited {
			t.Fatalf("attempt %d: should not be limited", i+1)
		}
	}

	rl.ForgetSuccessful(key)

	for i := 0; i < 3; i++ {
		_, limited := rl.Record(key)
		if limited {
			t.Fatalf("after forget, attempt %d: should not be limited", i+1)
		}
	}

	_, limited := rl.Record(key)
	if !limited {
		t.Fatal("should be limited after 5 attempts")
	}
}

// ─── Positive: keys are independent ────────────────────────────────────

func TestLoginRateLimiter_DifferentKeysIndependent(t *testing.T) {
	rl := NewLoginRateLimiter()
	defer rl.Shutdown()

	key1 := "127.0.0.1|user1"
	key2 := "127.0.0.1|user2"

	for i := 0; i < 5; i++ {
		_, limited := rl.Record(key1)
		if limited {
			t.Fatalf("key1 attempt %d should not be limited", i+1)
		}
	}
	_, limited := rl.Record(key1)
	if !limited {
		t.Fatal("key1 should be limited")
	}

	for i := 0; i < 5; i++ {
		_, limited := rl.Record(key2)
		if limited {
			t.Fatalf("key2 attempt %d should not be limited", i+1)
		}
	}
}

// ─── Positive: cleanup removes expired entries ──────────────────────────

func TestLoginRateLimiter_CleanupRemovesExpired(t *testing.T) {
	rl := &LoginRateLimiter{
		windows: make(map[string]loginAttemptWindow),
		stopCh:  make(chan struct{}),
	}

	key := "127.0.0.1|cleanupuser"
	rl.windows[key] = loginAttemptWindow{
		Count:     3,
		ExpiresAt: time.Now().Add(-time.Minute),
	}

	rl.Cleanup()

	if _, exists := rl.windows[key]; exists {
		t.Fatal("expired entry should be cleaned up")
	}
}

// ─── Negative: forget on non-existent key ───────────────────────────────

func TestLoginRateLimiter_ForgetNonExistentKey(t *testing.T) {
	rl := NewLoginRateLimiter()
	defer rl.Shutdown()

	// Should not panic
	rl.ForgetSuccessful("127.0.0.1|ghost")
}

// ─── Edge: retry-after value is positive integer ────────────────────────

func TestLoginRateLimiter_RetryAfterIsPositiveInteger(t *testing.T) {
	rl := NewLoginRateLimiter()
	defer rl.Shutdown()

	key := "127.0.0.1|retryafter"
	for i := 0; i < 5; i++ {
		rl.Record(key)
	}

	retryAfter, limited := rl.Record(key)
	if !limited {
		t.Fatal("should be limited")
	}

	secs, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("retry-after should be integer, got %q: %v", retryAfter, err)
	}
	if secs < 1 {
		t.Fatalf("retry-after should be >= 1, got %d", secs)
	}
}

// ─── Edge: window expiry resets count ───────────────────────────────────

func TestLoginRateLimiter_WindowExpiryResetsCount(t *testing.T) {
	rl := &LoginRateLimiter{
		windows: make(map[string]loginAttemptWindow),
		stopCh:  make(chan struct{}),
	}

	key := "127.0.0.1|expiry"
	rl.windows[key] = loginAttemptWindow{
		Count:     4,
		ExpiresAt: time.Now().Add(-time.Second), // Expired
	}

	// Record should reset the window since it's expired
	_, limited := rl.Record(key)
	if limited {
		t.Fatal("expired window should be reset, not limited")
	}
}

// ─── Edge: cleanup keeps non-expired entries ───────────────────────────

func TestLoginRateLimiter_CleanupKeepsValidEntries(t *testing.T) {
	rl := &LoginRateLimiter{
		windows: make(map[string]loginAttemptWindow),
		stopCh:  make(chan struct{}),
	}

	key := "127.0.0.1|valid"
	rl.windows[key] = loginAttemptWindow{
		Count:     3,
		ExpiresAt: time.Now().Add(time.Minute),
	}

	rl.Cleanup()

	if _, exists := rl.windows[key]; !exists {
		t.Fatal("non-expired entry should be kept")
	}
}

// ─── Edge: shutdown stops cleanup goroutine ─────────────────────────────

func TestLoginRateLimiter_ShutdownStopsCleanup(t *testing.T) {
	rl := NewLoginRateLimiter()
	// Should not panic or deadlock
	rl.Shutdown()
}

// ─── NormalizeLoginLimiterUsername ──────────────────────────────────────

func TestNormalizeLoginLimiterUsername(t *testing.T) {
	repeat := func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "TestUser", "testuser"},
		{"trim spaces", "  UPPER  ", "upper"},
		{"strip special chars", "User@Name!", "username"},
		{"empty returns dash", "", "-"},
		{"truncates to 30", repeat("a", 50), repeat("a", 30)},
		{"numbers preserved", "User123", "user123"},
		{"all special chars returns dash", "@#$%", "-"},
		{"single char", "a", "a"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeLoginLimiterUsername(tc.input)
			if got != tc.expected {
				t.Errorf("NormalizeLoginLimiterUsername(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}