package block

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBlock(t *testing.T) {
	// Test data
	data := json.RawMessage(`{"test": "data"}`)
	chainType := "commit"
	repoID := "repo123"

	// Create a genesis block
	genesisBlock, err := NewBlock(0, "", data, chainType, repoID)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), genesisBlock.Index)
	assert.Equal(t, "", genesisBlock.PreviousHash)
	assert.Equal(t, data, genesisBlock.Data)
	assert.NotEmpty(t, genesisBlock.Hash)
	assert.Contains(t, genesisBlock.ID, "chain://commit/repo123/block")

	// Create a non-genesis block
	block, err := NewBlock(1, genesisBlock.Hash, data, chainType, repoID)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), block.Index)
	assert.Equal(t, genesisBlock.Hash, block.PreviousHash)
	assert.Equal(t, data, block.Data)
	assert.NotEmpty(t, block.Hash)
	assert.Contains(t, block.ID, "chain://commit/repo123/block")

	// Test error case: non-genesis block without previous hash
	_, err = NewBlock(1, "", data, chainType, repoID)
	assert.Error(t, err)
}

func TestCalculateHash(t *testing.T) {
	// Create a block
	data := json.RawMessage(`{"test": "data"}`)
	block, err := NewBlock(0, "", data, "commit", "repo123")
	assert.NoError(t, err)

	// Calculate the hash
	hash, err := block.CalculateHash()
	assert.NoError(t, err)
	assert.Equal(t, block.Hash, hash)

	// Modify the block and check that the hash changes
	block.Data = json.RawMessage(`{"test": "modified"}`)
	newHash, err := block.CalculateHash()
	assert.NoError(t, err)
	assert.NotEqual(t, hash, newHash)
}

func TestVerify(t *testing.T) {
	// Create a block
	data := json.RawMessage(`{"test": "data"}`)
	block, err := NewBlock(0, "", data, "commit", "repo123")
	assert.NoError(t, err)

	// Verify the block
	valid, err := block.Verify()
	assert.NoError(t, err)
	assert.True(t, valid)

	// Modify the block and check that verification fails
	block.Data = json.RawMessage(`{"test": "modified"}`)
	valid, err = block.Verify()
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestSignAndVerifySignature(t *testing.T) {
	// Mock signing function
	mockSign := func(data, privateKey []byte) ([]byte, error) {
		// In a real implementation, this would use a proper signing algorithm
		// For testing, we'll just concatenate the data and private key
		return append(data, privateKey...), nil
	}

	// Mock verification function
	mockVerify := func(data, signature, publicKey []byte) (bool, error) {
		// In a real implementation, this would use a proper verification algorithm
		// For testing, we'll just check if the signature starts with the data
		return len(signature) > len(data) && string(signature[:len(data)]) == string(data), nil
	}

	// Create a block
	data := json.RawMessage(`{"test": "data"}`)
	block, err := NewBlock(0, "", data, "commit", "repo123")
	assert.NoError(t, err)

	// Sign the block
	privateKey := []byte("private-key")
	err = block.Sign(privateKey, mockSign)
	assert.NoError(t, err)
	assert.NotEmpty(t, block.Signature)

	// Verify the signature
	publicKey := []byte("public-key") // In a real implementation, this would be the corresponding public key
	valid, err := block.VerifySignature(publicKey, mockVerify)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestJSONLConversion(t *testing.T) {
	// Create a block
	data := json.RawMessage(`{"test": "data"}`)
	block, err := NewBlock(0, "", data, "commit", "repo123")
	assert.NoError(t, err)

	// Convert to JSONL
	jsonl, err := block.ToJSONL()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonl)

	// Convert back from JSONL
	parsedBlock, err := FromJSONL(jsonl)
	assert.NoError(t, err)
	assert.Equal(t, block.ID, parsedBlock.ID)
	assert.Equal(t, block.Index, parsedBlock.Index)
	assert.Equal(t, block.PreviousHash, parsedBlock.PreviousHash)
	assert.Equal(t, block.Hash, parsedBlock.Hash)

	// Parse the JSON data to compare the actual content rather than the formatting
	var originalData, parsedData map[string]interface{}
	err = json.Unmarshal(block.Data, &originalData)
	assert.NoError(t, err)
	err = json.Unmarshal(parsedBlock.Data, &parsedData)
	assert.NoError(t, err)
	assert.Equal(t, originalData, parsedData)
}
