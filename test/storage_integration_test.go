package test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/blockchain/validation"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

// TestStorageIntegration runs comprehensive integration tests for all storage backends
func TestStorageIntegration(t *testing.T) {
	// Test configurations for each storage backend
	configs := map[string]*storage.StorageConfig{
		"file": {
			Type: storage.StorageTypeFile,
			Path: "test-data/file-storage-integration",
		},
		"sqlite": {
			Type:             storage.StorageTypeSQLite,
			ConnectionString: "test-data/sqlite-storage-integration.db",
		},
		"aerospike": {
			Type:               storage.StorageTypeAerospike,
			AerospikeHost:      getEnv("AEROSPIKE_HOST", "aerospike"),
			AerospikePort:      3000,
			AerospikeNamespace: getEnv("AEROSPIKE_NAMESPACE", "theo"),
			AerospikeSet:       "integration-test",
		},
		"ipfs": {
			Type:         storage.StorageTypeIPFS,
			IPFSHost:     getEnv("IPFS_HOST", "ipfs"),
			IPFSPort:     5001,
			IPFSProtocol: "http",
			Path:         "test-data/ipfs-cache-integration",
		},
	}

	// Test each storage backend
	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			testStorageIntegration(t, config)
		})
	}
}

// testStorageIntegration tests a single storage backend with comprehensive tests
func testStorageIntegration(t *testing.T, config *storage.StorageConfig) {
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

// testIPFSCIDOperations tests IPFS CID operations
func testIPFSCIDOperations(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create test chain
	chainType := "ipfs-test"
	repoID := "repo-ipfs"
	blocks := createTestBlocks(chainType, repoID, 2)

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

	// Get IPFS CID
	cid, err := s.GetIPFSCID(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to get IPFS CID: %v", err)
	}

	// Verify CID
	if cid != testCID {
		t.Fatalf("Expected CID %s, got %s", testCID, cid)
	}

	// Update IPFS CID
	updatedCID := "QmUpdated987654321"
	err = s.SetIPFSCID(ctx, chainType, repoID, updatedCID)
	if err != nil {
		t.Fatalf("Failed to update IPFS CID: %v", err)
	}

	// Get updated IPFS CID
	cid, err = s.GetIPFSCID(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to get updated IPFS CID: %v", err)
	}

	// Verify updated CID
	if cid != updatedCID {
		t.Fatalf("Expected updated CID %s, got %s", updatedCID, cid)
	}

	// Verify CID is included in chain metadata
	chains, err := s.ListChains(ctx, &storage.QueryOptions{
		Filters: []storage.Filter{
			{
				Field:    "chain_type",
				Operator: "eq",
				Value:    chainType,
			},
			{
				Field:    "repo_id",
				Operator: "eq",
				Value:    repoID,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to list chains: %v", err)
	}

	if len(chains) != 1 {
		t.Fatalf("Expected 1 chain, got %d", len(chains))
	}

	if chains[0].IPFSCID != updatedCID {
		t.Fatalf("Expected CID %s in metadata, got %s", updatedCID, chains[0].IPFSCID)
	}

	// Clean up
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// testQueryOptions tests query options (filters, sorting, limit, offset)
func testQueryOptions(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create multiple test chains
	chainTypes := []string{"query-test-1", "query-test-2"}
	repoIDs := []string{"repo-query-1", "repo-query-2"}

	// Save chains with different block counts
	for i, chainType := range chainTypes {
		for j, repoID := range repoIDs {
			blocks := createTestBlocks(chainType, repoID, (i+1)*(j+1))
			err := s.SaveChain(ctx, chainType, repoID, blocks)
			if err != nil {
				t.Fatalf("Failed to save chain %s/%s: %v", chainType, repoID, err)
			}
		}
	}

	// Test filters
	t.Run("Filters", func(t *testing.T) {
		// Filter by chain type
		chains, err := s.ListChains(ctx, &storage.QueryOptions{
			Filters: []storage.Filter{
				{
					Field:    "chain_type",
					Operator: "eq",
					Value:    "query-test-2",
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to list chains with filter: %v", err)
		}

		if len(chains) != 2 {
			t.Fatalf("Expected 2 chains, got %d", len(chains))
		}

		for _, chain := range chains {
			if chain.ChainType != "query-test-2" {
				t.Fatalf("Expected chain type query-test-2, got %s", chain.ChainType)
			}
		}

		// Filter by repo ID
		chains, err = s.ListChains(ctx, &storage.QueryOptions{
			Filters: []storage.Filter{
				{
					Field:    "repo_id",
					Operator: "eq",
					Value:    "repo-query-2",
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to list chains with filter: %v", err)
		}

		if len(chains) != 2 {
			t.Fatalf("Expected 2 chains, got %d", len(chains))
		}

		for _, chain := range chains {
			if chain.RepoID != "repo-query-2" {
				t.Fatalf("Expected repo ID repo-query-2, got %s", chain.RepoID)
			}
		}

		// Filter by block count
		chains, err = s.ListChains(ctx, &storage.QueryOptions{
			Filters: []storage.Filter{
				{
					Field:    "block_count",
					Operator: "gt",
					Value:    float64(1),
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to list chains with filter: %v", err)
		}

		for _, chain := range chains {
			if chain.BlockCount <= 1 {
				t.Fatalf("Expected block count > 1, got %d", chain.BlockCount)
			}
		}

		// Multiple filters
		chains, err = s.ListChains(ctx, &storage.QueryOptions{
			Filters: []storage.Filter{
				{
					Field:    "chain_type",
					Operator: "eq",
					Value:    "query-test-2",
				},
				{
					Field:    "block_count",
					Operator: "gt",
					Value:    float64(1),
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to list chains with multiple filters: %v", err)
		}

		for _, chain := range chains {
			if chain.ChainType != "query-test-2" || chain.BlockCount <= 1 {
				t.Fatalf("Expected chain type query-test-2 and block count > 1, got %s and %d", chain.ChainType, chain.BlockCount)
			}
		}
	})

	// Test sorting
	t.Run("Sorting", func(t *testing.T) {
		// Sort by block count ascending
		chains, err := s.ListChains(ctx, &storage.QueryOptions{
			SortBy:    "block_count",
			SortOrder: "asc",
		})
		if err != nil {
			t.Fatalf("Failed to list chains with sorting: %v", err)
		}

		for i := 1; i < len(chains); i++ {
			if chains[i].BlockCount < chains[i-1].BlockCount {
				t.Fatalf("Expected ascending order, got %d before %d", chains[i-1].BlockCount, chains[i].BlockCount)
			}
		}

		// Sort by block count descending
		chains, err = s.ListChains(ctx, &storage.QueryOptions{
			SortBy:    "block_count",
			SortOrder: "desc",
		})
		if err != nil {
			t.Fatalf("Failed to list chains with sorting: %v", err)
		}

		for i := 1; i < len(chains); i++ {
			if chains[i].BlockCount > chains[i-1].BlockCount {
				t.Fatalf("Expected descending order, got %d before %d", chains[i-1].BlockCount, chains[i].BlockCount)
			}
		}

		// Sort by chain type
		chains, err = s.ListChains(ctx, &storage.QueryOptions{
			SortBy:    "chain_type",
			SortOrder: "asc",
		})
		if err != nil {
			t.Fatalf("Failed to list chains with sorting: %v", err)
		}

		for i := 1; i < len(chains); i++ {
			if chains[i].ChainType < chains[i-1].ChainType {
				t.Fatalf("Expected ascending order, got %s before %s", chains[i-1].ChainType, chains[i].ChainType)
			}
		}
	})

	// Test limit and offset
	t.Run("LimitOffset", func(t *testing.T) {
		// Get total count
		allChains, err := s.ListChains(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list all chains: %v", err)
		}
		totalCount := len(allChains)

		// Test limit
		limit := 2
		chains, err := s.ListChains(ctx, &storage.QueryOptions{
			Limit: limit,
		})
		if err != nil {
			t.Fatalf("Failed to list chains with limit: %v", err)
		}

		if len(chains) != limit {
			t.Fatalf("Expected %d chains, got %d", limit, len(chains))
		}

		// Test offset
		offset := 1
		chains, err = s.ListChains(ctx, &storage.QueryOptions{
			Offset: offset,
		})
		if err != nil {
			t.Fatalf("Failed to list chains with offset: %v", err)
		}

		if len(chains) != totalCount-offset {
			t.Fatalf("Expected %d chains, got %d", totalCount-offset, len(chains))
		}

		// Test limit and offset together
		chains, err = s.ListChains(ctx, &storage.QueryOptions{
			Limit:  1,
			Offset: 1,
		})
		if err != nil {
			t.Fatalf("Failed to list chains with limit and offset: %v", err)
		}

		if len(chains) != 1 {
			t.Fatalf("Expected 1 chain, got %d", len(chains))
		}
	})

	// Test combined options
	t.Run("CombinedOptions", func(t *testing.T) {
		chains, err := s.ListChains(ctx, &storage.QueryOptions{
			Filters: []storage.Filter{
				{
					Field:    "chain_type",
					Operator: "eq",
					Value:    "query-test-2",
				},
			},
			SortBy:    "block_count",
			SortOrder: "desc",
			Limit:     1,
		})
		if err != nil {
			t.Fatalf("Failed to list chains with combined options: %v", err)
		}

		if len(chains) != 1 {
			t.Fatalf("Expected 1 chain, got %d", len(chains))
		}

		for _, chain := range chains {
			if chain.ChainType != "query-test-2" {
				t.Fatalf("Expected chain type query-test-2, got %s", chain.ChainType)
			}
		}
	})

	// Clean up
	for _, chainType := range chainTypes {
		for _, repoID := range repoIDs {
			err := s.DeleteChain(ctx, chainType, repoID)
			if err != nil {
				t.Fatalf("Failed to delete chain %s/%s: %v", chainType, repoID, err)
			}
		}
	}
}

// testErrorHandling tests error handling
func testErrorHandling(t *testing.T, ctx context.Context, s storage.Storage) {
	// Test loading non-existent chain
	_, err := s.LoadChain(ctx, "non-existent", "non-existent")
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}

	// Test loading non-existent block
	_, err = s.LoadBlock(ctx, "non-existent", "non-existent", "non-existent")
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}

	// Test deleting non-existent chain
	err = s.DeleteChain(ctx, "non-existent", "non-existent")
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}

	// Test deleting non-existent block
	err = s.DeleteBlock(ctx, "non-existent", "non-existent", "non-existent")
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}

	// Test duplicate block
	chainType := "error-test"
	repoID := "repo-error"
	b := createTestBlock(chainType, repoID, 0, "")

	// Save block
	err = s.SaveBlock(ctx, chainType, repoID, b)
	if err != nil {
		t.Fatalf("Failed to save block: %v", err)
	}

	// Try to save the same block again
	err = s.SaveBlock(ctx, chainType, repoID, b)
	if err != storage.ErrAlreadyExists {
		t.Fatalf("Expected ErrAlreadyExists, got %v", err)
	}

	// Clean up
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// testConcurrentOperations tests concurrent operations
func testConcurrentOperations(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create test chain
	chainType := "concurrent-test"
	repoID := "repo-concurrent"
	blocks := createTestBlocks(chainType, repoID, 2)

	// Save chain
	err := s.SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		t.Fatalf("Failed to save chain: %v", err)
	}

	// Test concurrent reads
	t.Run("ConcurrentReads", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := s.LoadChain(ctx, chainType, repoID)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Fatalf("Concurrent read failed: %v", err)
		}
	})

	// Test concurrent writes
	t.Run("ConcurrentWrites", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 3)

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				newBlock := createTestBlock(chainType, repoID, uint64(2+index), blocks[1].Hash)
				err := s.SaveBlock(ctx, chainType, repoID, newBlock)
				if err != nil && err != storage.ErrAlreadyExists {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Fatalf("Unexpected error in concurrent write: %v", err)
		}
	})

	// Clean up
	err = s.DeleteChain(ctx, chainType, repoID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
}

// testLinkedJSONSupport tests linked-JSON support
func testLinkedJSONSupport(t *testing.T, ctx context.Context, s storage.Storage) {
	// Create test chain with linked JSON data
	chainType := "linked-json-test"
	repoID := "repo-linked-json"

	// Create blocks with linked JSON data
	blocks := make([]*block.Block, 2)
	prevHash := ""

	for i := 0; i < 2; i++ {
		// Create linked JSON data
		linkedData := map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "Person",
			"name":     fmt.Sprintf("Person %d", i),
			"knows": map[string]interface{}{
				"@type": "Person",
				"name":  fmt.Sprintf("Friend %d", i),
				"url":   fmt.Sprintf("https://example.com/person/%d", i),
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
			ID:           fmt.Sprintf("chain://%s/%s/block%d", chainType, repoID, i),
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
		if data["name"] != fmt.Sprintf("Person %d", i) {
			t.Fatalf("Expected name Person %d, got %v", i, data["name"])
		}

		// Check knows
		knows, ok := data["knows"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected knows to be a map, got %T", data["knows"])
		}

		if knows["@type"] != "Person" {
			t.Fatalf("Expected knows @type Person, got %v", knows["@type"])
		}

		if knows["name"] != fmt.Sprintf("Friend %d", i) {
			t.Fatalf("Expected knows name Friend %d, got %v", i, knows["name"])
		}

		// Check previous person reference
		if i > 0 {
			prevPerson, ok := data["previousPerson"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected previousPerson to be a map, got %T", data["previousPerson"])
			}

			if prevPerson["@id"] != blocks[i-1].ID {
				t.Fatalf("Expected previousPerson @id %s, got %v", blocks[i-1].ID, prevPerson["@id"])
			}
		}
	}

	// Validate blockchain
	validator := validation.NewBlockchainValidator()
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
