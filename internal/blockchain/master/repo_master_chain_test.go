package master

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepoMasterChain(t *testing.T) {
	// Create a new repo master chain
	repoID := "repo123"
	ownerID := "owner456"
	chain := NewRepoMasterChain(repoID, ownerID)

	// Verify the chain properties
	assert.Equal(t, repoID, chain.RepoID)
	assert.Equal(t, ownerID, chain.OwnerID)
	assert.Empty(t, chain.Blocks)
}

func TestAddActionHash(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Create user data
	userData := UserData{
		ID: "user789",
	}

	// Add an action hash
	actionType := "commit"
	actionHash := "hash123"
	hash, err := chain.AddActionHash(actionType, actionHash, userData)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, chain.Blocks, 1)

	// Verify the block content
	block := chain.Blocks[0]
	assert.Equal(t, uint64(0), block.Index)
	assert.Equal(t, "", block.PreviousHash)
	assert.Equal(t, hash, block.Hash)

	// Parse the block data
	var blockData struct {
		ActionRef ActionReference `json:"action_ref"`
		User      UserData        `json:"user"`
		Timestamp int64           `json:"timestamp"`
	}
	err = json.Unmarshal(block.Data, &blockData)
	assert.NoError(t, err)
	assert.Equal(t, actionType, blockData.ActionRef.ChainType)
	assert.Equal(t, actionHash, blockData.ActionRef.BlockHash)
	assert.Equal(t, userData.ID, blockData.User.ID)
	assert.NotZero(t, blockData.Timestamp)

	// Add another action hash
	actionType2 := "merge"
	actionHash2 := "hash456"
	hash2, err := chain.AddActionHash(actionType2, actionHash2, userData)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash2)
	assert.Len(t, chain.Blocks, 2)

	// Verify the second block
	block2 := chain.Blocks[1]
	assert.Equal(t, uint64(1), block2.Index)
	assert.Equal(t, hash, block2.PreviousHash)
	assert.Equal(t, hash2, block2.Hash)

	// Parse the second block data
	var blockData2 struct {
		ActionRef ActionReference `json:"action_ref"`
		User      UserData        `json:"user"`
		Timestamp int64           `json:"timestamp"`
	}
	err = json.Unmarshal(block2.Data, &blockData2)
	assert.NoError(t, err)
	assert.Equal(t, actionType2, blockData2.ActionRef.ChainType)
	assert.Equal(t, actionHash2, blockData2.ActionRef.BlockHash)
	assert.Equal(t, userData.ID, blockData2.User.ID)
	assert.NotZero(t, blockData2.Timestamp)
}

func TestAddBlock(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Add a block
	data := []byte(`{"test": "data"}`)
	hash, err := chain.AddBlock(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, chain.Blocks, 1)
	assert.Equal(t, uint64(0), chain.Blocks[0].Index)
	assert.Equal(t, "", chain.Blocks[0].PreviousHash)
	assert.Equal(t, hash, chain.Blocks[0].Hash)

	// Add another block
	data2 := []byte(`{"test": "data2"}`)
	hash2, err := chain.AddBlock(data2)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash2)
	assert.Len(t, chain.Blocks, 2)
	assert.Equal(t, uint64(1), chain.Blocks[1].Index)
	assert.Equal(t, hash, chain.Blocks[1].PreviousHash)
	assert.Equal(t, hash2, chain.Blocks[1].Hash)
}

func TestGetBlock(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Add a block
	data := []byte(`{"test": "data"}`)
	hash, err := chain.AddBlock(data)
	assert.NoError(t, err)

	// Get the block by hash
	block, err := chain.GetBlock(hash)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), block.Index)
	assert.Equal(t, "", block.PreviousHash)
	assert.Equal(t, hash, block.Hash)

	// Test getting a non-existent block
	_, err = chain.GetBlock("non-existent-hash")
	assert.Error(t, err)
}

func TestGetBlockByIndex(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Add a block
	data := []byte(`{"test": "data"}`)
	hash, err := chain.AddBlock(data)
	assert.NoError(t, err)

	// Get the block by index
	block, err := chain.GetBlockByIndex(0)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), block.Index)
	assert.Equal(t, "", block.PreviousHash)
	assert.Equal(t, hash, block.Hash)

	// Test getting a non-existent block
	_, err = chain.GetBlockByIndex(1)
	assert.Error(t, err)
}

