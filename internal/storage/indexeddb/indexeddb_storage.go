package indexeddb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
)

// IndexedDBStorage implements the Storage interface using IndexedDB
// This is a mock implementation since IndexedDB is a browser-based storage
// and cannot be directly accessed from Go. In a real implementation, this would
// be a client-side JavaScript implementation that communicates with a server.
type IndexedDBStorage struct {
	config     *storage.StorageConfig
	mutex      sync.RWMutex
	isOpen     bool
	cidMapping map[string]string // Maps chainType+repoID to IPFS CID

	// In-memory storage for mock implementation
	chains map[string]storage.ChainMetadata
	blocks map[string]map[string]*block.Block // chainKey -> blockHash -> block
}

// NewIndexedDBStorage creates a new IndexedDB storage backend
func NewIndexedDBStorage(config *storage.StorageConfig) (*IndexedDBStorage, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.DBName == "" {
		return nil, fmt.Errorf("database name cannot be empty")
	}

	return &IndexedDBStorage{
		config:     config,
		isOpen:     false,
		cidMapping: make(map[string]string),
		chains:     make(map[string]storage.ChainMetadata),
		blocks:     make(map[string]map[string]*block.Block),
	}, nil
}

// getChainKey returns a unique key for a chain
func getChainKey(chainType, repoID string) string {
	return chainType + ":" + repoID
}

// Open initializes the IndexedDB storage backend
func (is *IndexedDBStorage) Open(ctx context.Context) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if is.isOpen {
		return nil
	}

	// In a real implementation, this would initialize the IndexedDB database
	// and create the necessary object stores

	is.isOpen = true
	return nil
}

// Close closes the IndexedDB storage backend
func (is *IndexedDBStorage) Close() error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return nil
	}

	// In a real implementation, this would close the IndexedDB database

	is.isOpen = false
	return nil
}

// SaveChain saves a chain to storage
func (is *IndexedDBStorage) SaveChain(ctx context.Context, chainType, repoID string, blocks []*block.Block) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)

	// Create or update chain metadata
	is.chains[chainKey] = storage.ChainMetadata{
		ChainType:   chainType,
		RepoID:      repoID,
		BlockCount:  len(blocks),
		LastUpdated: time.Now(),
		IPFSCID:     is.cidMapping[chainKey],
	}

	// Create blocks map if it doesn't exist
	if _, ok := is.blocks[chainKey]; !ok {
		is.blocks[chainKey] = make(map[string]*block.Block)
	} else {
		// Clear existing blocks
		is.blocks[chainKey] = make(map[string]*block.Block)
	}

	// Save blocks
	for _, b := range blocks {
		is.blocks[chainKey][b.Hash] = b
	}

	return nil
}

// LoadChain loads a chain from storage
func (is *IndexedDBStorage) LoadChain(ctx context.Context, chainType, repoID string) ([]*block.Block, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return nil, storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)

	// Check if chain exists
	if _, ok := is.chains[chainKey]; !ok {
		return nil, storage.ErrNotFound
	}

	// Get blocks
	blockMap, ok := is.blocks[chainKey]
	if !ok {
		return []*block.Block{}, nil
	}

	// Convert map to slice
	var result []*block.Block
	for _, b := range blockMap {
		result = append(result, b)
	}

	// Sort blocks by index
	sortBlocksByIndex(result)

	return result, nil
}

// sortBlocksByIndex sorts blocks by index
func sortBlocksByIndex(blocks []*block.Block) {
	for i := 0; i < len(blocks); i++ {
		for j := i + 1; j < len(blocks); j++ {
			if blocks[i].Index > blocks[j].Index {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			}
		}
	}
}

// DeleteChain deletes a chain from storage
func (is *IndexedDBStorage) DeleteChain(ctx context.Context, chainType, repoID string) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)

	// Check if chain exists
	if _, ok := is.chains[chainKey]; !ok {
		return storage.ErrNotFound
	}

	// Delete chain
	delete(is.chains, chainKey)
	delete(is.blocks, chainKey)
	delete(is.cidMapping, chainKey)

	return nil
}

