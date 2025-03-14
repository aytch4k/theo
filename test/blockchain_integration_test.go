package test

import (
	"context"
	"testing"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/blockchain/sync"
	"github.com/gold2th/theo/internal/blockchain/validation"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

// TestBlockchainIntegration tests blockchain validation and synchronization
func TestBlockchainIntegration(t *testing.T) {
	// Use file storage for testing
	config := &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/blockchain-integration",
	}

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

	// Create validator
	validator := validation.NewBlockchainValidator()

	// Create synchronizer
	synchronizer := sync.NewChainSynchronizer(s, sync.WithValidator(validator))

	// Test chain validation and synchronization
	t.Run("ChainValidation", func(t *testing.T) {
		testChainValidation(t, validator)
	})

	t.Run("ChainSynchronization", func(t *testing.T) {
		testChainSynchronization(t, ctx, synchronizer)
	})

	t.Run("BlockSynchronization", func(t *testing.T) {
		testBlockSynchronization(t, ctx, synchronizer)
	})

	t.Run("ChainMerging", func(t *testing.T) {
		testChainMerging(t, ctx, synchronizer)
	})
}

// testChainValidation tests blockchain validation
func testChainValidation(t *testing.T, validator *validation.BlockchainValidator) {
	// Create a valid chain
	blocks := createTestBlockchain(5)

	// Validate chain
	err := validator.ValidateChain(blocks)
	if err != nil {
		t.Fatalf("Failed to validate chain: %v", err)
	}

	// Test invalid hash
	invalidHashBlocks := createTestBlockchain(5)
	invalidHashBlocks[2].Hash = "invalid-hash"
	err = validator.ValidateChain(invalidHashBlocks)
	if err == nil {
		t.Fatalf("Expected validation error for invalid hash")
	}

	// Test invalid previous hash
	invalidPrevHashBlocks := createTestBlockchain(5)
	invalidPrevHashBlocks[2].PreviousHash = "invalid-prev-hash"
	err = validator.ValidateChain(invalidPrevHashBlocks)
	if err == nil {
		t.Fatalf("Expected validation error for invalid previous hash")
	}

	// Test invalid index
	invalidIndexBlocks := createTestBlockchain(5)
	invalidIndexBlocks[2].Index = 10
	err = validator.ValidateChain(invalidIndexBlocks)
	if err == nil {
		t.Fatalf("Expected validation error for invalid index")
	}

	// Test invalid timestamp order
	invalidTimestampBlocks := createTestBlockchain(5)
	invalidTimestampBlocks[2].Timestamp = invalidTimestampBlocks[1].Timestamp - 100
	err = validator.ValidateChain(invalidTimestampBlocks)
	if err == nil {
		t.Fatalf("Expected validation error for invalid timestamp order")
	}
}

// testChainSynchronization tests chain synchronization
func testChainSynchronization(t *testing.T, ctx context.Context, synchronizer *sync.ChainSynchronizer) {
	chainType := "sync-test"
	repoID := "repo-sync"

	// Create remote chain
	remoteChain := createTestBlockchain(5)

	// Sync with remote chain (should keep remote chain as it's longer)
	err := synchronizer.SyncChain(ctx, chainType, repoID, remoteChain)
	if err != nil {
		t.Fatalf("Failed to sync chain: %v", err)
	}

	// Export chain
	exportedChain, err := synchronizer.ExportChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Verify chain length
	if len(exportedChain) != len(remoteChain) {
		t.Fatalf("Expected chain length %d, got %d", len(remoteChain), len(exportedChain))
	}

	// Clean up
	err = synchronizer.Storage().DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// testBlockSynchronization tests block synchronization
func testBlockSynchronization(t *testing.T, ctx context.Context, synchronizer *sync.ChainSynchronizer) {
	chainType := "block-sync-test"
	repoID := "repo-block-sync"

	// Create a chain
	blocks := createTestBlockchain(3)

	// Sync first block
	err := synchronizer.SyncBlock(ctx, chainType, repoID, blocks[0])
	if err != nil {
		t.Fatalf("Failed to sync first block: %v", err)
	}

	// Sync second block
	err = synchronizer.SyncBlock(ctx, chainType, repoID, blocks[1])
	if err != nil {
		t.Fatalf("Failed to sync second block: %v", err)
	}

	// Sync third block
	err = synchronizer.SyncBlock(ctx, chainType, repoID, blocks[2])
	if err != nil {
		t.Fatalf("Failed to sync third block: %v", err)
	}

	// Export chain
	exportedChain, err := synchronizer.ExportChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Verify chain length
	if len(exportedChain) != len(blocks) {
		t.Fatalf("Expected chain length %d, got %d", len(blocks), len(exportedChain))
	}

	// Try to sync invalid block
	invalidBlock := &block.Block{
		Index:        10,
		Timestamp:    time.Now().Unix(),
		Data:         []byte("invalid block"),
		PreviousHash: "invalid-prev-hash",
	}
	invalidBlock.Hash, _ = invalidBlock.CalculateHash()

	err = synchronizer.SyncBlock(ctx, chainType, repoID, invalidBlock)
	if err == nil {
		t.Fatalf("Expected error when syncing invalid block")
	}

	// Clean up
	err = synchronizer.Storage().DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// testChainMerging tests chain merging
func testChainMerging(t *testing.T, ctx context.Context, synchronizer *sync.ChainSynchronizer) {
	chainType := "merge-test"
	repoID := "repo-merge"

	// Create a chain
	blocks := createTestBlockchain(3)

	// Save the chain
	err := synchronizer.Storage().SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		t.Fatalf("Failed to save chain: %v", err)
	}

	// Create a new block to add to the chain
	newBlock := createTestBlock(chainType, repoID, 3, blocks[2].Hash)

	// Add the block using SyncBlock
	err = synchronizer.SyncBlock(ctx, chainType, repoID, newBlock)
	if err != nil {
		t.Fatalf("Failed to sync block: %v", err)
	}

	// Export the updated chain
	updatedChain, err := synchronizer.ExportChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Verify the chain length
	if len(updatedChain) != len(blocks)+1 {
		t.Fatalf("Expected chain length %d, got %d", len(blocks)+1, len(updatedChain))
	}

	// Verify the last block is the one we added
	lastBlock := updatedChain[len(updatedChain)-1]
	if lastBlock.Index != 3 {
		t.Fatalf("Expected last block index 3, got %d", lastBlock.Index)
	}

	// Clean up
	err = synchronizer.Storage().DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// createTestBlockchain creates a test blockchain
func createTestBlockchain(length int) []*block.Block {
	chainType := "test-chain"
	repoID := "test-repo"
	blocks := make([]*block.Block, 0, length)

	prevHash := ""
	for i := 0; i < length; i++ {
		b := createTestBlock(chainType, repoID, uint64(i), prevHash)
		blocks = append(blocks, b)
		prevHash = b.Hash
	}

	return blocks
}
