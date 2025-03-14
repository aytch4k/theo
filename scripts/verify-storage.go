package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v6"
	_ "github.com/mattn/go-sqlite3"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

// Test data
const (
	chainType = "verify-test"
	repoID    = "verify-repo"
)

func main() {
	// Create test block
	testBlock := createTestBlock(chainType, repoID, 0, "")

	// Verify each storage backend
	fmt.Println("=== Verifying Storage Backends ===")

	// Verify File Storage
	fmt.Println("\n=== File Storage ===")
	verifyFileStorage(testBlock)

	// Verify SQLite Storage
	fmt.Println("\n=== SQLite Storage ===")
	verifySQLiteStorage(testBlock)

	// Verify Aerospike Storage
	fmt.Println("\n=== Aerospike Storage ===")
	verifyAerospikeStorage(testBlock)

	// Note about IndexedDB
	fmt.Println("\n=== IndexedDB Storage ===")
	fmt.Println("IndexedDB is a browser-based storage and cannot be directly verified from a Go script.")
	fmt.Println("To verify IndexedDB, you would need to run the tests in a browser environment.")
}

// createTestBlock creates a test block
func createTestBlock(chainType, repoID string, index uint64, prevHash string) *block.Block {
	// Create test data
	data := map[string]interface{}{
		"message": fmt.Sprintf("Test block %d", index),
		"time":    time.Now().Format(time.RFC3339),
	}

	// Convert data to json.RawMessage
	dataJSON, _ := json.Marshal(data)

	// Create block
	b := &block.Block{
		Context:      "https://devhub-git.org/contexts/block.jsonld",
		ID:           fmt.Sprintf("chain://%s/%s/block%d", chainType, repoID, index),
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

// verifyFileStorage verifies file storage
func verifyFileStorage(testBlock *block.Block) {
	// Create file storage config
	config := &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/file-storage-verify",
	}

	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		fmt.Printf("Failed to create file storage: %v\n", err)
		return
	}

	// Open storage
	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		fmt.Printf("Failed to open file storage: %v\n", err)
		return
	}
	defer s.Close()

	// Save block
	err = s.SaveBlock(ctx, chainType, repoID, testBlock)
	if err != nil {
		fmt.Printf("Failed to save block to file storage: %v\n", err)
		return
	}
	fmt.Println("Successfully saved block to file storage")

	// Load block
	loadedBlock, err := s.LoadBlock(ctx, chainType, repoID, testBlock.Hash)
	if err != nil {
		fmt.Printf("Failed to load block from file storage: %v\n", err)
		return
	}
	fmt.Println("Successfully loaded block from file storage")

	// Verify block
	if loadedBlock.Hash != testBlock.Hash {
		fmt.Printf("Block hash mismatch: expected %s, got %s\n", testBlock.Hash, loadedBlock.Hash)
		return
	}
	fmt.Println("Block hash verified")

	// Verify file exists on disk
	dirPath := filepath.Join("test-data/file-storage-verify", chainType, repoID)
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("Failed to read directory: %v\n", err)
		return
	}

	found := false
	for _, file := range files {
		if file.Name() == testBlock.Hash+".json" {
			found = true
			break
		}
	}

	if !found {
		fmt.Println("Block file not found on disk")
		return
	}
	fmt.Println("Block file verified on disk")

	// Clean up
	err = s.DeleteBlock(ctx, chainType, repoID, testBlock.Hash)
	if err != nil {
		fmt.Printf("Failed to delete block from file storage: %v\n", err)
		return
	}
	fmt.Println("Successfully deleted block from file storage")
}

