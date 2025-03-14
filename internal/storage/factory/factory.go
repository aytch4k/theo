// Package factory provides factory functions for creating storage backends
package factory

import (
	"fmt"

	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/aerospike"
	"github.com/gold2th/theo/internal/storage/file"
	"github.com/gold2th/theo/internal/storage/indexeddb"
	"github.com/gold2th/theo/internal/storage/ipfs"
	"github.com/gold2th/theo/internal/storage/sqlite"
)

// NewStorage creates a new storage backend based on the provided configuration
func NewStorage(config *storage.StorageConfig) (storage.Storage, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	switch config.Type {
	case storage.StorageTypeFile:
		return file.NewFileStorage(config)
	case storage.StorageTypeSQLite:
		return sqlite.NewSQLiteStorage(config)
	case storage.StorageTypeIndexedDB:
		return indexeddb.NewIndexedDBStorage(config)
	case storage.StorageTypeIPFS:
		return ipfs.NewIPFSStorage(config)
	case storage.StorageTypeAerospike:
		return aerospike.NewAerospikeStorage(config)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}
