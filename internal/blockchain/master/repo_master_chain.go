package master

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/pkg/types"
)

// UserData represents user metadata
type UserData struct {
	ID        string `json:"id"`
	PublicKey []byte `json:"public_key,omitempty"`
	Signature []byte `json:"signature,omitempty"`
}

// ActionReference represents a reference to an action chain block
type ActionReference struct {
	ChainType  string `json:"chain_type"`
	BlockHash  string `json:"block_hash"`
	BlockIndex uint64 `json:"block_index"`
}

// RepoMasterChain implements the Chain interface for repository master
type RepoMasterChain struct {
	RepoID  string         `json:"repo_id"`  // Repository identifier
	OwnerID string         `json:"owner_id"` // Repository owner
	Blocks  []*block.Block `json:"blocks"`   // Chain blocks
	mutex   sync.RWMutex   // For thread safety
}

// NewRepoMasterChain creates a new repository master chain
func NewRepoMasterChain(repoID, ownerID string) *RepoMasterChain {
	return &RepoMasterChain{
		RepoID:  repoID,
		OwnerID: ownerID,
		Blocks:  make([]*block.Block, 0),
	}
}

// AddActionHash adds an action hash to the chain
func (c *RepoMasterChain) AddActionHash(actionType, actionHash string, userData UserData) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Create the action reference
	actionRef := ActionReference{
		ChainType: actionType,
		BlockHash: actionHash,
	}

	// Create the block data
	blockData := struct {
		ActionRef ActionReference `json:"action_ref"`
		User      UserData        `json:"user"`
		Timestamp int64           `json:"timestamp"`
	}{
		ActionRef: actionRef,
		User:      userData,
		Timestamp: time.Now().Unix(),
	}

	// Convert to JSON
	dataJSON, err := json.Marshal(blockData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal block data: %w", err)
	}

	// Get the index and previous hash
	index := uint64(len(c.Blocks))
	var previousHash string
	if index > 0 {
		previousHash = c.Blocks[index-1].Hash
	}

	// Create a new block
	newBlock, err := block.NewBlock(index, previousHash, json.RawMessage(dataJSON), "master", c.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to create block: %w", err)
	}

	// Add the block to the chain
	c.Blocks = append(c.Blocks, newBlock)

	return newBlock.Hash, nil
}

// AddBlock adds a new block to the chain
// This is part of the Chain interface, but for RepoMasterChain, we prefer to use AddActionHash
func (c *RepoMasterChain) AddBlock(data []byte) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Get the index and previous hash
	index := uint64(len(c.Blocks))
	var previousHash string
	if index > 0 {
		previousHash = c.Blocks[index-1].Hash
	}

	// Create a new block
	newBlock, err := block.NewBlock(index, previousHash, json.RawMessage(data), "master", c.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to create block: %w", err)
	}

	// Add the block to the chain
	c.Blocks = append(c.Blocks, newBlock)

	return newBlock.Hash, nil
}

// GetBlock retrieves a block by its hash
func (c *RepoMasterChain) GetBlock(hash string) (*block.Block, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, block := range c.Blocks {
		if block.Hash == hash {
			return block, nil
		}
	}

	return nil, fmt.Errorf("block with hash %s not found", hash)
}

// GetBlockByIndex retrieves a block by its index
func (c *RepoMasterChain) GetBlockByIndex(index uint64) (*block.Block, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if index >= uint64(len(c.Blocks)) {
		return nil, fmt.Errorf("block with index %d not found", index)
	}

	return c.Blocks[index], nil
}

// GetLatestBlock retrieves the latest block in the chain
func (c *RepoMasterChain) GetLatestBlock() (*block.Block, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.Blocks) == 0 {
		return nil, fmt.Errorf("chain is empty")
	}

	return c.Blocks[len(c.Blocks)-1], nil
}

// VerifyChain verifies the integrity of the entire chain
func (c *RepoMasterChain) VerifyChain() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// If the chain is empty, it's valid
	if len(c.Blocks) == 0 {
		return true
	}

	// Verify each block, including the genesis block
	for i := 0; i < len(c.Blocks); i++ {
		// Verify that the hash is correct for each block
		valid, err := c.Blocks[i].Verify()
		if err != nil || !valid {
			return false
		}

		// For non-genesis blocks, verify the previous hash
		if i > 0 {
			// Verify that the previous hash matches the hash of the previous block
			if c.Blocks[i].PreviousHash != c.Blocks[i-1].Hash {
				return false
			}

			// Verify that the index is one more than the previous block's index
			if c.Blocks[i].Index != c.Blocks[i-1].Index+1 {
				return false
			}
		}
	}

	return true
}

// Export exports the chain to a byte array
func (c *RepoMasterChain) Export() ([]byte, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return json.Marshal(c)
}

// Import imports the chain from a byte array
func (c *RepoMasterChain) Import(data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return json.Unmarshal(data, c)
}

// ExportToJSONL exports the chain to JSONL format
func (c *RepoMasterChain) ExportToJSONL() (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var result string
	for _, block := range c.Blocks {
		jsonl, err := block.ToJSONL()
		if err != nil {
			return "", fmt.Errorf("failed to convert block to JSONL: %w", err)
		}
		result += jsonl + "\n"
	}

	return result, nil
}

// ImportFromJSONL imports the chain from JSONL format
func (c *RepoMasterChain) ImportFromJSONL(jsonl string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Clear the existing blocks
	c.Blocks = make([]*block.Block, 0)

	// Split the JSONL by newlines
	lines := splitLines(jsonl)
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse the block
		block, err := block.FromJSONL(line)
		if err != nil {
			return fmt.Errorf("failed to parse block from JSONL: %w", err)
		}

		// Add the block to the chain
		c.Blocks = append(c.Blocks, block)
	}

	return nil
}

// SyncToLayer1 syncs the chain to Layer 1
// This is a placeholder for now, will be implemented when we add Layer 1 integration
func (c *RepoMasterChain) SyncToLayer1() error {
	// TODO: Implement Layer 1 sync
	return fmt.Errorf("not implemented")
}

// Helper function to split a string by newlines
func splitLines(s string) []string {
	var lines []string
	var line string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// Ensure RepoMasterChain implements the Chain interface
var _ types.Chain = (*RepoMasterChain)(nil)
