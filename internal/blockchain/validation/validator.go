package validation

import (
	"errors"
	"fmt"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// Common validation errors
var (
	ErrEmptyChain        = errors.New("chain is empty")
	ErrInvalidHash       = errors.New("invalid block hash")
	ErrInvalidPrevHash   = errors.New("invalid previous hash")
	ErrInvalidIndex      = errors.New("invalid block index")
	ErrInvalidBlockOrder = errors.New("invalid block order")
	ErrInvalidTimestamp  = errors.New("invalid block timestamp")
)

// BlockchainValidator validates blockchain integrity
type BlockchainValidator struct {
	// Configuration options
	StrictTimestampOrder bool
	AllowEmptyPrevHash   bool
}

// NewBlockchainValidator creates a new blockchain validator
func NewBlockchainValidator(options ...func(*BlockchainValidator)) *BlockchainValidator {
	v := &BlockchainValidator{
		StrictTimestampOrder: true,
		AllowEmptyPrevHash:   true,
	}

	// Apply options
	for _, option := range options {
		option(v)
	}

	return v
}

// WithStrictTimestampOrder sets the strict timestamp order option
func WithStrictTimestampOrder(strict bool) func(*BlockchainValidator) {
	return func(v *BlockchainValidator) {
		v.StrictTimestampOrder = strict
	}
}

// WithAllowEmptyPrevHash sets the allow empty previous hash option
func WithAllowEmptyPrevHash(allow bool) func(*BlockchainValidator) {
	return func(v *BlockchainValidator) {
		v.AllowEmptyPrevHash = allow
	}
}

// ValidateChain validates a blockchain
func (v *BlockchainValidator) ValidateChain(blocks []*block.Block) error {
	if len(blocks) == 0 {
		return ErrEmptyChain
	}

	// Validate each block
	for i, b := range blocks {
		// Validate block hash
		calculatedHash, err := b.CalculateHash()
		if err != nil {
			return fmt.Errorf("failed to calculate hash for block %d: %w", i, err)
		}

		if calculatedHash != b.Hash {
			return fmt.Errorf("%w for block %d: expected %s, got %s", ErrInvalidHash, i, calculatedHash, b.Hash)
		}

		// Validate block index
		if b.Index != uint64(i) {
			return fmt.Errorf("%w for block %d: expected index %d, got %d", ErrInvalidIndex, i, i, b.Index)
		}

		// Validate previous hash (except for genesis block)
		if i > 0 {
			if b.PreviousHash != blocks[i-1].Hash {
				return fmt.Errorf("%w for block %d: expected %s, got %s", ErrInvalidPrevHash, i, blocks[i-1].Hash, b.PreviousHash)
			}
		} else if !v.AllowEmptyPrevHash && b.PreviousHash != "" {
			return fmt.Errorf("%w for genesis block: expected empty, got %s", ErrInvalidPrevHash, b.PreviousHash)
		}

		// Validate timestamp order
		if v.StrictTimestampOrder && i > 0 {
			if b.Timestamp < blocks[i-1].Timestamp {
				return fmt.Errorf("%w for block %d: timestamp %d is before previous block timestamp %d", ErrInvalidTimestamp, i, b.Timestamp, blocks[i-1].Timestamp)
			}
		}
	}

	return nil
}

// ValidateBlock validates a single block against a previous block
func (v *BlockchainValidator) ValidateBlock(prevBlock, newBlock *block.Block) error {
	// Validate block hash
	calculatedHash, err := newBlock.CalculateHash()
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	if calculatedHash != newBlock.Hash {
		return fmt.Errorf("%w: expected %s, got %s", ErrInvalidHash, calculatedHash, newBlock.Hash)
	}

	// Validate block index
	if prevBlock != nil {
		if newBlock.Index != prevBlock.Index+1 {
			return fmt.Errorf("%w: expected index %d, got %d", ErrInvalidIndex, prevBlock.Index+1, newBlock.Index)
		}

		// Validate previous hash
		if newBlock.PreviousHash != prevBlock.Hash {
			return fmt.Errorf("%w: expected %s, got %s", ErrInvalidPrevHash, prevBlock.Hash, newBlock.PreviousHash)
		}

		// Validate timestamp order
		if v.StrictTimestampOrder && newBlock.Timestamp < prevBlock.Timestamp {
			return fmt.Errorf("%w: timestamp %d is before previous block timestamp %d", ErrInvalidTimestamp, newBlock.Timestamp, prevBlock.Timestamp)
		}
	} else if !v.AllowEmptyPrevHash && newBlock.PreviousHash != "" {
		return fmt.Errorf("%w for genesis block: expected empty, got %s", ErrInvalidPrevHash, newBlock.PreviousHash)
	}

	return nil
}
