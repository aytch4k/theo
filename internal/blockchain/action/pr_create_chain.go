package action

import (
	"encoding/json"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// PRCreateData represents the data structure for pull request creation actions
type PRCreateData struct {
	PRNumber     string `json:"pr_number"`
	Title        string `json:"title"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Creator      string `json:"creator"`
	Timestamp    int64  `json:"timestamp"`
	Description  string `json:"description"`
}

// PRCreateChain represents a blockchain for tracking pull request creation actions
type PRCreateChain struct {
	ActionChain
}

// NewPRCreateChain creates a new pull request creation chain
func NewPRCreateChain(repoID string) *PRCreateChain {
	return &PRCreateChain{
		ActionChain: *NewActionChain("pr-create", repoID),
	}
}

// AddPRCreate adds a new pull request creation action to the chain
func (c *PRCreateChain) AddPRCreate(prNumber, title, sourceBranch, targetBranch, creator, description string) (string, error) {
	// Create pull request creation data
	data := PRCreateData{
		PRNumber:     prNumber,
		Title:        title,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Creator:      creator,
		Timestamp:    time.Now().Unix(),
		Description:  description,
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Add block to chain
	return c.AddBlock(dataJSON)
}

// GetPRCreations returns all pull request creation actions in the chain
func (c *PRCreateChain) GetPRCreations() ([]PRCreateData, error) {
	var creations []PRCreateData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data PRCreateData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		creations = append(creations, data)
	}

	return creations, nil
}

// GetPRCreationByNumber returns a specific pull request creation action by PR number
func (c *PRCreateChain) GetPRCreationByNumber(prNumber string) (*PRCreateData, error) {
	// Find block by PR number
	for _, b := range c.Blocks {
		var data PRCreateData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.PRNumber == prNumber {
			return &data, nil
		}
	}

	return nil, ErrPRNotFound
}

// GetPRCreationByIndex returns a specific pull request creation action by index
func (c *PRCreateChain) GetPRCreationByIndex(index uint64) (*PRCreateData, error) {
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
	var data PRCreateData
	if err := json.Unmarshal(targetBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetLatestPRCreation returns the most recent pull request creation action
func (c *PRCreateChain) GetLatestPRCreation() (*PRCreateData, error) {
	if len(c.Blocks) == 0 {
		return nil, ErrEmptyChain
	}

	// Get the latest block
	latestBlock := c.Blocks[len(c.Blocks)-1]

	// Unmarshal data
	var data PRCreateData
	if err := json.Unmarshal(latestBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
