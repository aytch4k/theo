package action

import (
	"encoding/json"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// PRCloseData represents the data structure for pull request close actions
type PRCloseData struct {
	PRNumber    string `json:"pr_number"`
	Closer      string `json:"closer"`
	Timestamp   int64  `json:"timestamp"`
	CloseReason string `json:"close_reason"` // "completed", "abandoned", "superseded"
	Comment     string `json:"comment"`
}

// PRCloseChain represents a blockchain for tracking pull request close actions
type PRCloseChain struct {
	ActionChain
}

// NewPRCloseChain creates a new pull request close chain
func NewPRCloseChain(repoID string) *PRCloseChain {
	return &PRCloseChain{
		ActionChain: *NewActionChain("pr-close", repoID),
	}
}

// AddPRClose adds a new pull request close action to the chain
func (c *PRCloseChain) AddPRClose(prNumber, closer, closeReason, comment string) (string, error) {
	// Create pull request close data
	data := PRCloseData{
		PRNumber:    prNumber,
		Closer:      closer,
		Timestamp:   time.Now().Unix(),
		CloseReason: closeReason,
		Comment:     comment,
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Add block to chain
	return c.AddBlock(dataJSON)
}

// GetPRCloses returns all pull request close actions in the chain
func (c *PRCloseChain) GetPRCloses() ([]PRCloseData, error) {
	var closes []PRCloseData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data PRCloseData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		closes = append(closes, data)
	}

	return closes, nil
}

// GetPRCloseByNumber returns a specific pull request close action by PR number
func (c *PRCloseChain) GetPRCloseByNumber(prNumber string) (*PRCloseData, error) {
	// Find block by PR number
	for _, b := range c.Blocks {
		var data PRCloseData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.PRNumber == prNumber {
			return &data, nil
		}
	}

	return nil, ErrPRCloseNotFound
}

// GetPRCloseByIndex returns a specific pull request close action by index
func (c *PRCloseChain) GetPRCloseByIndex(index uint64) (*PRCloseData, error) {
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
	var data PRCloseData
	if err := json.Unmarshal(targetBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetLatestPRClose returns the most recent pull request close action
func (c *PRCloseChain) GetLatestPRClose() (*PRCloseData, error) {
	if len(c.Blocks) == 0 {
		return nil, ErrEmptyChain
	}

	// Get the latest block
	latestBlock := c.Blocks[len(c.Blocks)-1]

	// Unmarshal data
	var data PRCloseData
	if err := json.Unmarshal(latestBlock.Data, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// GetPRClosesByReason returns all pull request close actions with a specific reason
func (c *PRCloseChain) GetPRClosesByReason(reason string) ([]PRCloseData, error) {
	var closes []PRCloseData

	// Iterate through blocks
	for _, b := range c.Blocks {
		var data PRCloseData
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return nil, err
		}
		if data.CloseReason == reason {
			closes = append(closes, data)
		}
	}

	return closes, nil
}
