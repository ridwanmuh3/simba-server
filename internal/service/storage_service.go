package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type StorageService struct {
	log      *zap.SugaredLogger
	validate *validator.Validate
}

func NewStorageService(logger *zap.SugaredLogger, validate *validator.Validate) *StorageService {
	return &StorageService{
		log:      logger,
		validate: validate,
	}
}

func (s *StorageService) Download(ctx context.Context, request *model.StorageDownloadRequest) ([]byte, error) {
	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	basePath, _ := filepath.Abs("./storage/letters")
	targetPath := filepath.Join(basePath, request.Filename)

	if !strings.HasPrefix(targetPath, filepath.Clean(basePath)) {
		return nil, exception.ForbiddenFilepathError
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return nil, exception.FileNotFoundError
	}

	blob, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, exception.InternalServerError
	}

	return blob, nil
}
