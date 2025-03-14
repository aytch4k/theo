package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

func main() {
	ctx := context.Background()
	
	// Create test data for each storage backend
	if len(os.Args) < 2 {
		fmt.Println("Usage: test-data-generator <storage-type>")
		os.Exit(1)
	}
	
	storageType := os.Args[1]
	
	var config *storage.StorageConfig
	
	switch storageType {
	case "file":
		config = &storage.StorageConfig{
			Type: storage.StorageTypeFile,
			Path: "test-data/file-storage-verify",
		}
	case "sqlite":
		config = &storage.StorageConfig{
			Type:             storage.StorageTypeSQLite,
			ConnectionString: "test-data/sqlite-storage-verify.db",
		}
	case "aerospike":
		config = &storage.StorageConfig{
			Type:               storage.StorageTypeAerospike,
			AerospikeHost:      "localhost",
			AerospikePort:      3000,
			AerospikeNamespace: "theo",
			AerospikeSet:       "verify-test",
		}
	case "ipfs":
		config = &storage.StorageConfig{
			Type:         storage.StorageTypeIPFS,
			IPFSHost:     "localhost",
			IPFSPort:     5001,
			IPFSProtocol: "http",
			Path:         "test-data/ipfs-cache-verify",
		}
	default:
		fmt.Printf("Unknown storage type: %s\n", storageType)
		os.Exit(1)
	}
	
	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		fmt.Printf("Failed to create %s storage: %v\n", storageType, err)
		os.Exit(1)
	}
	
	// Open storage
	err = s.Open(ctx)
	if err != nil {
		fmt.Printf("Failed to open %s storage: %v\n", storageType, err)
		os.Exit(1)
	}
	defer s.Close()
	
	// Create test chain
	chainType := "verify-test"
	repoID := fmt.Sprintf("repo-%s", storageType)
	
	// Create blocks
	blocks := make([]*block.Block, 0, 5)
	prevHash := ""
	
	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"message":   fmt.Sprintf("Test data for block %d in %s storage", i, storageType),
			"timestamp": time.Now().Unix(),
			"index":     i,
		}
		
		dataBytes, err := json.Marshal(data)
		if err != nil {
			fmt.Printf("Failed to marshal data: %v\n", err)
			os.Exit(1)
		}
		
		b := &block.Block{
			Context:      "https://devhub-git.org/contexts/block.jsonld",
			ID:           fmt.Sprintf("chain://%s/%s/block%d", chainType, repoID, i),
			Index:        uint64(i),
			Timestamp:    time.Now().Unix(),
			Data:         dataBytes,
			PreviousHash: prevHash,
		}
		
		// Calculate hash
		hash, err := b.CalculateHash()
		if err != nil {
			fmt.Printf("Failed to calculate hash: %v\n", err)
			os.Exit(1)
		}
		b.Hash = hash
		prevHash = hash
		
		blocks = append(blocks, b)
	}
	
	// Save chain
	err = s.SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		fmt.Printf("Failed to save chain to %s storage: %v\n", storageType, err)
		os.Exit(1)
	}
	
	// Set IPFS CID
	cid := fmt.Sprintf("QmTest%s123456789", storageType)
	err = s.SetIPFSCID(ctx, chainType, repoID, cid)
	if err != nil {
		fmt.Printf("Failed to set IPFS CID in %s storage: %v\n", storageType, err)
		os.Exit(1)
	}
	
	// Load chain to verify
	loadedBlocks, err := s.LoadChain(ctx, chainType, repoID)
	if err != nil {
		fmt.Printf("Failed to load chain from %s storage: %v\n", storageType, err)
		os.Exit(1)
	}
	
	// Print loaded blocks
	fmt.Printf("Successfully loaded %d blocks from %s storage\n", len(loadedBlocks), storageType)
	for i, b := range loadedBlocks {
		fmt.Printf("Block %d:\n", i)
		fmt.Printf("  Hash: %s\n", b.Hash)
		fmt.Printf("  Previous Hash: %s\n", b.PreviousHash)
		fmt.Printf("  Index: %d\n", b.Index)
		fmt.Printf("  Timestamp: %d\n", b.Timestamp)
		
		var data map[string]interface{}
		if err := json.Unmarshal(b.Data, &data); err != nil {
			fmt.Printf("  Data: %s\n", string(b.Data))
		} else {
			fmt.Printf("  Data: %v\n", data)
		}
		fmt.Println()
	}
	
	// Get IPFS CID
	loadedCID, err := s.GetIPFSCID(ctx, chainType, repoID)
	if err != nil {
		fmt.Printf("Failed to get IPFS CID from %s storage: %v\n", storageType, err)
		os.Exit(1)
	}
	
	fmt.Printf("IPFS CID: %s\n", loadedCID)
	
	fmt.Printf("Successfully created and verified test data for %s storage\n", storageType)
}
