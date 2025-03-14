package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/blockchain/sync"
	"github.com/gold2th/theo/internal/blockchain/validation"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

// TestBlockchainValidation tests blockchain validation
func TestBlockchainValidation(t *testing.T) {
	// Create validator
	validator := validation.NewBlockchainValidator()

	// Create valid chain
	validChain := createTestBlocks("validation", "repo1", 3)

	// Test valid chain
	err := validator.ValidateChain(validChain)
	if err != nil {
		t.Fatalf("Valid chain validation failed: %v", err)
	}

	// Create invalid chain with wrong hash
	invalidHashChain := createTestBlocks("validation", "repo2", 3)
	invalidHashChain[1].Hash = "invalid-hash"

	// Test invalid hash
	err = validator.ValidateChain(invalidHashChain)
	if err == nil {
		t.Fatalf("Invalid hash validation should fail")
	}

	// Create invalid chain with wrong previous hash
	invalidPrevHashChain := createTestBlocks("validation", "repo3", 3)
	invalidPrevHashChain[1].PreviousHash = "invalid-prev-hash"

	// Test invalid previous hash
	err = validator.ValidateChain(invalidPrevHashChain)
	if err == nil {
		t.Fatalf("Invalid previous hash validation should fail")
	}

	// Create invalid chain with wrong index
	invalidIndexChain := createTestBlocks("validation", "repo4", 3)
	invalidIndexChain[1].Index = 10

	// Test invalid index
	err = validator.ValidateChain(invalidIndexChain)
	if err == nil {
		t.Fatalf("Invalid index validation should fail")
	}

	// Test single block validation
	prevBlock := validChain[0]
	nextBlock := validChain[1]

	err = validator.ValidateBlock(prevBlock, nextBlock)
	if err != nil {
		t.Fatalf("Valid block validation failed: %v", err)
	}

	// Test invalid block
	invalidBlock := createTestBlock("validation", "repo5", 1, prevBlock.Hash)
	invalidBlock.Hash = "invalid-hash"

	err = validator.ValidateBlock(prevBlock, invalidBlock)
	if err == nil {
		t.Fatalf("Invalid block validation should fail")
	}
}

// TestBlockchainSynchronization tests blockchain synchronization
func TestBlockchainSynchronization(t *testing.T) {
	// Create storage backend
	config := &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/blockchain-sync",
	}

	s, err := factory.NewStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer s.Close()

	// Create synchronizer
	synchronizer := sync.NewChainSynchronizer(s)

	// Test chain synchronization
	t.Run("ChainSync", func(t *testing.T) {
		// Create local chain
		localChain := createTestBlocks("sync", "repo1", 2)

		// Save local chain
		err := s.SaveChain(ctx, "sync", "repo1", localChain)
		if err != nil {
			t.Fatalf("Failed to save local chain: %v", err)
		}

		// Create remote chain (longer)
		remoteChain := createTestBlocks("sync", "repo1", 3)

		// Sync with remote chain
		err = synchronizer.SyncChain(ctx, "sync", "repo1", remoteChain)
		if err != nil {
			t.Fatalf("Failed to sync chain: %v", err)
		}

		// Load synced chain
		syncedChain, err := s.LoadChain(ctx, "sync", "repo1")
		if err != nil {
			t.Fatalf("Failed to load synced chain: %v", err)
		}

		// Verify synced chain (should be remote chain with longest strategy)
		if len(syncedChain) != len(remoteChain) {
			t.Fatalf("Expected %d blocks, got %d", len(remoteChain), len(syncedChain))
		}

		// Clean up
		err = s.DeleteChain(ctx, "sync", "repo1")
		if err != nil {
			t.Fatalf("Failed to delete chain: %v", err)
		}
	})

	// Test block synchronization
	t.Run("BlockSync", func(t *testing.T) {
		// Create chain
		chain := createTestBlocks("sync", "repo2", 2)

		// Save chain
		err := s.SaveChain(ctx, "sync", "repo2", chain)
		if err != nil {
			t.Fatalf("Failed to save chain: %v", err)
		}

		// Create new block
		newBlock := createTestBlock("sync", "repo2", 2, chain[1].Hash)

		// Sync new block
		err = synchronizer.SyncBlock(ctx, "sync", "repo2", newBlock)
		if err != nil {
			t.Fatalf("Failed to sync block: %v", err)
		}

		// Load synced chain
		syncedChain, err := s.LoadChain(ctx, "sync", "repo2")
		if err != nil {
			t.Fatalf("Failed to load synced chain: %v", err)
		}

		// Verify synced chain
		if len(syncedChain) != len(chain)+1 {
			t.Fatalf("Expected %d blocks, got %d", len(chain)+1, len(syncedChain))
		}

		// Verify new block
		lastBlock := syncedChain[len(syncedChain)-1]
		if lastBlock.Hash != newBlock.Hash {
			t.Fatalf("Expected hash %s, got %s", newBlock.Hash, lastBlock.Hash)
		}

		// Clean up
		err = s.DeleteChain(ctx, "sync", "repo2")
		if err != nil {
			t.Fatalf("Failed to delete chain: %v", err)
		}
	})

	// Test sync strategies
	t.Run("SyncStrategies", func(t *testing.T) {
		// Test KeepLongest strategy
		testSyncStrategy(t, ctx, s, sync.KeepLongest)

		// Test KeepNewest strategy
		testSyncStrategy(t, ctx, s, sync.KeepNewest)

		// Test KeepLocal strategy
		testSyncStrategy(t, ctx, s, sync.KeepLocal)

		// Test KeepRemote strategy
		testSyncStrategy(t, ctx, s, sync.KeepRemote)

		// Test MergeChains strategy
		testSyncStrategy(t, ctx, s, sync.MergeChains)
	})
}