func TestGetLatestBlock(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Test getting the latest block from an empty chain
	_, err := chain.GetLatestBlock()
	assert.Error(t, err)

	// Add a block
	data := []byte(`{"test": "data"}`)
	hash, err := chain.AddBlock(data)
	assert.NoError(t, err)

	// Get the latest block
	block, err := chain.GetLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), block.Index)
	assert.Equal(t, "", block.PreviousHash)
	assert.Equal(t, hash, block.Hash)

	// Add another block
	data2 := []byte(`{"test": "data2"}`)
	hash2, err := chain.AddBlock(data2)
	assert.NoError(t, err)

	// Get the latest block again
	block, err = chain.GetLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), block.Index)
	assert.Equal(t, hash, block.PreviousHash)
	assert.Equal(t, hash2, block.Hash)
}

func TestVerifyChain(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Verify an empty chain
	assert.True(t, chain.VerifyChain())

	// Add a block
	data := []byte(`{"test": "data"}`)
	_, err := chain.AddBlock(data)
	assert.NoError(t, err)

	// Verify the chain with one block
	assert.True(t, chain.VerifyChain())

	// Add another block
	data2 := []byte(`{"test": "data2"}`)
	_, err = chain.AddBlock(data2)
	assert.NoError(t, err)

	// Verify the chain with two blocks
	assert.True(t, chain.VerifyChain())

	// Tamper with the chain
	chain.Blocks[0].Data = json.RawMessage(`{"test": "tampered"}`)

	// Verify the tampered chain
	assert.False(t, chain.VerifyChain())
}

func TestExportImport(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Add some blocks
	data1 := []byte(`{"test": "data1"}`)
	_, err := chain.AddBlock(data1)
	assert.NoError(t, err)

	data2 := []byte(`{"test": "data2"}`)
	_, err = chain.AddBlock(data2)
	assert.NoError(t, err)

	// Export the chain
	exportedData, err := chain.Export()
	assert.NoError(t, err)
	assert.NotEmpty(t, exportedData)

	// Create a new chain and import the data
	importedChain := NewRepoMasterChain("", "")
	err = importedChain.Import(exportedData)
	assert.NoError(t, err)

	// Verify the imported chain
	assert.Equal(t, chain.RepoID, importedChain.RepoID)
	assert.Equal(t, chain.OwnerID, importedChain.OwnerID)
	assert.Len(t, importedChain.Blocks, 2)
	assert.Equal(t, chain.Blocks[0].Hash, importedChain.Blocks[0].Hash)
	assert.Equal(t, chain.Blocks[1].Hash, importedChain.Blocks[1].Hash)
}

func TestJSONLExportImport(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Add some blocks
	data1 := []byte(`{"test": "data1"}`)
	_, err := chain.AddBlock(data1)
	assert.NoError(t, err)

	data2 := []byte(`{"test": "data2"}`)
	_, err = chain.AddBlock(data2)
	assert.NoError(t, err)

	// Export the chain to JSONL
	jsonl, err := chain.ExportToJSONL()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonl)

	// Create a new chain and import the JSONL
	importedChain := NewRepoMasterChain("repo123", "owner456")
	err = importedChain.ImportFromJSONL(jsonl)
	assert.NoError(t, err)

	// Verify the imported chain
	assert.Len(t, importedChain.Blocks, 2)

	// Compare the blocks
	for i := 0; i < len(chain.Blocks); i++ {
		assert.Equal(t, chain.Blocks[i].Index, importedChain.Blocks[i].Index)
		assert.Equal(t, chain.Blocks[i].PreviousHash, importedChain.Blocks[i].PreviousHash)
		assert.Equal(t, chain.Blocks[i].Hash, importedChain.Blocks[i].Hash)

		// Parse the JSON data to compare the actual content
		var originalData, importedData map[string]interface{}
		err = json.Unmarshal(chain.Blocks[i].Data, &originalData)
		assert.NoError(t, err)
		err = json.Unmarshal(importedChain.Blocks[i].Data, &importedData)
		assert.NoError(t, err)
		assert.Equal(t, originalData, importedData)
	}
}

func TestSyncToLayer1(t *testing.T) {
	// Create a new repo master chain
	chain := NewRepoMasterChain("repo123", "owner456")

	// Test syncing to Layer 1 (should return "not implemented" error)
	err := chain.SyncToLayer1()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
