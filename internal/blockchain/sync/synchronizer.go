package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/blockchain/validation"
	"github.com/gold2th/theo/internal/storage"
)

// Common synchronization errors
var (
	ErrInvalidChain      = errors.New("invalid chain")
	ErrSyncFailed        = errors.New("synchronization failed")
	ErrNoCommonAncestor  = errors.New("no common ancestor found")
	ErrInvalidMergePoint = errors.New("invalid merge point")
)

// SyncStrategy defines the strategy for resolving conflicts
type SyncStrategy string

// Sync strategies
const (
	// KeepLongest keeps the longest chain
	KeepLongest SyncStrategy = "longest"
	// KeepNewest keeps the chain with the newest blocks
	KeepNewest SyncStrategy = "newest"
	// KeepLocal keeps the local chain
	KeepLocal SyncStrategy = "local"
	// KeepRemote keeps the remote chain
	KeepRemote SyncStrategy = "remote"
	// MergeChains merges the chains
	MergeChains SyncStrategy = "merge"
)

// ChainSynchronizer synchronizes blockchains
type ChainSynchronizer struct {
	storage   storage.Storage
	validator *validation.BlockchainValidator
	strategy  SyncStrategy
}

// Storage returns the storage backend
func (s *ChainSynchronizer) Storage() storage.Storage {
	return s.storage
}

// NewChainSynchronizer creates a new chain synchronizer
func NewChainSynchronizer(s storage.Storage, options ...func(*ChainSynchronizer)) *ChainSynchronizer {
	sync := &ChainSynchronizer{
		storage:   s,
		validator: validation.NewBlockchainValidator(),
		strategy:  KeepLongest,
	}

	// Apply options
	for _, option := range options {
		option(sync)
	}

	return sync
}

// WithValidator sets the blockchain validator
func WithValidator(validator *validation.BlockchainValidator) func(*ChainSynchronizer) {
	return func(s *ChainSynchronizer) {
		s.validator = validator
	}
}

// WithStrategy sets the sync strategy
func WithStrategy(strategy SyncStrategy) func(*ChainSynchronizer) {
	return func(s *ChainSynchronizer) {
		s.strategy = strategy
	}
}

// SyncChain synchronizes a chain with a remote chain
func (s *ChainSynchronizer) SyncChain(ctx context.Context, chainType, repoID string, remoteBlocks []*block.Block) error {
	// Validate remote chain
	if err := s.validator.ValidateChain(remoteBlocks); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidChain, err)
	}

	// Load local chain
	localBlocks, err := s.storage.LoadChain(ctx, chainType, repoID)
	if err != nil && err != storage.ErrNotFound {
		return fmt.Errorf("failed to load local chain: %w", err)
	}

	// If local chain doesn't exist, just save the remote chain
	if err == storage.ErrNotFound || len(localBlocks) == 0 {
		return s.storage.SaveChain(ctx, chainType, repoID, remoteBlocks)
	}

	// Validate local chain
	if err := s.validator.ValidateChain(localBlocks); err != nil {
		return fmt.Errorf("invalid local chain: %w", err)
	}

	// Determine which chain to keep based on strategy
	var blocksToSave []*block.Block
	switch s.strategy {
	case KeepLongest:
		if len(remoteBlocks) > len(localBlocks) {
			blocksToSave = remoteBlocks
		} else {
			blocksToSave = localBlocks
		}
	case KeepNewest:
		if remoteBlocks[len(remoteBlocks)-1].Timestamp > localBlocks[len(localBlocks)-1].Timestamp {
			blocksToSave = remoteBlocks
		} else {
			blocksToSave = localBlocks
		}
	case KeepLocal:
		blocksToSave = localBlocks
	case KeepRemote:
		blocksToSave = remoteBlocks
	case MergeChains:
		mergedBlocks, err := s.mergeChains(localBlocks, remoteBlocks)
		if err != nil {
			return fmt.Errorf("failed to merge chains: %w", err)
		}
		blocksToSave = mergedBlocks
	default:
		return fmt.Errorf("unknown sync strategy: %s", s.strategy)
	}

	// Save the chain
	return s.storage.SaveChain(ctx, chainType, repoID, blocksToSave)
}

// SyncBlock synchronizes a single block with the chain
func (s *ChainSynchronizer) SyncBlock(ctx context.Context, chainType, repoID string, newBlock *block.Block) error {
	// Load local chain
	localBlocks, err := s.storage.LoadChain(ctx, chainType, repoID)
	if err != nil && err != storage.ErrNotFound {
		return fmt.Errorf("failed to load local chain: %w", err)
	}

	// If local chain doesn't exist, create it with the new block
	if err == storage.ErrNotFound || len(localBlocks) == 0 {
		// Validate block as genesis block
		if err := s.validator.ValidateBlock(nil, newBlock); err != nil {
			return fmt.Errorf("invalid genesis block: %w", err)
		}

		return s.storage.SaveBlock(ctx, chainType, repoID, newBlock)
	}

	// Validate local chain
	if err := s.validator.ValidateChain(localBlocks); err != nil {
		return fmt.Errorf("invalid local chain: %w", err)
	}

	// Get the last block in the chain
	lastBlock := localBlocks[len(localBlocks)-1]

	// Validate the new block against the last block
	if err := s.validator.ValidateBlock(lastBlock, newBlock); err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}

	// Save the new block
	return s.storage.SaveBlock(ctx, chainType, repoID, newBlock)
}