// verifySQLiteStorage verifies SQLite storage
func verifySQLiteStorage(testBlock *block.Block) {
	// Create SQLite storage config
	dbPath := "test-data/sqlite-storage-verify.db"
	config := &storage.StorageConfig{
		Type:             storage.StorageTypeSQLite,
		ConnectionString: dbPath,
	}

	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		fmt.Printf("Failed to create SQLite storage: %v\n", err)
		return
	}

	// Open storage
	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		fmt.Printf("Failed to open SQLite storage: %v\n", err)
		return
	}
	defer s.Close()

	// Save block
	err = s.SaveBlock(ctx, chainType, repoID, testBlock)
	if err != nil {
		fmt.Printf("Failed to save block to SQLite storage: %v\n", err)
		return
	}
	fmt.Println("Successfully saved block to SQLite storage")

	// Load block
	loadedBlock, err := s.LoadBlock(ctx, chainType, repoID, testBlock.Hash)
	if err != nil {
		fmt.Printf("Failed to load block from SQLite storage: %v\n", err)
		return
	}
	fmt.Println("Successfully loaded block from SQLite storage")

	// Verify block
	if loadedBlock.Hash != testBlock.Hash {
		fmt.Printf("Block hash mismatch: expected %s, got %s\n", testBlock.Hash, loadedBlock.Hash)
		return
	}
	fmt.Println("Block hash verified")

	// Verify database directly
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Failed to open SQLite database: %v\n", err)
		return
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM blocks WHERE hash = ?", testBlock.Hash).Scan(&count)
	if err != nil {
		fmt.Printf("Failed to query SQLite database: %v\n", err)
		return
	}

	if count != 1 {
		fmt.Printf("Expected 1 block in database, got %d\n", count)
		return
	}
	fmt.Println("Block verified in SQLite database")

	// Clean up
	err = s.DeleteBlock(ctx, chainType, repoID, testBlock.Hash)
	if err != nil {
		fmt.Printf("Failed to delete block from SQLite storage: %v\n", err)
		return
	}
	fmt.Println("Successfully deleted block from SQLite storage")
}

// verifyAerospikeStorage verifies Aerospike storage
func verifyAerospikeStorage(testBlock *block.Block) {
	// Create Aerospike storage config
	config := &storage.StorageConfig{
		Type:               storage.StorageTypeAerospike,
		AerospikeHost:      "aerospike", // Use Docker service name
		AerospikePort:      3000,
		AerospikeNamespace: "theo",
		AerospikeSet:       "verify-test",
	}

	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		fmt.Printf("Failed to create Aerospike storage: %v\n", err)
		return
	}

	// Open storage
	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		fmt.Printf("Failed to open Aerospike storage: %v\n", err)
		return
	}
	defer s.Close()

	// Save block
	err = s.SaveBlock(ctx, chainType, repoID, testBlock)
	if err != nil {
		fmt.Printf("Failed to save block to Aerospike storage: %v\n", err)
		return
	}
	fmt.Println("Successfully saved block to Aerospike storage")

	// Load block
	loadedBlock, err := s.LoadBlock(ctx, chainType, repoID, testBlock.Hash)
	if err != nil {
		fmt.Printf("Failed to load block from Aerospike storage: %v\n", err)
		return
	}
	fmt.Println("Successfully loaded block from Aerospike storage")

	// Verify block
	if loadedBlock.Hash != testBlock.Hash {
		fmt.Printf("Block hash mismatch: expected %s, got %s\n", testBlock.Hash, loadedBlock.Hash)
		return
	}
	fmt.Println("Block hash verified")

	// Verify Aerospike directly
	client, err := aero.NewClient("aerospike", 3000) // Use Docker service name
	if err != nil {
		fmt.Printf("Failed to connect to Aerospike: %v\n", err)
		return
	}
	defer client.Close()

	// Construct the key
	blockKey := fmt.Sprintf("block:%s:%s:%s", chainType, repoID, testBlock.Hash)
	key, err := aero.NewKey("theo", "verify-test", blockKey+":"+blockKey)
	if err != nil {
		fmt.Printf("Failed to create Aerospike key: %v\n", err)
		return
	}

	// Check if the record exists
	exists, err := client.Exists(nil, key)
	if err != nil {
		fmt.Printf("Failed to check if record exists in Aerospike: %v\n", err)
		return
	}

	if !exists {
		fmt.Println("Block not found in Aerospike")
		return
	}
	fmt.Println("Block verified in Aerospike")

	// Clean up
	err = s.DeleteBlock(ctx, chainType, repoID, testBlock.Hash)
	if err != nil {
		fmt.Printf("Failed to delete block from Aerospike storage: %v\n", err)
		return
	}
	fmt.Println("Successfully deleted block from Aerospike storage")
}
