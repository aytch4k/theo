package action

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewActionChain(t *testing.T) {
	// Create a new action chain
	chainType := "test-chain"
	repoID := "test-repo"
	chain := NewActionChain(chainType, repoID)

	// Verify chain properties
	if chain.ChainType != chainType {
		t.Errorf("Expected chain type %s, got %s", chainType, chain.ChainType)
	}
	if chain.RepoID != repoID {
		t.Errorf("Expected repo ID %s, got %s", repoID, chain.RepoID)
	}
	if len(chain.Blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(chain.Blocks))
	}
}

func TestActionChain_AddBlock(t *testing.T) {
	// Create a new action chain
	chain := NewActionChain("test-chain", "test-repo")

	// Create test data
	testData := struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Message string `json:"message"`
	}{
		ID:      "test-id",
		Name:    "test-name",
		Message: "test-message",
	}
	testDataJSON, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Add block to chain
	hash, err := chain.AddBlock(testDataJSON)
	if err != nil {
		t.Fatalf("Failed to add block to chain: %v", err)
	}

	// Verify block was added
	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}

	// Verify block properties
	block := chain.Blocks[0]
	if block.Index != 0 {
		t.Errorf("Expected block index 0, got %d", block.Index)
	}
	if block.PreviousHash != "" {
		t.Errorf("Expected empty previous hash, got %s", block.PreviousHash)
	}
	if block.Hash != hash {
		t.Errorf("Expected hash %s, got %s", hash, block.Hash)
	}
	if string(block.Data) != string(testDataJSON) {
		t.Errorf("Expected data %s, got %s", string(testDataJSON), string(block.Data))
	}
}

func TestActionChain_AddMultipleBlocks(t *testing.T) {
	// Create a new action chain
	chain := NewActionChain("test-chain", "test-repo")

	// Add multiple blocks
	numBlocks := 5
	for i := 0; i < numBlocks; i++ {
		// Create test data
		testData := struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Message string `json:"message"`
			Index   int    `json:"index"`
		}{
			ID:      "test-id",
			Name:    "test-name",
			Message: "test-message",
			Index:   i,
		}
		testDataJSON, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("Failed to marshal test data: %v", err)
		}

		// Add block to chain
		_, err = chain.AddBlock(testDataJSON)
		if err != nil {
			t.Fatalf("Failed to add block to chain: %v", err)
		}
	}

	// Verify number of blocks
	if len(chain.Blocks) != numBlocks {
		t.Errorf("Expected %d blocks, got %d", numBlocks, len(chain.Blocks))
	}

	// Verify block indices and previous hashes
	for i := 0; i < numBlocks; i++ {
		block := chain.Blocks[i]
		if block.Index != uint64(i) {
			t.Errorf("Expected block index %d, got %d", i, block.Index)
		}
		if i > 0 && block.PreviousHash != chain.Blocks[i-1].Hash {
			t.Errorf("Block %d: Expected previous hash %s, got %s", i, chain.Blocks[i-1].Hash, block.PreviousHash)
		}
	}
}

func TestActionChain_VerifyChain(t *testing.T) {
	// Create a new action chain
	chain := NewActionChain("test-chain", "test-repo")

	// Add multiple blocks
	numBlocks := 5
	for i := 0; i < numBlocks; i++ {
		// Create test data
		testData := struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Message string `json:"message"`
			Index   int    `json:"index"`
		}{
			ID:      "test-id",
			Name:    "test-name",
			Message: "test-message",
			Index:   i,
		}
		testDataJSON, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("Failed to marshal test data: %v", err)
		}

		// Add block to chain
		_, err = chain.AddBlock(testDataJSON)
		if err != nil {
			t.Fatalf("Failed to add block to chain: %v", err)
		}
	}

	// Verify chain
	if !chain.VerifyChain() {
		t.Error("Chain verification failed")
	}

	// Tamper with a block
	chain.Blocks[2].Data = []byte(`{"id":"tampered","name":"tampered","message":"tampered","index":2}`)

	// Verify chain again
	if chain.VerifyChain() {
		t.Error("Chain verification should have failed after tampering")
	}
}

