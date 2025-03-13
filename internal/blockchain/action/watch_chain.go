package action

import (
	"encoding/json"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// WatchData represents the data structure for repository watch actions
type WatchData struct {
	User      string `json:"user"`
	Timestamp int64  `json:"timestamp"`
}

// WatchChain represents a blockchain for tracking repository watch actions
type WatchChain struct {
	ActionChain
}

// NewWatchChain creates a new repository watch chain
func NewWatchChain(repoID string) *WatchChain {
	return &WatchChain{
		ActionChain: *NewActionChain("watch", repoID),
	}
}

// AddWatch adds a new repository watch action to the chain
func (c *WatchChain) AddWatch(user string) (string, error) {
	// Create watch data
	data := WatchData{
		User:      user,
		Timestamp: time.Now().Unix(),
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Add block to chain
	return c.AddBlock(dataJSON)
}

// GetWatches returns all repository watch actions in the chain
func (c *WatchChain) GetWatches() ([]WatchData, error) {
	var watches []WatchData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data WatchData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		watches = append(watches, data)
	}

	return watches, nil
}

// GetWatchByIndex returns a specific repository watch action by index
func (c *WatchChain) GetWatchByIndex(index uint64) (*WatchData, error) {
	// Find block by index
	var targetBlock *block.Block
	for _, b := range c.Blocks {
		if b.Index == index {
			targetBlock = b
			break
		}
	}

	if targetBlock == nil {
		return nil, ErrBlockNotFound
	}

	// Unmarshal data
	var data WatchData
	if err := json.Unmarshal(targetBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetWatchesByUser returns all repository watch actions by a specific user
func (c *WatchChain) GetWatchesByUser(user string) ([]WatchData, error) {
	var watches []WatchData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data WatchData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.User == user {
			watches = append(watches, data)
		}
	}

	return watches, nil
}

// GetLatestWatch returns the most recent repository watch action
func (c *WatchChain) GetLatestWatch() (*WatchData, error) {
	if len(c.Blocks) == 0 {
		return nil, ErrEmptyChain
	}

	// Get the latest block
	latestBlock := c.Blocks[len(c.Blocks)-1]

	// Unmarshal data
	var data WatchData
	if err := json.Unmarshal(latestBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetWatchCount returns the total number of watches
func (c *WatchChain) GetWatchCount() int {
	return len(c.Blocks)
}

// IsUserWatching checks if a specific user is watching the repository
func (c *WatchChain) IsUserWatching(user string) bool {
	// Iterate through blocks
	for _, b := range c.Blocks {
		var data WatchData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			continue
		}
		if data.User == user {
			return true
		}
	}

	return false
}
