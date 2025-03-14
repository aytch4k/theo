package file

import (
	"github.com/gold2th/theo/internal/storage"
)

// NewStorage creates a new file storage backend
func NewStorage(config *storage.StorageConfig) (storage.Storage, error) {
	return NewFileStorage(config)
}