// ListChains lists all chains in storage
func (is *IndexedDBStorage) ListChains(ctx context.Context, options *storage.QueryOptions) ([]storage.ChainMetadata, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Convert map to slice
	var result []storage.ChainMetadata
	for _, metadata := range is.chains {
		result = append(result, metadata)
	}

	// Apply filters if options are provided
	if options != nil && len(options.Filters) > 0 {
		filteredResult := make([]storage.ChainMetadata, 0)
		for _, metadata := range result {
			if applyFilters(metadata, options.Filters) {
				filteredResult = append(filteredResult, metadata)
			}
		}
		result = filteredResult
	}

	// Apply sorting if options are provided
	if options != nil && options.SortBy != "" {
		sortChainMetadata(result, options.SortBy, options.SortOrder)
	}

	// Apply limit and offset if options are provided
	if options != nil {
		// Apply offset
		if options.Offset > 0 && options.Offset < len(result) {
			result = result[options.Offset:]
		}

		// Apply limit
		if options.Limit > 0 && options.Limit < len(result) {
			result = result[:options.Limit]
		}
	}

	return result, nil
}

// sortChainMetadata sorts chain metadata by the specified field
func sortChainMetadata(metadata []storage.ChainMetadata, sortBy, sortOrder string) {
	for i := 0; i < len(metadata); i++ {
		for j := i + 1; j < len(metadata); j++ {
			shouldSwap := false

			switch sortBy {
			case "chain_type":
				if sortOrder == "desc" {
					shouldSwap = metadata[i].ChainType < metadata[j].ChainType
				} else {
					shouldSwap = metadata[i].ChainType > metadata[j].ChainType
				}
			case "repo_id":
				if sortOrder == "desc" {
					shouldSwap = metadata[i].RepoID < metadata[j].RepoID
				} else {
					shouldSwap = metadata[i].RepoID > metadata[j].RepoID
				}
			case "block_count":
				if sortOrder == "desc" {
					shouldSwap = metadata[i].BlockCount < metadata[j].BlockCount
				} else {
					shouldSwap = metadata[i].BlockCount > metadata[j].BlockCount
				}
			case "last_updated":
				if sortOrder == "desc" {
					shouldSwap = metadata[i].LastUpdated.Before(metadata[j].LastUpdated)
				} else {
					shouldSwap = metadata[i].LastUpdated.After(metadata[j].LastUpdated)
				}
			case "ipfs_cid":
				if sortOrder == "desc" {
					shouldSwap = metadata[i].IPFSCID < metadata[j].IPFSCID
				} else {
					shouldSwap = metadata[i].IPFSCID > metadata[j].IPFSCID
				}
			}

			if shouldSwap {
				metadata[i], metadata[j] = metadata[j], metadata[i]
			}
		}
	}
}

// applyFilters applies filters to a chain metadata
func applyFilters(metadata storage.ChainMetadata, filters []storage.Filter) bool {
	for _, filter := range filters {
		switch filter.Field {
		case "chain_type":
			if !applyStringFilter(metadata.ChainType, filter) {
				return false
			}
		case "repo_id":
			if !applyStringFilter(metadata.RepoID, filter) {
				return false
			}
		case "block_count":
			if !applyIntFilter(metadata.BlockCount, filter) {
				return false
			}
		case "last_updated":
			if !applyTimeFilter(metadata.LastUpdated, filter) {
				return false
			}
		case "ipfs_cid":
			if !applyStringFilter(metadata.IPFSCID, filter) {
				return false
			}
		}
	}
	return true
}

// applyStringFilter applies a filter to a string value
func applyStringFilter(value string, filter storage.Filter) bool {
	filterValue, ok := filter.Value.(string)
	if !ok {
		return false
	}

	switch filter.Operator {
	case "eq":
		return value == filterValue
	case "neq":
		return value != filterValue
	case "contains":
		return strings.Contains(value, filterValue)
	case "startswith":
		return strings.HasPrefix(value, filterValue)
	case "endswith":
		return strings.HasSuffix(value, filterValue)
	default:
		return false
	}
}

// applyIntFilter applies a filter to an int value
func applyIntFilter(value int, filter storage.Filter) bool {
	filterValue, ok := filter.Value.(float64)
	if !ok {
		return false
	}

	intFilterValue := int(filterValue)

	switch filter.Operator {
	case "eq":
		return value == intFilterValue
	case "neq":
		return value != intFilterValue
	case "gt":
		return value > intFilterValue
	case "gte":
		return value >= intFilterValue
	case "lt":
		return value < intFilterValue
	case "lte":
		return value <= intFilterValue
	default:
		return false
	}
}

