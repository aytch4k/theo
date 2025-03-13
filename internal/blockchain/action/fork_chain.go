package action

import (
	"encoding/json"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// ForkData represents the data structure for repository fork actions
type ForkData struct {
	ParentRepo   string `json:"parent_repo"`
	ForkRepoName string `json:"fork_repo_name"`
	Forker       string `json:"forker"`
	Timestamp    int64  `json:"timestamp"`
}

// ForkChain represents a blockchain for tracking repository fork actions
type ForkChain struct {
	ActionChain
}

// NewForkChain creates a new repository fork chain
func NewForkChain(repoID string) *ForkChain {
	return &ForkChain{
		ActionChain: *NewActionChain("fork", repoID),
	}
}

// AddFork adds a new repository fork action to the chain
func (c *ForkChain) AddFork(parentRepo, forkRepoName, forker string) (string, error) {
	// Create fork data
	data := ForkData{
		ParentRepo:   parentRepo,
		ForkRepoName: forkRepoName,
		Forker:       forker,
		Timestamp:    time.Now().Unix(),
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Add block to chain
	return c.AddBlock(dataJSON)
}

// GetForks returns all repository fork actions in the chain
func (c *ForkChain) GetForks() ([]ForkData, error) {
	var forks []ForkData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data ForkData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		forks = append(forks, data)
	}

	return forks, nil
}

// GetForkByIndex returns a specific repository fork action by index
func (c *ForkChain) GetForkByIndex(index uint64) (*ForkData, error) {
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
	var data ForkData
	if err := json.Unmarshal(targetBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetForksByForker returns all repository fork actions by a specific forker
func (c *ForkChain) GetForksByForker(forker string) ([]ForkData, error) {
	var forks []ForkData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data ForkData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.Forker == forker {
			forks = append(forks, data)
		}
	}

	return forks, nil
}

// GetLatestFork returns the most recent repository fork action
func (c *ForkChain) GetLatestFork() (*ForkData, error) {
	if len(c.Blocks) == 0 {
		return nil, ErrEmptyChain
	}

	// Get the latest block
	latestBlock := c.Blocks[len(c.Blocks)-1]

	// Unmarshal data
	var data ForkData
	if err := json.Unmarshal(latestBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
