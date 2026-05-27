package handler

import (
	"mime/multipart"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

const (
	financeProofDir       = "uploads/finances-proof"
	financeProofURLPrefix = "/uploads/finances-proof/"
	financeProofMaxSize   = int64(15 * 1024 * 1024)
)

var financeProofAllowedExt = []string{".png", ".jpg", ".jpeg"}

func saveFinanceProofImage(log *zap.SugaredLogger, c *fiber.Ctx, fileHeader *multipart.FileHeader) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !slices.Contains(financeProofAllowedExt, ext) {
		log.Warnf("invalid uploaded file format: %s", ext)
		return "", "", exception.InvalidFileFormatError
	}

	if fileHeader.Size > financeProofMaxSize {
		log.Warnf("file size exceeded: %d bytes", fileHeader.Size)
		return "", "", exception.ExceedMaximumFileSizeError
	}

	if err := os.MkdirAll(financeProofDir, 0755); err != nil {
		log.Warnf("failed to create upload directory: %v", err)
		return "", "", exception.InternalServerError
	}

	fileName := uuid.New().String() + ext
	diskPath := filepath.Join(financeProofDir, fileName)
	if err := c.SaveFile(fileHeader, diskPath); err != nil {
		log.Warnf("failed to save uploaded file: %v", err)
		return "", "", exception.InternalServerError
	}

	return financeProofURLPrefix + fileName, diskPath, nil
}

func removeSavedFile(log *zap.SugaredLogger, diskPath string) {
	if diskPath == "" {
		return
	}
	if err := os.Remove(diskPath); err != nil && !os.IsNotExist(err) {
		log.Warnf("failed to remove saved file: %v", err)
	}
}

func removeFinanceProofImage(log *zap.SugaredLogger, publicPath string) {
	if publicPath == "" {
		return
	}
	if err := util.DeleteFile(publicPath); err != nil && !os.IsNotExist(err) {
		log.Warnf("failed to delete proof image file: %v", err)
	}
}
