package model

type StorageDownloadRequest struct {
	Filename string `param:"filename" validate:"required,printascii"`
}
