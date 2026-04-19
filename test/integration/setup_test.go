package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/config"
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/service"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

const (
	testResetSecret        = "test-reset-secret-xyz"
	testSuperAdminUsername = "supertestadmin"
	testSuperAdminPassword = "supertestpass"
	testSuperAdminFullname = "Super Test Admin"
	testAdminUsername      = "admintest01"
	testAdminPassword      = "admintestpass"
	testAdminFullname      = "Admin Test One"
)

var (
	testApp           *fiber.App
	testDB            *gorm.DB
	testLog           *zap.SugaredLogger
	superAdminCookies []*http.Cookie
	adminCookies      []*http.Cookie
)

func TestMain(m *testing.M) {
	// CWD must be the server root so relative paths (uploads/) work correctly
	if err := os.Chdir("../.."); err != nil {
		panic("chdir to server root failed: " + err.Error())
	}

	setupApp()
	truncateTestDB()
	seedTestUsers()

	superAdminCookies = mustLoginUser(testSuperAdminUsername, testSuperAdminPassword)
	adminCookies = mustLoginUser(testAdminUsername, testAdminPassword)

	code := m.Run()

	truncateTestDB()
	os.Exit(code)
}

func setupApp() {
	// Use AutomaticEnv so we don't read the real .env file
	os.Setenv("APP_MODE", "test")
	os.Setenv("APP_NAME", "simba-test")
	os.Setenv("APP_PORT", "3099")
	os.Setenv("APP_PREFORK", "false")
	os.Setenv("APP_CORS_ALLOWED_ORIGINS", "*")
	os.Setenv("APP_RESET_SECRET", testResetSecret)

	os.Setenv("DB_HOST", getEnvOrDefault("TEST_DB_HOST", "localhost"))
	os.Setenv("DB_USER", getEnvOrDefault("TEST_DB_USER", "postgres"))
	os.Setenv("DB_PASSWORD", getEnvOrDefault("TEST_DB_PASSWORD", "postgres"))
	os.Setenv("DB_NAME", getEnvOrDefault("TEST_DB_NAME", "simba_test"))
	os.Setenv("DB_PORT", getEnvOrDefault("TEST_DB_PORT", "5432"))
	os.Setenv("DB_SSL_MODE", "disable")
	os.Setenv("DB_POOL_IDLE", "5")
	os.Setenv("DB_MAX_POOL", "10")
	os.Setenv("DB_MAX_LIFETIME", "300")

	v := viper.New()
	v.AutomaticEnv()

	zapLog, _ := zap.NewDevelopment()
	testLog = zapLog.Sugar()

	testDB = config.NewDB(v, testLog)

	if err := testDB.AutoMigrate(
		&entity.User{},
		&entity.Item{},
		&entity.StockTracking{},
		&entity.Finance{},
		&entity.ActivityLog{},
		&entity.AppSetting{},
		&entity.Invoice{},
		&entity.InvoiceItem{},
	); err != nil {
		panic(fmt.Sprintf("migration failed: %v", err))
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.RegisterValidation("userrole", util.ValidateUserRole); err != nil {
		panic("failed to register userrole validator: " + err.Error())
	}

	testApp = config.NewFiber(v)
	config.Bootstrap(&config.BootstrapConfig{
		DB:       testDB,
		App:      testApp,
		Log:      testLog,
		Validate: validate,
		Config:   v,
	})
}

func truncateTestDB() {
	testDB.Exec(
		"TRUNCATE TABLE users, items, stock_tracks, finances, activity_logs, app_settings, invoices, invoice_items RESTART IDENTITY CASCADE",
	)
}

func seedTestUsers() {
	userRepo := repository.NewUserRepository()
	validate := validator.New(validator.WithRequiredStructEnabled())
	_ = validate.RegisterValidation("userrole", util.ValidateUserRole)
	userSvc := service.NewUserService(testDB, testLog, validate, userRepo)

	ctx := context.Background()
	for _, u := range []model.CreateUserRequest{
		{Username: testSuperAdminUsername, Fullname: testSuperAdminFullname, Role: "Super Admin", Password: testSuperAdminPassword},
		{Username: testAdminUsername, Fullname: testAdminFullname, Role: "Admin", Password: testAdminPassword},
	} {
		req := u
		if _, err := userSvc.Create(ctx, &req); err != nil {
			panic(fmt.Sprintf("seed user %s failed: %v", u.Username, err))
		}
	}
}

func mustLoginUser(username, password string) []*http.Cookie {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req, -1)
	if err != nil {
		panic(fmt.Sprintf("login request error: %v", err))
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("login failed status=%d body=%s", resp.StatusCode, b))
	}
	return resp.Cookies()
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// testResp is a generic parsed API response.
type testResp struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   any             `json:"error"`
	Paging  json.RawMessage `json:"paging"`
}

func parseResp(resp *http.Response) testResp {
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var r testResp
	_ = json.Unmarshal(b, &r)
	return r
}

// doJSON sends a JSON request and returns the raw *http.Response.
func doJSON(method, path string, body any, cookies []*http.Cookie, headers ...map[string]string) *http.Response {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	if len(headers) > 0 {
		for k, v := range headers[0] {
			req.Header.Set(k, v)
		}
	}

	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := testApp.Test(req, -1)
	if err != nil {
		panic(fmt.Sprintf("test request error: %v", err))
	}
	return resp
}

// doMultipart sends a multipart/form-data request.
func doMultipart(method, path string, fields map[string]string, fileField, fileName string, fileContent []byte, cookies []*http.Cookie) *http.Response {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	for k, v := range fields {
		_ = w.WriteField(k, v)
	}

	if fileField != "" && fileContent != nil {
		part, err := w.CreateFormFile(fileField, fileName)
		if err != nil {
			panic("create form file: " + err.Error())
		}
		_, _ = part.Write(fileContent)
	}
	_ = w.Close()

	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := testApp.Test(req, -1)
	if err != nil {
		panic(fmt.Sprintf("multipart request error: %v", err))
	}
	return resp
}

// minimalPNG is a 1x1 transparent PNG, valid for upload tests.
var minimalPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}
