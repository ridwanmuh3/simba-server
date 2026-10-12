package integration_test

import (
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
		{"super admin success", testLoginSuperUsername, testAuthTestPassword, http.StatusOK},
		{"admin success", testLoginAdminUsername, testAuthTestPassword, http.StatusOK},
		{"wrong password", testSuperAdminUsername, "wrongpassword", http.StatusUnauthorized},
		{"nonexistent user", "notexistuser99", "Somepass1!", http.StatusUnauthorized},
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
	body := map[string]string{"username": testLoginAdminUsername, "password": testAuthTestPassword}
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

func TestCSRFProtection(t *testing.T) {
	t.Run("unsafe request without csrf returns 403", func(t *testing.T) {
		body := map[string]string{"username": testAdminUsername, "password": testAdminPassword}
		resp := doJSONWithoutCSRF(http.MethodPost, "/api/auth/login", body, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("got %d, want 403", resp.StatusCode)
		}
	})

	t.Run("csrf endpoint returns token cookie", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/auth/csrf", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("got %d, want 200", resp.StatusCode)
		}

		found := false
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "csrf_" && cookie.Value != "" {
				found = true
			}
		}
		if !found {
			t.Error("csrf cookie not set")
		}
	})
}

func TestLoginRateLimit(t *testing.T) {
	body := map[string]string{"username": "ratelimit01", "password": "Wrongpass1!"}
	for i := 0; i < 5; i++ {
		resp := doJSON(http.MethodPost, "/api/auth/login", body, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d got %d, want 401", i+1, resp.StatusCode)
		}
	}

	resp := doJSON(http.MethodPost, "/api/auth/login", body, nil)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", resp.StatusCode)
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
		// Dedicated account: logging in and out destroys this session on purpose.
		cookies := mustLoginUser(testLogoutUsername, testAuthTestPassword)
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
		// Dedicated account: refresh rotates the stored token for this user.
		cookies := mustLoginUser(testRefreshUsername, testAuthTestPassword)
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
			"password": "Register1!",
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
			"password": "Register1!",
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
			"password": "Register1!",
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
			"password": "Somepass1!",
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
	// Dedicated account: resetting a password invalidates that user's session,
	// and it must not be one the rest of the suite depends on.
	const newPassword = "Resetpass1!"

	t.Run("success with valid secret", func(t *testing.T) {
		body := map[string]string{
			"username":     testResetUsername,
			"new_password": newPassword,
		}
		resp := doJSON(http.MethodPost, "/api/auth/reset-password", body, nil, map[string]string{
			"X-Reset-Secret": testResetSecret,
		})
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}

		// The new password must actually work, and the old one must not.
		login := doJSON(http.MethodPost, "/api/auth/login", map[string]string{
			"username": testResetUsername,
			"password": newPassword,
		}, nil)
		if login.StatusCode != http.StatusOK {
			parseResp(login)
			t.Errorf("login with new password got %d, want 200", login.StatusCode)
		}

		stale := doJSON(http.MethodPost, "/api/auth/login", map[string]string{
			"username": testResetUsername,
			"password": testAuthTestPassword,
		}, nil)
		if stale.StatusCode != http.StatusUnauthorized {
			parseResp(stale)
			t.Errorf("login with old password got %d, want 401", stale.StatusCode)
		}
	})

	t.Run("missing secret returns 401", func(t *testing.T) {
		body := map[string]string{"username": testAdminUsername, "new_password": "Newpass1!"}
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
