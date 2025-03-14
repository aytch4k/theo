package test

import (
	"context"
	"testing"

	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

// TestSQLiteStorage tests the SQLite storage backend
func TestSQLiteStorage(t *testing.T) {
	// Create SQLite storage config
	config := &storage.StorageConfig{
		Type:             storage.StorageTypeSQLite,
		ConnectionString: "test-data/sqlite-storage-test.db",
	}

	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}

	// Open storage
	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		t.Fatalf("Failed to open SQLite storage: %v", err)
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
