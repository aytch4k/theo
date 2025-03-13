package action

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/pkg/types"
)

// ActionChain implements the Chain interface for specific Git actions
type ActionChain struct {
	ChainType string         `json:"chain_type"` // Type of action (commit, merge, etc.)
	RepoID    string         `json:"repo_id"`    // Repository identifier
	Blocks    []*block.Block `json:"blocks"`     // Chain blocks
	mutex     sync.RWMutex   // For thread safety
}

// NewActionChain creates a new action chain
func NewActionChain(chainType, repoID string) *ActionChain {
	return &ActionChain{
		ChainType: chainType,
		RepoID:    repoID,
		Blocks:    make([]*block.Block, 0),
	}
}

// AddBlock adds a new block to the chain
func (c *ActionChain) AddBlock(data []byte) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Get the index and previous hash
	index := uint64(len(c.Blocks))
	var previousHash string
	if index > 0 {
		previousHash = c.Blocks[index-1].Hash
	}

	// Create a new block
	newBlock, err := block.NewBlock(index, previousHash, json.RawMessage(data), c.ChainType, c.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to create block: %w", err)
	}

	// Add the block to the chain
	c.Blocks = append(c.Blocks, newBlock)

	return newBlock.Hash, nil
}

// GetBlock retrieves a block by its hash
func (c *ActionChain) GetBlock(hash string) (*block.Block, error) {
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
func (c *ActionChain) GetBlockByIndex(index uint64) (*block.Block, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if index >= uint64(len(c.Blocks)) {
		return nil, fmt.Errorf("block with index %d not found", index)
	}

	return c.Blocks[index], nil
}

// GetLatestBlock retrieves the latest block in the chain
func (c *ActionChain) GetLatestBlock() (*block.Block, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.Blocks) == 0 {
		return nil, fmt.Errorf("chain is empty")
	}

	return c.Blocks[len(c.Blocks)-1], nil
}

// VerifyChain verifies the integrity of the entire chain
func (c *ActionChain) VerifyChain() bool {
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
func (c *ActionChain) Export() ([]byte, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return json.Marshal(c)
}

// Import imports the chain from a byte array
func (c *ActionChain) Import(data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return json.Unmarshal(data, c)
}

// ExportToJSONL exports the chain to JSONL format
func (c *ActionChain) ExportToJSONL() (string, error) {
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
func (c *ActionChain) ImportFromJSONL(jsonl string) error {
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

// Ensure ActionChain implements the Chain interface
var _ types.Chain = (*ActionChain)(nil)