// applyTimeFilter applies a filter to a time value
func applyTimeFilter(value time.Time, filter storage.Filter) bool {
	filterValue, ok := filter.Value.(string)
	if !ok {
		return false
	}

	filterTime, err := time.Parse(time.RFC3339, filterValue)
	if err != nil {
		return false
	}

	switch filter.Operator {
	case "eq":
		return value.Equal(filterTime)
	case "neq":
		return !value.Equal(filterTime)
	case "gt":
		return value.After(filterTime)
	case "gte":
		return value.After(filterTime) || value.Equal(filterTime)
	case "lt":
		return value.Before(filterTime)
	case "lte":
		return value.Before(filterTime) || value.Equal(filterTime)
	default:
		return false
	}
}

// SaveBlock saves a block to a chain
func (is *IndexedDBStorage) SaveBlock(ctx context.Context, chainType, repoID string, b *block.Block) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)

	// Check if chain exists
	metadata, ok := is.chains[chainKey]
	if !ok {
		// Create chain
		metadata = storage.ChainMetadata{
			ChainType:   chainType,
			RepoID:      repoID,
			BlockCount:  0,
			LastUpdated: time.Now(),
			IPFSCID:     is.cidMapping[chainKey],
		}
	}

	// Create blocks map if it doesn't exist
	if _, ok := is.blocks[chainKey]; !ok {
		is.blocks[chainKey] = make(map[string]*block.Block)
	}

	// Check if block already exists
	if _, ok := is.blocks[chainKey][b.Hash]; ok {
		return storage.ErrAlreadyExists
	}

	// Save block
	is.blocks[chainKey][b.Hash] = b

	// Update chain metadata
	metadata.BlockCount++
	metadata.LastUpdated = time.Now()
	is.chains[chainKey] = metadata

	return nil
}

// LoadBlock loads a block from a chain
func (is *IndexedDBStorage) LoadBlock(ctx context.Context, chainType, repoID string, blockHash string) (*block.Block, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return nil, storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)

	// Check if chain exists
	if _, ok := is.chains[chainKey]; !ok {
		return nil, storage.ErrNotFound
	}

	// Check if blocks map exists
	blockMap, ok := is.blocks[chainKey]
	if !ok {
		return nil, storage.ErrNotFound
	}

	// Get block
	block, ok := blockMap[blockHash]
	if !ok {
		return nil, storage.ErrNotFound
	}

	return block, nil
}

// LoadBlockByIndex loads a block from a chain by its index
func (is *IndexedDBStorage) LoadBlockByIndex(ctx context.Context, chainType, repoID string, blockIndex uint64) (*block.Block, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return nil, storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)

	// Check if chain exists
	if _, ok := is.chains[chainKey]; !ok {
		return nil, storage.ErrNotFound
	}

	// Check if blocks map exists
	blockMap, ok := is.blocks[chainKey]
	if !ok {
		return nil, storage.ErrNotFound
	}

	// Find block by index
	for _, block := range blockMap {
		if block.Index == blockIndex {
			return block, nil
		}
	}

	return nil, storage.ErrNotFound
}

// DeleteBlock deletes a block from a chain
func (is *IndexedDBStorage) DeleteBlock(ctx context.Context, chainType, repoID string, blockHash string) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)

	// Check if chain exists
	metadata, ok := is.chains[chainKey]
	if !ok {
		return storage.ErrNotFound
	}

	// Check if blocks map exists
	blockMap, ok := is.blocks[chainKey]
	if !ok {
		return storage.ErrNotFound
	}

	// Check if block exists
	if _, ok := blockMap[blockHash]; !ok {
		return storage.ErrNotFound
	}

	// Delete block
	delete(blockMap, blockHash)

	// Update chain metadata
	metadata.BlockCount--
	metadata.LastUpdated = time.Now()
	is.chains[chainKey] = metadata

	return nil
}

