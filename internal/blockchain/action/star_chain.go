package action

import (
	"encoding/json"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// StarData represents the data structure for repository star actions
type StarData struct {
	User      string `json:"user"`
	Timestamp int64  `json:"timestamp"`
}

// StarChain represents a blockchain for tracking repository star actions
type StarChain struct {
	ActionChain
}

// NewStarChain creates a new repository star chain
func NewStarChain(repoID string) *StarChain {
	return &StarChain{
		ActionChain: *NewActionChain("star", repoID),
	}
}

// AddStar adds a new repository star action to the chain
func (c *StarChain) AddStar(user string) (string, error) {
	// Create star data
	data := StarData{
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

// GetStars returns all repository star actions in the chain
func (c *StarChain) GetStars() ([]StarData, error) {
	var stars []StarData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data StarData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		stars = append(stars, data)
	}

	return stars, nil
}

// GetStarByIndex returns a specific repository star action by index
func (c *StarChain) GetStarByIndex(index uint64) (*StarData, error) {
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
	var data StarData
	if err := json.Unmarshal(targetBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetStarsByUser returns all repository star actions by a specific user
func (c *StarChain) GetStarsByUser(user string) ([]StarData, error) {
	var stars []StarData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data StarData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.User == user {
			stars = append(stars, data)
		}
	}

	return stars, nil
}

// GetLatestStar returns the most recent repository star action
func (c *StarChain) GetLatestStar() (*StarData, error) {
	if len(c.Blocks) == 0 {
		return nil, ErrEmptyChain
	}

	// Get the latest block
	latestBlock := c.Blocks[len(c.Blocks)-1]

	// Unmarshal data
	var data StarData
	if err := json.Unmarshal(latestBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetStarCount returns the total number of stars
func (c *StarChain) GetStarCount() int {
	return len(c.Blocks)
}

// HasUserStarred checks if a specific user has starred the repository
func (c *StarChain) HasUserStarred(user string) bool {
	// Iterate through blocks
	for _, b := range c.Blocks {
		var data StarData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			continue
		}
		if data.User == user {
			return true
		}
	}

	return false
}
