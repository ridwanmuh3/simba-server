package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name       string
		cookies    []*http.Cookie
		body       map[string]string
		wantStatus int
	}{
		{
			name:    "super admin creates user",
			cookies: superAdminCookies,
			body:    map[string]string{"username": "newadmin01", "fullname": "New Admin One", "role": "Admin", "password": "newadmin01"},
			wantStatus: http.StatusCreated,
		},
		{
			name:    "admin cannot create user (403)",
			cookies: adminCookies,
			body:    map[string]string{"username": "newadmin02", "fullname": "New Admin Two", "role": "Admin", "password": "newadmin02"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "unauthenticated returns 401",
			cookies:    nil,
			body:       map[string]string{"username": "newadmin03", "fullname": "New Admin Three", "role": "Admin", "password": "newadmin03"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing username returns 400",
			cookies:    superAdminCookies,
			body:       map[string]string{"fullname": "No Username", "role": "Admin", "password": "somepass"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid role returns 400",
			cookies:    superAdminCookies,
			body:       map[string]string{"username": "newadmin04", "fullname": "Bad Role", "role": "Manager", "password": "somepass"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "username too short returns 400",
			cookies:    superAdminCookies,
			body:       map[string]string{"username": "abc", "fullname": "Short Name", "role": "Admin", "password": "somepass"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(http.MethodPost, "/api/users", tc.body, tc.cookies)
			if resp.StatusCode != tc.wantStatus {
				r := parseResp(resp)
				t.Errorf("got %d, want %d — error: %v", resp.StatusCode, tc.wantStatus, r.Error)
			}
		})
	}

	// Cleanup users created in this test
	testDB.Exec("DELETE FROM users WHERE username IN ('newadmin01','newadmin02','newadmin03','newadmin04')")
}

func TestCreateUserDuplicateUsername(t *testing.T) {
	body := map[string]string{
		"username": "dupeuser1",
		"fullname": "Dupe User One",
		"role":     "Admin",
		"password": "dupepass",
	}
	defer testDB.Exec("DELETE FROM users WHERE username = 'dupeuser1'")

	resp1 := doJSON(http.MethodPost, "/api/users", body, superAdminCookies)
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first create got %d", resp1.StatusCode)
	}

	resp2 := doJSON(http.MethodPost, "/api/users", body, superAdminCookies)
	if resp2.StatusCode == http.StatusCreated {
		t.Error("expected error for duplicate username, got 201")
	}
}

func TestUpdateUser(t *testing.T) {
	// Create a user to update
	createBody := map[string]string{
		"username": "updatetarget",
		"fullname": "Update Target",
		"role":     "Admin",
		"password": "updatepass",
	}
	createResp := doJSON(http.MethodPost, "/api/users", createBody, superAdminCookies)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: create user got %d", createResp.StatusCode)
	}

	r := parseResp(createResp)
	var data map[string]any
	_ = json.Unmarshal(r.Data, &data)
	userID := int(data["id"].(float64))
	defer testDB.Exec("DELETE FROM users WHERE id = ?", userID)

	t.Run("update fullname success", func(t *testing.T) {
		body := map[string]string{"fullname": "Updated Fullname"}
		resp := doJSON(http.MethodPut, fmt.Sprintf("/api/users/%d", userID), body, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Errorf("got %d: %v", resp.StatusCode, rr.Error)
		}
	})

	t.Run("update non-existent user returns 404", func(t *testing.T) {
		body := map[string]string{"fullname": "Ghost User"}
		resp := doJSON(http.MethodPut, "/api/users/999999", body, superAdminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("admin cannot update user (403)", func(t *testing.T) {
		body := map[string]string{"fullname": "Attempted Update"}
		resp := doJSON(http.MethodPut, fmt.Sprintf("/api/users/%d", userID), body, adminCookies)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("got %d, want 403", resp.StatusCode)
		}
	})
}

func TestDeleteUser(t *testing.T) {
	createBody := map[string]string{
		"username": "deletetarget",
		"fullname": "Delete Target",
		"role":     "Admin",
		"password": "deletepass",
	}
	createResp := doJSON(http.MethodPost, "/api/users", createBody, superAdminCookies)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: got %d", createResp.StatusCode)
	}

	r := parseResp(createResp)
	var data map[string]any
	_ = json.Unmarshal(r.Data, &data)
	userID := int(data["id"].(float64))

	t.Run("delete success", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/users/%d", userID), nil, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Errorf("got %d: %v", resp.StatusCode, rr.Error)
		}
	})

	t.Run("delete already deleted returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/users/%d", userID), nil, superAdminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("delete non-existent user returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, "/api/users/999999", nil, superAdminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("admin cannot delete user (403)", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, "/api/users/1", nil, adminCookies)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("got %d, want 403", resp.StatusCode)
		}
	})
}

func TestFindUserById(t *testing.T) {
	// Get super admin ID from current endpoint
	currentResp := doJSON(http.MethodGet, "/api/auth/_current", nil, superAdminCookies)
	cr := parseResp(currentResp)
	var authData map[string]any
	_ = json.Unmarshal(cr.Data, &authData)
	superAdminID := int(authData["id"].(float64))

	t.Run("find existing user", func(t *testing.T) {
		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/users/%d", superAdminID), nil, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Errorf("got %d: %v", resp.StatusCode, rr.Error)
		}
	})

	t.Run("find non-existent user returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/users/999999", nil, superAdminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("admin cannot access user route (403)", func(t *testing.T) {
		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/users/%d", superAdminID), nil, adminCookies)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("got %d, want 403", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/users/%d", superAdminID), nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestFindAllUsers(t *testing.T) {
	t.Run("list users paginated", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/users?page=1&size=10", nil, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}

		r := parseResp(resp)
		if r.Paging == nil {
			t.Error("paging metadata missing")
		}
	})

	t.Run("filter by fullname", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/users?fullname=Super&page=1&size=10", nil, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got %d", resp.StatusCode)
		}
	})

	t.Run("admin cannot list users (403)", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/users", nil, adminCookies)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("got %d, want 403", resp.StatusCode)
		}
	})
}

func TestGetUsersStats(t *testing.T) {
	t.Run("super admin gets stats", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/users/stats", nil, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}

		r := parseResp(resp)
		var stats map[string]any
		_ = json.Unmarshal(r.Data, &stats)

		if _, ok := stats["users_total"]; !ok {
			t.Error("users_total field missing in response")
		}
	})

	t.Run("admin cannot access stats (403)", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/users/stats", nil, adminCookies)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("got %d, want 403", resp.StatusCode)
		}
	})
}
