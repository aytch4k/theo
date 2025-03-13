package action

import (
	"encoding/json"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// PRMergeData represents the data structure for pull request merge actions
type PRMergeData struct {
	PRNumber        string `json:"pr_number"`
	MergeCommitHash string `json:"merge_commit_hash"`
	Merger          string `json:"merger"`
	Timestamp       int64  `json:"timestamp"`
	MergeMethod     string `json:"merge_method"` // "merge", "squash", "rebase"
	MergeMessage    string `json:"merge_message"`
}

// PRMergeChain represents a blockchain for tracking pull request merge actions
type PRMergeChain struct {
	ActionChain
}

// NewPRMergeChain creates a new pull request merge chain
func NewPRMergeChain(repoID string) *PRMergeChain {
	return &PRMergeChain{
		ActionChain: *NewActionChain("pr-merge", repoID),
	}
}

// AddPRMerge adds a new pull request merge action to the chain
func (c *PRMergeChain) AddPRMerge(prNumber, mergeCommitHash, merger, mergeMethod, mergeMessage string) (string, error) {
	// Create pull request merge data
	data := PRMergeData{
		PRNumber:        prNumber,
		MergeCommitHash: mergeCommitHash,
		Merger:          merger,
		Timestamp:       time.Now().Unix(),
		MergeMethod:     mergeMethod,
		MergeMessage:    mergeMessage,
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Add block to chain
	return c.AddBlock(dataJSON)
}

// GetPRMerges returns all pull request merge actions in the chain
func (c *PRMergeChain) GetPRMerges() ([]PRMergeData, error) {
	var merges []PRMergeData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data PRMergeData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		merges = append(merges, data)
	}

	return merges, nil
}

// GetPRMergeByNumber returns a specific pull request merge action by PR number
func (c *PRMergeChain) GetPRMergeByNumber(prNumber string) (*PRMergeData, error) {
	// Find block by PR number
	for _, b := range c.Blocks {
		var data PRMergeData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.PRNumber == prNumber {
			return &data, nil
		}
	}

	return nil, ErrPRMergeNotFound
}

// GetPRMergeByCommitHash returns a specific pull request merge action by commit hash
func (c *PRMergeChain) GetPRMergeByCommitHash(commitHash string) (*PRMergeData, error) {
	// Find block by commit hash
	for _, b := range c.Blocks {
		var data PRMergeData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.MergeCommitHash == commitHash {
			return &data, nil
		}
	}

	return nil, ErrPRMergeNotFound
}

// GetPRMergeByIndex returns a specific pull request merge action by index
func (c *PRMergeChain) GetPRMergeByIndex(index uint64) (*PRMergeData, error) {
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
	var data PRMergeData
	if err := json.Unmarshal(targetBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetLatestPRMerge returns the most recent pull request merge action
func (c *PRMergeChain) GetLatestPRMerge() (*PRMergeData, error) {
	if len(c.Blocks) == 0 {
		return nil, ErrEmptyChain
	}

	// Get the latest block
	latestBlock := c.Blocks[len(c.Blocks)-1]

	// Unmarshal data
	var data PRMergeData
	if err := json.Unmarshal(latestBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
