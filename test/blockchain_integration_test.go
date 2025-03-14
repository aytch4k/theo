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

// TestBlockchainIntegration tests blockchain integration with storage backends
func TestBlockchainIntegration(t *testing.T) {
	// Test configurations for each storage backend
	configs := map[string]*storage.StorageConfig{
		"file": {
			Type: storage.StorageTypeFile,
			Path: "test-data/blockchain-integration-file",
		},
		"sqlite": {
			Type:             storage.StorageTypeSQLite,
			ConnectionString: "test-data/blockchain-integration-sqlite.db",
		},
		"aerospike": {
			Type:               storage.StorageTypeAerospike,
			AerospikeHost:      getEnv("AEROSPIKE_HOST", "localhost"),
			AerospikePort:      3000,
			AerospikeNamespace: getEnv("AEROSPIKE_NAMESPACE", "theo"),
			AerospikeSet:       "blockchain-integration-test",
		},
		"ipfs": {
			Type:         storage.StorageTypeIPFS,
			IPFSHost:     getEnv("IPFS_HOST", "localhost"),
			IPFSPort:     5001,
			IPFSProtocol: "http",
			Path:         "test-data/blockchain-integration-ipfs-cache",
		},
	}

	// Test each storage backend
	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			testBlockchainIntegration(t, config)
		})
	}
}

