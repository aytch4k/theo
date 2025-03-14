package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

func main() {
	ctx := context.Background()
	
	// Create test data for each storage backend
	createTestData(ctx, &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/file-storage-verify",
	}, "file")
	
	createTestData(ctx, &storage.StorageConfig{
		Type:             storage.StorageTypeSQLite,
		ConnectionString: "test-data/sqlite-storage-verify.db",
	}, "sqlite")
	
	createTestData(ctx, &storage.StorageConfig{
		Type:               storage.StorageTypeAerospike,
		AerospikeHost:      "localhost",
		AerospikePort:      3000,
		AerospikeNamespace: "theo",
		AerospikeSet:       "verify-test",
	}, "aerospike")
	
	createTestData(ctx, &storage.StorageConfig{
		Type:         storage.StorageTypeIPFS,
		IPFSHost:     "localhost",
		IPFSPort:     5001,
		IPFSProtocol: "http",
		Path:         "test-data/ipfs-cache-verify",
	}, "ipfs")
}

func createTestData(ctx context.Context, config *storage.StorageConfig, name string) {
	fmt.Printf("Creating test data for %s storage...\n", name)
	
	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		fmt.Printf("Failed to create %s storage: %v\n", name, err)
		return
	}
	
	// Open storage
	err = s.Open(ctx)
	if err != nil {
		fmt.Printf("Failed to open %s storage: %v\n", name, err)
		return
	}
	defer s.Close()
	
	// Create test chain
	chainType := "verify-test"
	repoID := fmt.Sprintf("repo-%s", name)
	
	// Create blocks
	blocks := make([]*block.Block, 0, 5)
	prevHash := ""
	
	for i := 0; i < 5; i++ {
		b := &block.Block{
			Context:      "https://devhub-git.org/contexts/block.jsonld",
			ID:           fmt.Sprintf("chain://%s/%s/block%d", chainType, repoID, i),
			Index:        uint64(i),
			Timestamp:    time.Now().Unix(),
			Data:         []byte(fmt.Sprintf("Test data for block %d in %s storage", i, name)),
			PreviousHash: prevHash,
		}
		
		// Calculate hash
		hash, err := b.CalculateHash()
		if err != nil {
			fmt.Printf("Failed to calculate hash: %v\n", err)
			return
		}
		b.Hash = hash
		prevHash = hash
		
		blocks = append(blocks, b)
	}
	
	// Save chain
	err = s.SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		fmt.Printf("Failed to save chain to %s storage: %v\n", name, err)
		return
	}
	
	// Set IPFS CID
	cid := fmt.Sprintf("QmTest%s123456789", name)
	err = s.SetIPFSCID(ctx, chainType, repoID, cid)
	if err != nil {
		fmt.Printf("Failed to set IPFS CID in %s storage: %v\n", name, err)
		return
	}
	
	fmt.Printf("Successfully created test data for %s storage\n", name)
}
