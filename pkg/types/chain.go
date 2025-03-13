package types

import (
	"github.com/gold2th/theo/internal/blockchain/block"
)

// Chain defines the interface for all blockchain types
type Chain interface {
	// AddBlock adds a new block to the chain
	AddBlock(data []byte) (string, error)

	// GetBlock retrieves a block by its hash
	GetBlock(hash string) (*block.Block, error)

	// GetBlockByIndex retrieves a block by its index
	GetBlockByIndex(index uint64) (*block.Block, error)

	// GetLatestBlock retrieves the latest block in the chain
	GetLatestBlock() (*block.Block, error)

	// VerifyChain verifies the integrity of the entire chain
	VerifyChain() bool

	// Export exports the chain to a byte array
	Export() ([]byte, error)

	// Import imports the chain from a byte array
	Import(data []byte) error
}