// ExportChain exports a chain to a byte array
func (is *IndexedDBStorage) ExportChain(ctx context.Context, chainType, repoID string, format string) ([]byte, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Load chain
	blocks, err := is.LoadChain(ctx, chainType, repoID)
	if err != nil {
		return nil, err
	}

	// Export based on format
	switch format {
	case "json":
		// Export as JSON
		type exportedChain struct {
			ChainType string         `json:"chain_type"`
			RepoID    string         `json:"repo_id"`
			Blocks    []*block.Block `json:"blocks"`
			IPFSCID   string         `json:"ipfs_cid,omitempty"`
		}

		chainKey := getChainKey(chainType, repoID)
		chain := exportedChain{
			ChainType: chainType,
			RepoID:    repoID,
			Blocks:    blocks,
			IPFSCID:   is.cidMapping[chainKey],
		}

		return json.Marshal(chain)
	case "jsonl":
		// Export as JSONL
		var result string
		for _, b := range blocks {
			jsonl, err := b.ToJSONL()
			if err != nil {
				return nil, fmt.Errorf("failed to convert block to JSONL: %w", err)
			}
			result += jsonl + "\n"
		}
		return []byte(result), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// ImportChain imports a chain from a byte array
func (is *IndexedDBStorage) ImportChain(ctx context.Context, chainType, repoID string, data []byte, format string) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	var blocks []*block.Block

	// Import based on format
	switch format {
	case "json":
		// Import from JSON
		type importedChain struct {
			ChainType string         `json:"chain_type"`
			RepoID    string         `json:"repo_id"`
			Blocks    []*block.Block `json:"blocks"`
			IPFSCID   string         `json:"ipfs_cid,omitempty"`
		}

		var chain importedChain
		if err := json.Unmarshal(data, &chain); err != nil {
			return fmt.Errorf("failed to unmarshal chain data: %w", err)
		}

		blocks = chain.Blocks

		// Update CID mapping
		chainKey := getChainKey(chainType, repoID)
		if chain.IPFSCID != "" {
			is.cidMapping[chainKey] = chain.IPFSCID
		}
	case "jsonl":
		// Import from JSONL
		lines := strings.Split(string(data), "\n")
		blocks = make([]*block.Block, 0, len(lines))

		for _, line := range lines {
			if line == "" {
				continue
			}

			b, err := block.FromJSONL(line)
			if err != nil {
				return fmt.Errorf("failed to parse block from JSONL: %w", err)
			}

			blocks = append(blocks, b)
		}
	default:
		return fmt.Errorf("unsupported import format: %s", format)
	}

	// Save chain
	chainKey := getChainKey(chainType, repoID)

	// Create or update chain metadata
	is.chains[chainKey] = storage.ChainMetadata{
		ChainType:   chainType,
		RepoID:      repoID,
		BlockCount:  len(blocks),
		LastUpdated: time.Now(),
		IPFSCID:     is.cidMapping[chainKey],
	}

	// Create blocks map if it doesn't exist
	if _, ok := is.blocks[chainKey]; !ok {
		is.blocks[chainKey] = make(map[string]*block.Block)
	} else {
		// Clear existing blocks
		is.blocks[chainKey] = make(map[string]*block.Block)
	}

	// Save blocks
	for _, b := range blocks {
		is.blocks[chainKey][b.Hash] = b
	}

	return nil
}

// BackupStorage creates a backup of the entire storage
func (is *IndexedDBStorage) BackupStorage(ctx context.Context, backupPath string) error {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	// In a real implementation, this would export the IndexedDB database
	// to a file at the specified path

	// For the mock implementation, we'll just return nil
	return nil
}

// RestoreStorage restores the storage from a backup
func (is *IndexedDBStorage) RestoreStorage(ctx context.Context, backupPath string) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	// In a real implementation, this would import the IndexedDB database
	// from a file at the specified path

	// For the mock implementation, we'll just return nil
	return nil
}

// GetStorageInfo returns information about the storage
func (is *IndexedDBStorage) GetStorageInfo(ctx context.Context) (map[string]interface{}, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Count blocks
	var blockCount int
	for _, blockMap := range is.blocks {
		blockCount += len(blockMap)
	}

	return map[string]interface{}{
		"type":         "indexeddb",
		"db_name":      is.config.DBName,
		"chains_count": len(is.chains),
		"blocks_count": blockCount,
		"is_open":      is.isOpen,
	}, nil
}

