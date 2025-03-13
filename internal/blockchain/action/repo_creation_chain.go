package action

import (
	"encoding/json"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// RepoCreationData represents the data structure for repository creation actions
type RepoCreationData struct {
	RepoName          string `json:"repo_name"`
	Owner             string `json:"owner"`
	CreationTimestamp int64  `json:"creation_timestamp"`
	Visibility        string `json:"visibility"` // "public" or "private"
}

// RepoCreationChain represents a blockchain for tracking repository creation actions
type RepoCreationChain struct {
	ActionChain
}

// NewRepoCreationChain creates a new repository creation chain
func NewRepoCreationChain(repoID string) *RepoCreationChain {
	return &RepoCreationChain{
		ActionChain: *NewActionChain("repo-creation", repoID),
	}
}

// AddRepoCreation adds a new repository creation action to the chain
func (c *RepoCreationChain) AddRepoCreation(repoName, owner, visibility string) (string, error) {
	// Create repository creation data
	data := RepoCreationData{
		RepoName:          repoName,
		Owner:             owner,
		CreationTimestamp: time.Now().Unix(),
		Visibility:        visibility,
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Add block to chain
	return c.AddBlock(dataJSON)
}

// GetRepoCreations returns all repository creation actions in the chain
func (c *RepoCreationChain) GetRepoCreations() ([]RepoCreationData, error) {
	var creations []RepoCreationData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data RepoCreationData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		creations = append(creations, data)
	}

	return creations, nil
}

// GetRepoCreationByIndex returns a specific repository creation action by index
func (c *RepoCreationChain) GetRepoCreationByIndex(index uint64) (*RepoCreationData, error) {
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
	var data RepoCreationData
	if err := json.Unmarshal(targetBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetLatestRepoCreation returns the most recent repository creation action
func (c *RepoCreationChain) GetLatestRepoCreation() (*RepoCreationData, error) {
	if len(c.Blocks) == 0 {
		return nil, ErrEmptyChain
	}

	// Get the latest block
	latestBlock := c.Blocks[len(c.Blocks)-1]

	// Unmarshal data
	var data RepoCreationData
	if err := json.Unmarshal(latestBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