// ExportChain exports a chain for synchronization
func (s *ChainSynchronizer) ExportChain(ctx context.Context, chainType, repoID string) ([]*block.Block, error) {
	// Load chain
	blocks, err := s.storage.LoadChain(ctx, chainType, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to load chain: %w", err)
	}

	// Validate chain
	if err := s.validator.ValidateChain(blocks); err != nil {
		return nil, fmt.Errorf("invalid chain: %w", err)
	}

	return blocks, nil
}

// mergeChains merges two chains
func (s *ChainSynchronizer) mergeChains(localBlocks, remoteBlocks []*block.Block) ([]*block.Block, error) {
	// Find common ancestor
	ancestorIndex := -1
	for i := 0; i < len(localBlocks) && i < len(remoteBlocks); i++ {
		if localBlocks[i].Hash == remoteBlocks[i].Hash {
			ancestorIndex = i
		} else {
			break
		}
	}

	// If no common ancestor, return error
	if ancestorIndex == -1 {
		return nil, ErrNoCommonAncestor
	}

	// Create merged chain
	mergedBlocks := make([]*block.Block, ancestorIndex+1)
	copy(mergedBlocks, localBlocks[:ancestorIndex+1])

	// Add blocks from both chains after the common ancestor
	localBranch := localBlocks[ancestorIndex+1:]
	remoteBranch := remoteBlocks[ancestorIndex+1:]

	// Sort blocks by timestamp
	allBlocks := append(localBranch, remoteBranch...)
	sortBlocksByTimestamp(allBlocks)

	// Add blocks to merged chain, ensuring valid chain
	for _, b := range allBlocks {
		// Skip blocks that are already in the chain
		alreadyExists := false
		for _, mb := range mergedBlocks {
			if mb.Hash == b.Hash {
				alreadyExists = true
				break
			}
		}
		if alreadyExists {
			continue
		}

		// Get the last block in the merged chain
		lastBlock := mergedBlocks[len(mergedBlocks)-1]

		// Create a new block with correct index and previous hash
		newBlock := &block.Block{
			Context:      b.Context,
			ID:           b.ID,
			Index:        lastBlock.Index + 1,
			Timestamp:    b.Timestamp,
			Data:         b.Data,
			PreviousHash: lastBlock.Hash,
		}

		// Calculate hash
		hash, err := newBlock.CalculateHash()
		if err != nil {
			return nil, fmt.Errorf("failed to calculate hash: %w", err)
		}
		newBlock.Hash = hash

		// Add to merged chain
		mergedBlocks = append(mergedBlocks, newBlock)
	}

	// Validate merged chain
	if err := s.validator.ValidateChain(mergedBlocks); err != nil {
		return nil, fmt.Errorf("invalid merged chain: %w", err)
	}

	return mergedBlocks, nil
}

// sortBlocksByTimestamp sorts blocks by timestamp
func sortBlocksByTimestamp(blocks []*block.Block) {
	for i := 0; i < len(blocks); i++ {
		for j := i + 1; j < len(blocks); j++ {
			if blocks[i].Timestamp > blocks[j].Timestamp {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			}
		}
	}
}

// SyncWithIPFS synchronizes a chain with IPFS
func (s *ChainSynchronizer) SyncWithIPFS(ctx context.Context, chainType, repoID string) error {
	// Get IPFS CID
	cid, err := s.storage.GetIPFSCID(ctx, chainType, repoID)
	if err != nil {
		return fmt.Errorf("failed to get IPFS CID: %w", err)
	}

	// Export chain from IPFS and import it
	exportedData, err := s.storage.ExportChain(ctx, chainType, repoID, "json")
	if err != nil {
		return fmt.Errorf("failed to export chain: %w", err)
	}

	// Import the chain data
	if err := s.storage.ImportChain(ctx, chainType, repoID, exportedData, "json"); err != nil {
		return fmt.Errorf("failed to import chain: %w", err)
	}

	// Update IPFS CID
	if err := s.storage.SetIPFSCID(ctx, chainType, repoID, cid); err != nil {
		return fmt.Errorf("failed to set IPFS CID: %w", err)
	}

	return nil
}

// PeriodicSync periodically synchronizes chains
func (s *ChainSynchronizer) PeriodicSync(ctx context.Context, chainType, repoID string, interval time.Duration, syncFunc func(context.Context, string, string) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := syncFunc(ctx, chainType, repoID); err != nil {
				// Log error but continue
				fmt.Printf("Sync error: %v\n", err)
			}
		}
	}
}
