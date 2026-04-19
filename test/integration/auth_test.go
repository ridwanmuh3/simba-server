package integration_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   string
		wantStatus int
	}{
		{"super admin success", testSuperAdminUsername, testSuperAdminPassword, http.StatusOK},
		{"admin success", testAdminUsername, testAdminPassword, http.StatusOK},
		{"wrong password", testSuperAdminUsername, "wrongpassword", http.StatusUnauthorized},
		{"nonexistent user", "notexistuser99", "somepass", http.StatusUnauthorized},
		{"empty username", "", testAdminPassword, http.StatusBadRequest},
		{"empty password", testAdminUsername, "", http.StatusBadRequest},
		{"username too short", "abc", "validpass", http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]string{"username": tc.username, "password": tc.password}
			resp := doJSON(http.MethodPost, "/api/auth/login", body, nil)
			if resp.StatusCode != tc.wantStatus {
				r := parseResp(resp)
				t.Errorf("status got %d, want %d — error: %v", resp.StatusCode, tc.wantStatus, r.Error)
			}
		})
	}
}

func TestLoginSetsCookies(t *testing.T) {
	body := map[string]string{"username": testAdminUsername, "password": testAdminPassword}
	resp := doJSON(http.MethodPost, "/api/auth/login", body, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d", resp.StatusCode)
	}

	var tokenCookie, refreshCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == "token" {
			tokenCookie = true
		}
		if c.Name == "refresh_token" {
			refreshCookie = true
		}
	}

	if !tokenCookie {
		t.Error("token cookie not set")
	}
	if !refreshCookie {
		t.Error("refresh_token cookie not set")
	}
}

func TestCurrentUser(t *testing.T) {
	t.Run("authenticated returns user", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/auth/_current", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("got %d", resp.StatusCode)
		}
		r := parseResp(resp)
		if r.Status != http.StatusOK {
			t.Errorf("response status got %d, want 200", r.Status)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/auth/_current", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		fakeCookie := &http.Cookie{Name: "token", Value: "totallyfaketoken123456789"}
		resp := doJSON(http.MethodGet, "/api/auth/_current", nil, []*http.Cookie{fakeCookie})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestLogout(t *testing.T) {
	t.Run("logout success", func(t *testing.T) {
		// Login fresh session for this test
		cookies := mustLoginUser(testAdminUsername, testAdminPassword)
		resp := doJSON(http.MethodDelete, "/api/auth/logout", nil, cookies)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("logout got %d", resp.StatusCode)
		}

		// Token should be invalidated — subsequent request returns 401
		resp2 := doJSON(http.MethodGet, "/api/auth/_current", nil, cookies)
		if resp2.StatusCode != http.StatusUnauthorized {
			t.Errorf("after logout got %d, want 401", resp2.StatusCode)
		}
	})

	t.Run("logout without auth returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, "/api/auth/logout", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestRefreshToken(t *testing.T) {
	t.Run("valid refresh token rotates session", func(t *testing.T) {
		cookies := mustLoginUser(testAdminUsername, testAdminPassword)
		resp := doJSON(http.MethodPost, "/api/auth/refresh", nil, cookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("refresh got %d: %v", resp.StatusCode, r.Error)
		}

		var newToken bool
		for _, c := range resp.Cookies() {
			if c.Name == "token" && c.Value != "" {
				newToken = true
			}
		}
		if !newToken {
			t.Error("new token cookie not returned after refresh")
		}
	})

	t.Run("missing refresh token returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodPost, "/api/auth/refresh", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("invalid refresh token returns 401", func(t *testing.T) {
		fakeCookie := &http.Cookie{Name: "refresh_token", Value: "invalidrefreshtoken999"}
		resp := doJSON(http.MethodPost, "/api/auth/refresh", nil, []*http.Cookie{fakeCookie})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestRegister(t *testing.T) {
	t.Run("success with valid secret", func(t *testing.T) {
		body := map[string]string{
			"username": "registertest1",
			"fullname": "Register Test One",
			"role":     "Admin",
			"password": "registerpass",
		}
		resp := doJSON(http.MethodPost, "/api/auth/register", body, nil, map[string]string{
			"X-Reset-Secret": testResetSecret,
		})
		if resp.StatusCode != http.StatusCreated {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}
		// Cleanup
		testDB.Exec("DELETE FROM users WHERE username = 'registertest1'")
	})

	t.Run("missing secret returns 401", func(t *testing.T) {
		body := map[string]string{
			"username": "registertest2",
			"fullname": "Register Test Two",
			"role":     "Admin",
			"password": "registerpass",
		}
		resp := doJSON(http.MethodPost, "/api/auth/register", body, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("wrong secret returns 401", func(t *testing.T) {
		body := map[string]string{
			"username": "registertest3",
			"fullname": "Register Test Three",
			"role":     "Admin",
			"password": "registerpass",
		}
		resp := doJSON(http.MethodPost, "/api/auth/register", body, nil, map[string]string{
			"X-Reset-Secret": "wrongsecret",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("duplicate username returns error", func(t *testing.T) {
		body := map[string]string{
			"username": testAdminUsername,
			"fullname": "Duplicate User",
			"role":     "Admin",
			"password": "somepass",
		}
		resp := doJSON(http.MethodPost, "/api/auth/register", body, nil, map[string]string{
			"X-Reset-Secret": testResetSecret,
		})
		if resp.StatusCode == http.StatusCreated {
			t.Error("expected error for duplicate username, got 201")
		}
	})
}

func TestResetPassword(t *testing.T) {
	// Create a throwaway user to reset
	throwaway := fmt.Sprintf("throwaway%d", 99)
	testDB.Exec(
		"INSERT INTO users (username, fullname, role, password, is_active, created_at, updated_at) VALUES (?, 'Throwaway', 'Admin', 'hash', false, NOW(), NOW())",
		throwaway,
	)
	defer testDB.Exec("DELETE FROM users WHERE username = ?", throwaway)

	t.Run("success with valid secret", func(t *testing.T) {
		body := map[string]string{
			"username":     testSuperAdminUsername,
			"new_password": "newpassword99",
		}
		resp := doJSON(http.MethodPost, "/api/auth/reset-password", body, nil, map[string]string{
			"X-Reset-Secret": testResetSecret,
		})
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}
		// Restore password so subsequent tests that use superAdminCookies keep working
		body2 := map[string]string{
			"username":     testSuperAdminUsername,
			"new_password": testSuperAdminPassword,
		}
		doJSON(http.MethodPost, "/api/auth/reset-password", body2, nil, map[string]string{
			"X-Reset-Secret": testResetSecret,
		})
		// Re-login super admin
		superAdminCookies = mustLoginUser(testSuperAdminUsername, testSuperAdminPassword)
	})

	t.Run("missing secret returns 401", func(t *testing.T) {
		body := map[string]string{"username": testAdminUsername, "new_password": "newpass"}
		resp := doJSON(http.MethodPost, "/api/auth/reset-password", body, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("missing new_password returns 400", func(t *testing.T) {
		body := map[string]string{"username": testAdminUsername}
		resp := doJSON(http.MethodPost, "/api/auth/reset-password", body, nil, map[string]string{
			"X-Reset-Secret": testResetSecret,
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("got %d, want 400", resp.StatusCode)
		}
	})
}
