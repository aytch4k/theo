package test

import (
	"context"
	"testing"

	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

// TestFileStorage tests the file storage backend
func TestFileStorage(t *testing.T) {
	// Create file storage config
	config := &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/file-storage-test",
	}

	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Open storage
	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		t.Fatalf("Failed to open file storage: %v", err)
	}
	defer s.Close()

	// Run storage tests
	t.Run("ChainOperations", func(t *testing.T) {
		testChainOperations(t, ctx, s)
	})

	t.Run("BlockOperations", func(t *testing.T) {
		testBlockOperations(t, ctx, s)
	})

	t.Run("SearchOperations", func(t *testing.T) {
		testSearchOperations(t, ctx, s)
	})

	t.Run("ExportImportOperations", func(t *testing.T) {
		testExportImportOperations(t, ctx, s)
	})

	t.Run("StorageInfo", func(t *testing.T) {
		testStorageInfo(t, ctx, s)
	})

	// Run integration tests
	t.Run("IPFSCIDOperations", func(t *testing.T) {
		testIPFSCIDOperations(t, ctx, s)
	})

	t.Run("QueryOptions", func(t *testing.T) {
		testQueryOptions(t, ctx, s)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, ctx, s)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		testConcurrentOperations(t, ctx, s)
	})

	t.Run("LinkedJSONSupport", func(t *testing.T) {
		testLinkedJSONSupport(t, ctx, s)
	})
}
