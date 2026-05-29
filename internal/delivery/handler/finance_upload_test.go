package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func TestSaveFinanceProofImage(t *testing.T) {
	log := zap.NewNop().Sugar()
	_ = os.RemoveAll("uploads")
	defer os.RemoveAll("uploads")

	app := fiber.New()
	app.Post("/upload", func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("proof_image")
		if err != nil {
			return err
		}

		publicPath, diskPath, err := saveFinanceProofImage(log, c, fileHeader)
		if err != nil {
			return err
		}

		if _, err := os.Stat(diskPath); err != nil {
			t.Fatalf("saved file stat error: %v", err)
		}

		removeFinanceProofImage(log, publicPath)
		if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
			t.Fatalf("saved file remains after cleanup, err=%v", err)
		}

		return c.JSON(map[string]string{"public_path": publicPath})
	})

	req := multipartRequest(t, "/upload", "proof_image", "proof.png", []byte("png-bytes"))
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["public_path"] == "" {
		t.Fatal("public_path missing")
	}
}

func TestSaveFinanceProofImageRejectsInvalidInputs(t *testing.T) {
	log := zap.NewNop().Sugar()

	tests := []struct {
		name string
		file *multipart.FileHeader
	}{
		{
			name: "bad extension",
			file: &multipart.FileHeader{Filename: "proof.txt", Size: 10},
		},
		{
			name: "too large",
			file: &multipart.FileHeader{Filename: "proof.png", Size: financeProofMaxSize + 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := saveFinanceProofImage(log, nil, tt.file); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func multipartRequest(t *testing.T, path, field, filename string, content []byte) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