// SearchBlocks searches for blocks across all chains
func (is *IndexedDBStorage) SearchBlocks(ctx context.Context, query string, options *storage.QueryOptions) ([]*block.Block, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return nil, storage.ErrStorageClosed
	}

	var results []*block.Block

	// Search in each chain
	for _, blockMap := range is.blocks {
		for _, b := range blockMap {
			// Convert block to JSON for searching
			blockData, err := json.Marshal(b)
			if err != nil {
				continue
			}

			// Check if query is in block data
			if strings.Contains(string(blockData), query) {
				results = append(results, b)
			}
		}
	}

	// Apply filters if options are provided
	if options != nil && len(options.Filters) > 0 {
		filteredResults := make([]*block.Block, 0)
		for _, b := range results {
			if applyBlockFilters(b, options.Filters) {
				filteredResults = append(filteredResults, b)
			}
		}
		results = filteredResults
	}

	// Apply limit and offset if options are provided
	if options != nil {
		// Apply offset
		if options.Offset > 0 && options.Offset < len(results) {
			results = results[options.Offset:]
		}

		// Apply limit
		if options.Limit > 0 && options.Limit < len(results) {
			results = results[:options.Limit]
		}
	}

	return results, nil
}

// applyBlockFilters applies filters to a block
func applyBlockFilters(b *block.Block, filters []storage.Filter) bool {
	// Extract chain type and repo ID from block ID
	chainType, repoID := extractChainTypeAndRepoID(b.ID)

	for _, filter := range filters {
		switch filter.Field {
		case "hash":
			if !applyStringFilter(b.Hash, filter) {
				return false
			}
		case "previous_hash":
			if !applyStringFilter(b.PreviousHash, filter) {
				return false
			}
		case "index":
			if !applyUint64Filter(b.Index, filter) {
				return false
			}
		case "timestamp":
			if !applyInt64Filter(b.Timestamp, filter) {
				return false
			}
		case "chain_type":
			if !applyStringFilter(chainType, filter) {
				return false
			}
		case "repo_id":
			if !applyStringFilter(repoID, filter) {
				return false
			}
		}
	}
	return true
}

// extractChainTypeAndRepoID extracts the chain type and repo ID from a block ID
// Block ID format: chain://{chainType}/{repoID}/block{uuid}
func extractChainTypeAndRepoID(blockID string) (string, string) {
	// Check if the ID has the expected format
	if !strings.HasPrefix(blockID, "chain://") {
		return "", ""
	}

	// Remove the "chain://" prefix
	parts := strings.TrimPrefix(blockID, "chain://")

	// Split by "/"
	segments := strings.Split(parts, "/")

	// Check if we have at least 3 segments (chainType, repoID, blockUUID)
	if len(segments) < 3 {
		return "", ""
	}

	return segments[0], segments[1]
}

// applyUint64Filter applies a filter to a uint64 value
func applyUint64Filter(value uint64, filter storage.Filter) bool {
	filterValue, ok := filter.Value.(float64)
	if !ok {
		return false
	}

	uint64FilterValue := uint64(filterValue)

	switch filter.Operator {
	case "eq":
		return value == uint64FilterValue
	case "neq":
		return value != uint64FilterValue
	case "gt":
		return value > uint64FilterValue
	case "gte":
		return value >= uint64FilterValue
	case "lt":
		return value < uint64FilterValue
	case "lte":
		return value <= uint64FilterValue
	default:
		return false
	}
}

// applyInt64Filter applies a filter to an int64 value
func applyInt64Filter(value int64, filter storage.Filter) bool {
	filterValue, ok := filter.Value.(float64)
	if !ok {
		return false
	}

	int64FilterValue := int64(filterValue)

	switch filter.Operator {
	case "eq":
		return value == int64FilterValue
	case "neq":
		return value != int64FilterValue
	case "gt":
		return value > int64FilterValue
	case "gte":
		return value >= int64FilterValue
	case "lt":
		return value < int64FilterValue
	case "lte":
		return value <= int64FilterValue
	default:
		return false
	}
}

// GetIPFSCID returns the IPFS CID for a chain
func (is *IndexedDBStorage) GetIPFSCID(ctx context.Context, chainType, repoID string) (string, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return "", storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)
	cid, ok := is.cidMapping[chainKey]
	if !ok {
		return "", storage.ErrNotFound
	}

	return cid, nil
}

// SetIPFSCID sets the IPFS CID for a chain
func (is *IndexedDBStorage) SetIPFSCID(ctx context.Context, chainType, repoID, cid string) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := getChainKey(chainType, repoID)
	is.cidMapping[chainKey] = cid

	// Update chain metadata if it exists
	if metadata, ok := is.chains[chainKey]; ok {
		metadata.IPFSCID = cid
		metadata.LastUpdated = time.Now()
		is.chains[chainKey] = metadata
	}

	return nil
}