// testSyncStrategy tests a specific sync strategy
func testSyncStrategy(t *testing.T, ctx context.Context, s storage.Storage, strategy sync.SyncStrategy) {
	// Create chain type and repo ID for this strategy
	chainType := "sync-strategy"
	repoID := string(strategy)

	// Create local chain
	localChain := createTestBlocks(chainType, repoID, 2)

	// Modify timestamps to test KeepNewest strategy
	if strategy == sync.KeepNewest {
		// Make local chain newer
		for i := range localChain {
			localChain[i].Timestamp = time.Now().Unix() + int64(i)
		}
	}

	// Save local chain
	err := s.SaveChain(ctx, chainType, repoID, localChain)
	if err != nil {
		t.Fatalf("Failed to save local chain: %v", err)
	}

	// Create remote chain
	remoteChain := createTestBlocks(chainType, repoID, 3)

	// Create synchronizer with strategy
	synchronizer := sync.NewChainSynchronizer(s, sync.WithStrategy(strategy))

	// Sync with remote chain
	err = synchronizer.SyncChain(ctx, chainType, repoID, remoteChain)
	if err != nil {
		t.Fatalf("Failed to sync chain with strategy %s: %v", strategy, err)
	}

	// Load synced chain
	syncedChain, err := s.LoadChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to load synced chain: %v", err)
	}

	// Verify synced chain based on strategy
	switch strategy {
	case sync.KeepLongest:
		if len(syncedChain) != len(remoteChain) {
			t.Fatalf("KeepLongest: Expected %d blocks, got %d", len(remoteChain), len(syncedChain))
		}
	case sync.KeepNewest:
		if len(syncedChain) != len(localChain) {
			t.Fatalf("KeepNewest: Expected %d blocks, got %d", len(localChain), len(syncedChain))
		}
	case sync.KeepLocal:
		if len(syncedChain) != len(localChain) {
			t.Fatalf("KeepLocal: Expected %d blocks, got %d", len(localChain), len(syncedChain))
		}
	case sync.KeepRemote:
		if len(syncedChain) != len(remoteChain) {
			t.Fatalf("KeepRemote: Expected %d blocks, got %d", len(remoteChain), len(syncedChain))
		}
	case sync.MergeChains:
		// For merge, we expect at least the common blocks plus some additional blocks
		if len(syncedChain) <= 2 {
			t.Fatalf("MergeChains: Expected more than 2 blocks, got %d", len(syncedChain))
		}
	}

	// Clean up
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// TestLinkedJSONBlockchain tests blockchain with linked JSON data
func TestLinkedJSONBlockchain(t *testing.T) {
	// Create storage backend
	config := &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/linked-json",
	}

	s, err := factory.NewStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer s.Close()

	// Create chain with linked JSON data
	chainType := "linked-json"
	repoID := "repo1"
	blocks := make([]*block.Block, 3)
	prevHash := ""

	for i := 0; i < 3; i++ {
		// Create linked JSON data
		linkedData := map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "Person",
			"name":     "Person " + string(rune(65+i)), // A, B, C
			"knows": []map[string]interface{}{
				{
					"@type": "Person",
					"name":  "Friend " + string(rune(65+i)),
				},
			},
		}

		// Add references to previous blocks
		if i > 0 {
			linkedData["previousPerson"] = map[string]interface{}{
				"@id": blocks[i-1].ID,
			}
		}

		// Convert data to json.RawMessage
		dataJSON, _ := json.Marshal(linkedData)

		blocks[i] = &block.Block{
			Context:      "https://devhub-git.org/contexts/block.jsonld",
			ID:           "chain://" + chainType + "/" + repoID + "/block" + string(rune(65+i)),
			Index:        uint64(i),
			Timestamp:    time.Now().Unix(),
			Data:         json.RawMessage(dataJSON),
			PreviousHash: prevHash,
		}

		// Calculate hash
		hash, _ := blocks[i].CalculateHash()
		blocks[i].Hash = hash
		prevHash = hash
	}

	// Save chain
	err = s.SaveChain(ctx, chainType, repoID, blocks)
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

	// Verify linked JSON data
	for i, b := range loadedBlocks {
		var data map[string]interface{}
		err := json.Unmarshal(b.Data, &data)
		if err != nil {
			t.Fatalf("Failed to unmarshal data: %v", err)
		}

		// Check context
		if data["@context"] != "https://schema.org" {
			t.Fatalf("Expected @context https://schema.org, got %v", data["@context"])
		}

		// Check type
		if data["@type"] != "Person" {
			t.Fatalf("Expected @type Person, got %v", data["@type"])
		}

		// Check name
		expectedName := "Person " + string(rune(65+i))
		if data["name"] != expectedName {
			t.Fatalf("Expected name %s, got %v", expectedName, data["name"])
		}

		// Check previous person reference
		if i > 0 {
			prevPerson, ok := data["previousPerson"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected previousPerson to be a map, got %T", data["previousPerson"])
			}

			expectedID := blocks[i-1].ID
			if prevPerson["@id"] != expectedID {
				t.Fatalf("Expected previousPerson @id %s, got %v", expectedID, prevPerson["@id"])
			}
		}
	}

	// Create validator
	validator := validation.NewBlockchainValidator()

	// Validate chain
	err = validator.ValidateChain(loadedBlocks)
	if err != nil {
		t.Fatalf("Chain validation failed: %v", err)
	}

	// Clean up
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}