func TestActionChain_Export(t *testing.T) {
	// Create a new action chain
	chain := NewActionChain("test-chain", "test-repo")

	// Add a block
	testData := struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Message string `json:"message"`
	}{
		ID:      "test-id",
		Name:    "test-name",
		Message: "test-message",
	}
	testDataJSON, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	_, err = chain.AddBlock(testDataJSON)
	if err != nil {
		t.Fatalf("Failed to add block to chain: %v", err)
	}

	// Export chain
	exportData, err := chain.Export()
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Import chain
	importedChain := NewActionChain("", "")
	err = importedChain.Import(exportData)
	if err != nil {
		t.Fatalf("Failed to import chain: %v", err)
	}

	// Verify imported chain
	if importedChain.ChainType != chain.ChainType {
		t.Errorf("Expected chain type %s, got %s", chain.ChainType, importedChain.ChainType)
	}
	if importedChain.RepoID != chain.RepoID {
		t.Errorf("Expected repo ID %s, got %s", chain.RepoID, importedChain.RepoID)
	}
	if len(importedChain.Blocks) != len(chain.Blocks) {
		t.Errorf("Expected %d blocks, got %d", len(chain.Blocks), len(importedChain.Blocks))
	}
	if !importedChain.VerifyChain() {
		t.Error("Imported chain verification failed")
	}
}

func TestActionChain_ExportToJSONL(t *testing.T) {
	// Create a new action chain
	chain := NewActionChain("test-chain", "test-repo")

	// Add multiple blocks
	numBlocks := 5
	for i := 0; i < numBlocks; i++ {
		// Create test data
		testData := struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Message string `json:"message"`
			Index   int    `json:"index"`
		}{
			ID:      "test-id",
			Name:    "test-name",
			Message: "test-message",
			Index:   i,
		}
		testDataJSON, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("Failed to marshal test data: %v", err)
		}

		// Add block to chain
		_, err = chain.AddBlock(testDataJSON)
		if err != nil {
			t.Fatalf("Failed to add block to chain: %v", err)
		}
	}

	// Export chain to JSONL
	jsonlStr, err := chain.ExportToJSONL()
	if err != nil {
		t.Fatalf("Failed to export chain to JSONL: %v", err)
	}

	// Verify JSONL format
	lines := strings.Split(jsonlStr, "\n")
	lineCount := 0

	for i, line := range lines {
		if line == "" {
			continue
		}

		// Each line should be a valid JSON object
		var blockData map[string]interface{}
		err = json.Unmarshal([]byte(line), &blockData)
		if err != nil {
			t.Fatalf("Failed to unmarshal JSONL line %d: %v", i, err)
		}

		// Verify block data
		if index, ok := blockData["index"].(float64); ok {
			if uint64(index) != uint64(lineCount) {
				t.Errorf("Expected block index %d, got %f", lineCount, index)
			}
		}

		lineCount++
	}

	// Verify number of lines
	if lineCount != numBlocks {
		t.Errorf("Expected %d lines, got %d", numBlocks, lineCount)
	}
}

func TestActionChain_BlockTimestamps(t *testing.T) {
	// Create a new action chain
	chain := NewActionChain("test-chain", "test-repo")

	// Add a block
	testData := struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Message string `json:"message"`
	}{
		ID:      "test-id",
		Name:    "test-name",
		Message: "test-message",
	}
	testDataJSON, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Record time before adding block
	beforeTime := time.Now().Unix()

	// Add block to chain
	_, err = chain.AddBlock(testDataJSON)
	if err != nil {
		t.Fatalf("Failed to add block to chain: %v", err)
	}

	// Record time after adding block
	afterTime := time.Now().Unix()

	// Verify block timestamp
	blockTime := chain.Blocks[0].Timestamp
	if blockTime < beforeTime || blockTime > afterTime {
		t.Errorf("Block timestamp %d is outside the expected range [%d, %d]", blockTime, beforeTime, afterTime)
	}
}
