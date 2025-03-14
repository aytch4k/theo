package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

// TestStorageBackends tests all storage backends
func TestStorageBackends(t *testing.T) {
	// Test configurations for each storage backend
	configs := map[string]*storage.StorageConfig{
		"file": {
			Type: storage.StorageTypeFile,
			Path: "test-data/file-storage",
		},
		"sqlite": {
			Type:             storage.StorageTypeSQLite,
			ConnectionString: "test-data/sqlite-storage.db",
		},
		"aerospike": {
			Type:               storage.StorageTypeAerospike,
			AerospikeHost:      getEnv("AEROSPIKE_HOST", "localhost"),
			AerospikePort:      3000,
			AerospikeNamespace: getEnv("AEROSPIKE_NAMESPACE", "theo"),
			AerospikeSet:       "test",
		},
		"ipfs": {
			Type:         storage.StorageTypeIPFS,
			IPFSHost:     getEnv("IPFS_HOST", "localhost"),
			IPFSPort:     5001,
			IPFSProtocol: "http",
			Path:         "test-data/ipfs-cache",
		},
	}

	// Create test directory
	err := os.MkdirAll("test-data", 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll("test-data")

	// Test each storage backend
	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			testStorageBackend(t, config)
		})
	}
}

// testStorageBackend tests a single storage backend
func testStorageBackend(t *testing.T, config *storage.StorageConfig) {
	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage backend: %v", err)
	}

	// Open storage
	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer s.Close()

	// Test chain operations
	testChainOperations(t, ctx, s)

	// Test block operations
	testBlockOperations(t, ctx, s)

	// Test search operations
	testSearchOperations(t, ctx, s)

	// Test export/import operations
	testExportImportOperations(t, ctx, s)

	// Test storage info
	testStorageInfo(t, ctx, s)
}

// testChainOperations tests chain operations
func testChainOperations(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create test chain
	chainType := "test"
	repoID := "repo1"
	blocks := createTestBlocks(chainType, repoID, 5)

	// Save chain
	err := s.SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		t.Fatalf("Failed to save chain: %v", err)
	}

	// Load chain
	loadedBlocks, err := s.LoadChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to load chain: %v", err)
	}

	// Verify loaded blocks
	if len(loadedBlocks) != len(blocks) {
		t.Fatalf("Expected %d blocks, got %d", len(blocks), len(loadedBlocks))
	}

	// List chains
	chains, err := s.ListChains(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list chains: %v", err)
	}

	// Verify chains
	if len(chains) < 1 {
		t.Fatalf("Expected at least 1 chain, got %d", len(chains))
	}

	// Delete chain
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}

	// Verify chain is deleted
	_, err = s.LoadChain(ctx, chainType, repoID)
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}
}

// testBlockOperations tests block operations
func testBlockOperations(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create test block
	chainType := "test"
	repoID := "repo2"
	b := createTestBlock(chainType, repoID, 0, "")

	// Save block
	err := s.SaveBlock(ctx, chainType, repoID, b)
	if err != nil {
		t.Fatalf("Failed to save block: %v", err)
	}

	// Load block
	loadedBlock, err := s.LoadBlock(ctx, chainType, repoID, b.Hash)
	if err != nil {
		t.Fatalf("Failed to load block: %v", err)
	}

	// Verify loaded block
	if loadedBlock.Hash != b.Hash {
		t.Fatalf("Expected hash %s, got %s", b.Hash, loadedBlock.Hash)
	}

	// Load block by index
	loadedBlockByIndex, err := s.LoadBlockByIndex(ctx, chainType, repoID, 0)
	if err != nil {
		t.Fatalf("Failed to load block by index: %v", err)
	}

	// Verify loaded block by index
	if loadedBlockByIndex.Hash != b.Hash {
		t.Fatalf("Expected hash %s, got %s", b.Hash, loadedBlockByIndex.Hash)
	}

	// Delete block
	err = s.DeleteBlock(ctx, chainType, repoID, b.Hash)
	if err != nil {
		t.Fatalf("Failed to delete block: %v", err)
	}

	// Verify block is deleted
	_, err = s.LoadBlock(ctx, chainType, repoID, b.Hash)
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}
}

// testSearchOperations tests search operations
func testSearchOperations(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create test chain
	chainType := "test"
	repoID := "repo3"
	blocks := createTestBlocks(chainType, repoID, 5)

	// Save chain
	err := s.SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		t.Fatalf("Failed to save chain: %v", err)
	}

	// Search blocks
	searchResults, err := s.SearchBlocks(ctx, "test", nil)
	if err != nil {
		t.Fatalf("Failed to search blocks: %v", err)
	}

	// Verify search results
	if len(searchResults) < 1 {
		t.Fatalf("Expected at least 1 search result, got %d", len(searchResults))
	}

	// Clean up
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// testExportImportOperations tests export/import operations
func testExportImportOperations(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create test chain
	chainType := "test"
	repoID := "repo4"
	blocks := createTestBlocks(chainType, repoID, 5)

	// Save chain
	err := s.SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		t.Fatalf("Failed to save chain: %v", err)
	}

	// Export chain
	exportedData, err := s.ExportChain(ctx, chainType, repoID, "json")
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Delete chain
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}

	// Import chain
	err = s.ImportChain(ctx, chainType, repoID, exportedData, "json")
	if err != nil {
		t.Fatalf("Failed to import chain: %v", err)
	}

	// Verify imported chain
	loadedBlocks, err := s.LoadChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to load chain: %v", err)
	}

	// Verify loaded blocks
	if len(loadedBlocks) != len(blocks) {
		t.Fatalf("Expected %d blocks, got %d", len(blocks), len(loadedBlocks))
	}

	// Clean up
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// testStorageInfo tests storage info
func testStorageInfo(t *testing.T, ctx context.Context, s storage.Storage) {
	// Get storage info
	info, err := s.GetStorageInfo(ctx)
	if err != nil {
		t.Fatalf("Failed to get storage info: %v", err)
	}

	// Verify storage info
	if info == nil {
		t.Fatalf("Expected storage info, got nil")
	}
}

// createTestBlocks creates test blocks
func createTestBlocks(chainType, repoID string, count int) []*block.Block {
	blocks := make([]*block.Block, count)
	prevHash := ""
	for i := 0; i < count; i++ {
		blocks[i] = createTestBlock(chainType, repoID, uint64(i), prevHash)
		prevHash = blocks[i].Hash
	}
	return blocks
}

// createTestBlock creates a test block
func createTestBlock(chainType, repoID string, index uint64, prevHash string) *block.Block {
	data := map[string]interface{}{
		"test_key": "test_value",
		"index":    index,
	}

	// Convert data to json.RawMessage
	dataJSON, _ := json.Marshal(data)

	b := &block.Block{
		Context:      "https://devhub-git.org/contexts/block.jsonld",
		ID:           "chain://" + chainType + "/" + repoID + "/block" + time.Now().String(),
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Data:         json.RawMessage(dataJSON),
		PreviousHash: prevHash,
	}

	// Calculate hash
	hash, _ := b.CalculateHash()
	b.Hash = hash

	return b
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