// testBlockchainIntegration tests blockchain integration with a specific storage backend
func testBlockchainIntegration(t *testing.T, config *storage.StorageConfig) {
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
	synchronizer := sync.NewChainSynchronizer(s)

	// Test blockchain validation and synchronization
	t.Run("ValidationAndSync", func(t *testing.T) {
		// Create chain
		chainType := "blockchain-integration"
		repoID := "repo1"
		blocks := createTestBlocks(chainType, repoID, 3)

		// Validate chain
		err := validator.ValidateChain(blocks)
		if err != nil {
			t.Fatalf("Chain validation failed: %v", err)
		}

		// Save chain
		err = s.SaveChain(ctx, chainType, repoID, blocks)
		if err != nil {
			t.Fatalf("Failed to save chain: %v", err)
		}

		// Create new block
		newBlock := createTestBlock(chainType, repoID, 3, blocks[2].Hash)

		// Validate new block
		err = validator.ValidateBlock(blocks[2], newBlock)
		if err != nil {
			t.Fatalf("Block validation failed: %v", err)
		}

		// Sync new block
		err = synchronizer.SyncBlock(ctx, chainType, repoID, newBlock)
		if err != nil {
			t.Fatalf("Failed to sync block: %v", err)
		}

		// Load synced chain
		syncedChain, err := s.LoadChain(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to load synced chain: %v", err)
		}

		// Verify synced chain
		if len(syncedChain) != len(blocks)+1 {
			t.Fatalf("Expected %d blocks, got %d", len(blocks)+1, len(syncedChain))
		}

		// Validate synced chain
		err = validator.ValidateChain(syncedChain)
		if err != nil {
			t.Fatalf("Synced chain validation failed: %v", err)
		}

		// Clean up
		err = s.DeleteChain(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to delete chain: %v", err)
		}
	})

	// Test IPFS synchronization
	t.Run("IPFSSync", func(t *testing.T) {
		// Skip if not IPFS storage
		if config.Type != storage.StorageTypeIPFS {
			t.Skip("Skipping IPFS sync test for non-IPFS storage")
		}

		// Create chain
		chainType := "ipfs-sync"
		repoID := "repo1"
		blocks := createTestBlocks(chainType, repoID, 3)

		// Save chain
		err := s.SaveChain(ctx, chainType, repoID, blocks)
		if err != nil {
			t.Fatalf("Failed to save chain: %v", err)
		}

		// Set IPFS CID
		testCID := "QmTest123456789"
		err = s.SetIPFSCID(ctx, chainType, repoID, testCID)
		if err != nil {
			t.Fatalf("Failed to set IPFS CID: %v", err)
		}

		// Sync with IPFS
		err = synchronizer.SyncWithIPFS(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to sync with IPFS: %v", err)
		}

		// Load synced chain
		syncedChain, err := s.LoadChain(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to load synced chain: %v", err)
		}

		// Verify synced chain
		if len(syncedChain) != len(blocks) {
			t.Fatalf("Expected %d blocks, got %d", len(blocks), len(syncedChain))
		}

		// Validate synced chain
		err = validator.ValidateChain(syncedChain)
		if err != nil {
			t.Fatalf("Synced chain validation failed: %v", err)
		}

		// Clean up
		err = s.DeleteChain(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to delete chain: %v", err)
		}
	})

	// Test periodic synchronization
	t.Run("PeriodicSync", func(t *testing.T) {
		// Create chain
		chainType := "periodic-sync"
		repoID := "repo1"
		blocks := createTestBlocks(chainType, repoID, 3)

		// Save chain
		err := s.SaveChain(ctx, chainType, repoID, blocks)
		if err != nil {
			t.Fatalf("Failed to save chain: %v", err)
		}

		// Create context with timeout
		syncCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		// Start periodic sync
		syncCount := 0
		syncFunc := func(ctx context.Context, chainType, repoID string) error {
			syncCount++
			return nil
		}

		// Run periodic sync in a goroutine
		go synchronizer.PeriodicSync(syncCtx, chainType, repoID, 500*time.Millisecond, syncFunc)

		// Wait for sync to run a few times
		time.Sleep(1500 * time.Millisecond)

		// Verify sync count
		if syncCount < 2 {
			t.Fatalf("Expected at least 2 syncs, got %d", syncCount)
		}

		// Clean up
		err = s.DeleteChain(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to delete chain: %v", err)
		}
	})

	// Test linked JSON with blockchain
	t.Run("LinkedJSON", func(t *testing.T) {
		// Create chain with linked JSON data
		chainType := "linked-json-blockchain"
		repoID := "repo1"
		blocks := make([]*block.Block, 3)
		prevHash := ""

		for i := 0; i < 3; i++ {
			// Create linked JSON data
			linkedData := map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "Person",
				"name":     "Person " + string(rune(65+i)), // A, B, C
				"knows": map[string]interface{}{
					"@type": "Person",
					"name":  "Friend " + string(rune(65+i)),
					"url":   "https://example.com/person/" + string(rune(65+i)),
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

		// Validate chain
		err := validator.ValidateChain(blocks)
		if err != nil {
			t.Fatalf("Chain validation failed: %v", err)
		}

		// Save chain
		err = s.SaveChain(ctx, chainType, repoID, blocks)
		if err != nil {
			t.Fatalf("Failed to save chain: %v", err)
		}

		// Create new linked JSON block
		linkedData := map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "Person",
			"name":     "Person D",
			"knows": map[string]interface{}{
				"@type": "Person",
				"name":  "Friend D",
				"url":   "https://example.com/person/D",
			},
			"previousPerson": map[string]interface{}{
				"@id": blocks[2].ID,
			},
		}

		// Convert data to json.RawMessage
		dataJSON, _ := json.Marshal(linkedData)

		newBlock := &block.Block{
			Context:      "https://devhub-git.org/contexts/block.jsonld",
			ID:           "chain://" + chainType + "/" + repoID + "/blockD",
			Index:        3,
			Timestamp:    time.Now().Unix(),
			Data:         json.RawMessage(dataJSON),
			PreviousHash: blocks[2].Hash,
		}

		// Calculate hash
		hash, _ := newBlock.CalculateHash()
		newBlock.Hash = hash

		// Validate new block
		err = validator.ValidateBlock(blocks[2], newBlock)
		if err != nil {
			t.Fatalf("Block validation failed: %v", err)
		}

		// Sync new block
		err = synchronizer.SyncBlock(ctx, chainType, repoID, newBlock)
		if err != nil {
			t.Fatalf("Failed to sync block: %v", err)
		}

		// Load synced chain
		syncedChain, err := s.LoadChain(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to load synced chain: %v", err)
		}

		// Verify synced chain
		if len(syncedChain) != len(blocks)+1 {
			t.Fatalf("Expected %d blocks, got %d", len(blocks)+1, len(syncedChain))
		}

		// Verify linked JSON data in new block
		lastBlock := syncedChain[len(syncedChain)-1]
		var data map[string]interface{}
		err = json.Unmarshal(lastBlock.Data, &data)
		if err != nil {
			t.Fatalf("Failed to unmarshal data: %v", err)
		}

		if data["name"] != "Person D" {
			t.Fatalf("Expected name Person D, got %v", data["name"])
		}

		prevPerson, ok := data["previousPerson"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected previousPerson to be a map, got %T", data["previousPerson"])
		}

		if prevPerson["@id"] != blocks[2].ID {
			t.Fatalf("Expected previousPerson @id %s, got %v", blocks[2].ID, prevPerson["@id"])
		}

		// Clean up
		err = s.DeleteChain(ctx, chainType, repoID)
		if err != nil {
			t.Fatalf("Failed to delete chain: %v", err)
		}
	})
}
