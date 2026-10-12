package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

type notificationEntry struct {
	ID     uint   `json:"id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Level  string `json:"level"`
	IsRead bool   `json:"is_read"`
}

func adminUserID(t *testing.T) uint {
	t.Helper()
	var user entity.User
	if err := testDB.Where("username = ?", testAdminUsername).First(&user).Error; err != nil {
		t.Fatalf("load admin user: %v", err)
	}
	return user.ID
}

func seedNotification(t *testing.T, dapurID uint, userID *uint, title string, isRead bool) entity.Notification {
	t.Helper()
	n := entity.Notification{
		Type:    "LOW_STOCK",
		Title:   title,
		Message: title + " message",
		Level:   "warning",
		IsRead:  isRead,
		UserID:  userID,
		DapurID: dapurID,
	}
	if err := testDB.Create(&n).Error; err != nil {
		t.Fatalf("seed notification %q: %v", title, err)
	}
	return n
}

func listNotifications(t *testing.T, query string) ([]notificationEntry, int64) {
	t.Helper()
	resp := doJSON(http.MethodGet, "/api/notifications"+query, nil, adminCookies)
	if resp.StatusCode != http.StatusOK {
		r := parseResp(resp)
		t.Fatalf("list notifications got %d: %v", resp.StatusCode, r.Error)
	}
	r := parseResp(resp)

	var items []notificationEntry
	if err := json.Unmarshal(r.Data, &items); err != nil {
		t.Fatalf("decode notifications: %v raw=%s", err, r.Data)
	}

	var paging struct {
		Total int64 `json:"total_item"`
	}
	_ = json.Unmarshal(r.Paging, &paging)
	return items, paging.Total
}

// TestNotificationListVisibility guards who may see which rows: broadcast rows
// (user_id NULL) are visible to every user of the dapur, targeted rows only to
// their owner, and rows of another dapur to nobody.
func TestNotificationListVisibility(t *testing.T) {
	me := adminUserID(t)
	otherDapur := entity.Dapur{Name: "Dapur Notif Lain", IsActive: true}
	if err := testDB.Create(&otherDapur).Error; err != nil {
		t.Fatalf("seed other dapur: %v", err)
	}

	var otherUser entity.User
	if err := testDB.Where("id <> ?", me).Where("role = ?", "Admin").First(&otherUser).Error; err != nil {
		t.Fatalf("load other user: %v", err)
	}

	broadcast := seedNotification(t, testDapurID, nil, "notif-broadcast", false)
	mine := seedNotification(t, testDapurID, &me, "notif-mine", false)
	foreign := seedNotification(t, testDapurID, &otherUser.ID, "notif-foreign", false)
	crossDapur := seedNotification(t, otherDapur.ID, nil, "notif-cross-dapur", false)

	items, total := listNotifications(t, "?page=1&size=100&type=LOW_STOCK")

	seen := map[uint]bool{}
	for _, it := range items {
		seen[it.ID] = true
	}

	if total < 2 {
		t.Fatalf("paging total = %d, want at least 2", total)
	}
	if !seen[broadcast.ID] {
		t.Errorf("broadcast notification %d missing from list", broadcast.ID)
	}
	if !seen[mine.ID] {
		t.Errorf("own notification %d missing from list", mine.ID)
	}
	if seen[foreign.ID] {
		t.Errorf("notification %d targeted at another user leaked into list", foreign.ID)
	}
	if seen[crossDapur.ID] {
		t.Errorf("notification %d from another dapur leaked into list", crossDapur.ID)
	}
}

func TestNotificationUnreadCountAndFilter(t *testing.T) {
	me := adminUserID(t)
	unread := seedNotification(t, testDapurID, &me, "notif-unread", false)
	read := seedNotification(t, testDapurID, &me, "notif-read", true)

	resp := doJSON(http.MethodGet, "/api/notifications/unread-count", nil, adminCookies)
	if resp.StatusCode != http.StatusOK {
		r := parseResp(resp)
		t.Fatalf("unread-count got %d: %v", resp.StatusCode, r.Error)
	}
	r := parseResp(resp)
	var counted struct {
		Count int64 `json:"count"`
	}
	_ = json.Unmarshal(r.Data, &counted)
	if counted.Count < 1 {
		t.Fatalf("unread count = %d, want >= 1", counted.Count)
	}

	readItems, _ := listNotifications(t, "?page=1&size=100&is_read=true")
	for _, it := range readItems {
		if !it.IsRead {
			t.Fatalf("is_read=true filter returned unread notification %d", it.ID)
		}
	}

	// is_read=1 must be parsed the same way as is_read=true.
	oneItems, _ := listNotifications(t, "?page=1&size=100&is_read=1")
	for _, it := range oneItems {
		if !it.IsRead {
			t.Fatalf("is_read=1 filter returned unread notification %d", it.ID)
		}
	}

	unreadItems, _ := listNotifications(t, "?page=1&size=100&is_read=false")
	seenUnread := false
	for _, it := range unreadItems {
		if it.IsRead {
			t.Fatalf("is_read=false filter returned read notification %d", it.ID)
		}
		if it.ID == unread.ID {
			seenUnread = true
		}
	}
	if !seenUnread {
		t.Fatalf("seeded unread notification %d missing from is_read=false results", unread.ID)
	}

	_ = read
}

func TestNotificationMarkRead(t *testing.T) {
	me := adminUserID(t)
	targeted := seedNotification(t, testDapurID, &me, "notif-mark-read", false)

	resp := doJSON(http.MethodPatch, fmt.Sprintf("/api/notifications/%d/read", targeted.ID), nil, adminCookies)
	if resp.StatusCode != http.StatusOK {
		r := parseResp(resp)
		t.Fatalf("mark read got %d: %v", resp.StatusCode, r.Error)
	}
	_ = parseResp(resp)

	var reloaded entity.Notification
	if err := testDB.First(&reloaded, targeted.ID).Error; err != nil {
		t.Fatalf("reload notification: %v", err)
	}
	if !reloaded.IsRead {
		t.Fatal("notification not marked read")
	}
}

func TestNotificationInvalidID(t *testing.T) {
	requests := []struct {
		method string
		path   string
	}{
		{http.MethodPatch, "/api/notifications/abc/read"},
		{http.MethodPatch, "/api/notifications/0/read"},
		{http.MethodDelete, "/api/notifications/abc"},
		{http.MethodDelete, "/api/notifications/0"},
	}

	for _, req := range requests {
		resp := doJSON(req.method, req.path, nil, adminCookies)
		if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
			parseResp(resp)
			t.Fatalf("%s %s got %d, want 400/404", req.method, req.path, resp.StatusCode)
		}
		parseResp(resp)
	}
}

// A notification owned by another dapur must not be mutable through this dapur.
func TestNotificationCrossDapurIsolation(t *testing.T) {
	otherDapur := entity.Dapur{Name: "Dapur Notif Isolasi", IsActive: true}
	if err := testDB.Create(&otherDapur).Error; err != nil {
		t.Fatalf("seed other dapur: %v", err)
	}
	foreign := seedNotification(t, otherDapur.ID, nil, "notif-foreign-mutable", false)

	resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/notifications/%d", foreign.ID), nil, adminCookies)
	parseResp(resp)

	var reloaded entity.Notification
	if err := testDB.Unscoped().First(&reloaded, foreign.ID).Error; err != nil {
		t.Fatalf("foreign notification should survive delete: %v", err)
	}
	if reloaded.DeletedAt.Valid {
		t.Fatal("cross-dapur delete soft-deleted a foreign notification")
	}
}
