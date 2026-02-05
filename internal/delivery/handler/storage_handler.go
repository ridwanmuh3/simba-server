package handler

// type StorageHandler struct {
// 	config         *viper.Viper
// 	log            *zap.SugaredLogger
// 	storageService *service.StorageService
// }

// func NewStorageHandler(config *viper.Viper, logger *zap.SugaredLogger, storageService *service.StorageService) *StorageHandler {
// 	return &StorageHandler{
// 		config:         config,
// 		log:            logger,
// 		storageService: storageService,
// 	}
// }

// func (h *StorageHandler) Download(c *fiber.Ctx) error {
// 	request := new(model.StorageDownloadRequest)
// 	if err := c.ParamsParser(request); err != nil {
// 		h.log.Warnf("failed to parse request body: %v", err)
// 		return err
// 	}

// 	blob, err := h.storageService.Download(c.Context(), request)
// 	if err != nil {
// 		h.log.Warnf("failed to download file: %v", err)
// 		return err
// 	}

// 	c.Set("Content-Type", "application/pdf")
// 	c.Set("Content-Disposition", `attachment; filename="surat-pendaftaran-hak-akses.pdf"`)
// 	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")

// 	return c.Send(blob)
// }
