package ipfs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
)

// IPFSStorage implements the Storage interface using IPFS
// This is a mock implementation since IPFS integration requires additional dependencies
// In a real implementation, this would use the IPFS HTTP API or go-ipfs-api
type IPFSStorage struct {
	config     *storage.StorageConfig
	mutex      sync.RWMutex
	isOpen     bool
	cidMapping map[string]string // Maps chainType+repoID to IPFS CID

	// In-memory storage for mock implementation
	chains map[string]storage.ChainMetadata
	blocks map[string]map[string]*block.Block // chainKey -> blockHash -> block

	// Local cache directory
	cacheDir string
}

// NewIPFSStorage creates a new IPFS storage backend
func NewIPFSStorage(config *storage.StorageConfig) (*IPFSStorage, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.IPFSHost == "" {
		return nil, fmt.Errorf("IPFS host cannot be empty")
	}

	// Default port if not specified
	if config.IPFSPort == 0 {
		config.IPFSPort = 5001
	}

	// Default protocol if not specified
	if config.IPFSProtocol == "" {
		config.IPFSProtocol = "http"
	}

	// Default cache directory if not specified
	cacheDir := config.Path
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "theo-ipfs-cache")
	}

	return &IPFSStorage{
		config:     config,
		isOpen:     false,
		cidMapping: make(map[string]string),
		chains:     make(map[string]storage.ChainMetadata),
		blocks:     make(map[string]map[string]*block.Block),
		cacheDir:   cacheDir,
	}, nil
}

// getChainKey returns a unique key for a chain
func getChainKey(chainType, repoID string) string {
	return chainType + ":" + repoID
}

// Open initializes the IPFS storage backend
func (is *IPFSStorage) Open(ctx context.Context) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if is.isOpen {
		return nil
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(is.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// In a real implementation, this would initialize the IPFS client
	// and test the connection to the IPFS node

	is.isOpen = true
	return nil
}

// Close closes the IPFS storage backend
func (is *IPFSStorage) Close() error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return nil
	}

	// In a real implementation, this would close the IPFS client

	is.isOpen = false
	return nil
}

// SaveChain saves a chain to storage
func (is *IPFSStorage) SaveChain(ctx context.Context, chainType, repoID string, blocks []*block.Block) error {
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

	// In a real implementation, this would add the chain to IPFS
	// and update the CID mapping with the new CID

	// Mock implementation: generate a fake CID
	is.cidMapping[chainKey] = generateFakeCID(chainType, repoID)

	// Update chain metadata with new CID
	metadata := is.chains[chainKey]
	metadata.IPFSCID = is.cidMapping[chainKey]
	is.chains[chainKey] = metadata

	return nil
}

// generateFakeCID generates a fake IPFS CID for testing
func generateFakeCID(chainType, repoID string) string {
	// In a real implementation, this would be the actual CID returned by IPFS
	// For the mock implementation, we'll generate a deterministic fake CID
	return fmt.Sprintf("Qm%x", []byte(chainType+repoID))
}

// LoadChain loads a chain from storage
func (is *IPFSStorage) LoadChain(ctx context.Context, chainType, repoID string) ([]*block.Block, error) {
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
func (is *IPFSStorage) DeleteChain(ctx context.Context, chainType, repoID string) error {
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

	// In a real implementation, this would not actually delete the data from IPFS
	// since IPFS content is immutable and persistent
	// It would just remove the local reference to the chain

	return nil
}

// ListChains lists all chains in storage
func (is *IPFSStorage) ListChains(ctx context.Context, options *storage.QueryOptions) ([]storage.ChainMetadata, error) {
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
func (is *IPFSStorage) SaveBlock(ctx context.Context, chainType, repoID string, b *block.Block) error {
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

	// In a real implementation, this would add the block to IPFS
	// and update the chain's CID in IPFS

	// Mock implementation: update the fake CID
	is.cidMapping[chainKey] = generateFakeCID(chainType, repoID)
	metadata.IPFSCID = is.cidMapping[chainKey]
	is.chains[chainKey] = metadata

	return nil
}

// LoadBlock loads a block from a chain
func (is *IPFSStorage) LoadBlock(ctx context.Context, chainType, repoID string, blockHash string) (*block.Block, error) {
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
func (is *IPFSStorage) LoadBlockByIndex(ctx context.Context, chainType, repoID string, blockIndex uint64) (*block.Block, error) {
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
func (is *IPFSStorage) DeleteBlock(ctx context.Context, chainType, repoID string, blockHash string) error {
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

	// In a real implementation, this would not actually delete the block from IPFS
	// since IPFS content is immutable and persistent
	// It would just remove the local reference to the block and update the chain's CID

	// Mock implementation: update the fake CID
	is.cidMapping[chainKey] = generateFakeCID(chainType, repoID)
	metadata.IPFSCID = is.cidMapping[chainKey]
	is.chains[chainKey] = metadata

	return nil
}

// ExportChain exports a chain to a byte array
func (is *IPFSStorage) ExportChain(ctx context.Context, chainType, repoID string, format string) ([]byte, error) {
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
func (is *IPFSStorage) ImportChain(ctx context.Context, chainType, repoID string, data []byte, format string) error {
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

	// In a real implementation, this would add the chain to IPFS
	// and update the CID mapping with the new CID

	// Mock implementation: generate a fake CID if not already set
	if is.cidMapping[chainKey] == "" {
		is.cidMapping[chainKey] = generateFakeCID(chainType, repoID)
	}

	// Update chain metadata with CID
	metadata := is.chains[chainKey]
	metadata.IPFSCID = is.cidMapping[chainKey]
	is.chains[chainKey] = metadata

	return nil
}

// BackupStorage creates a backup of the entire storage
func (is *IPFSStorage) BackupStorage(ctx context.Context, backupPath string) error {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// In a real implementation, this would export all chains from IPFS
	// to the backup directory

	// For the mock implementation, we'll just save the CID mapping
	cidMappingPath := filepath.Join(backupPath, "cid_mapping.json")
	cidMappingData, err := json.Marshal(is.cidMapping)
	if err != nil {
		return fmt.Errorf("failed to marshal CID mapping: %w", err)
	}

	if err := os.WriteFile(cidMappingPath, cidMappingData, 0644); err != nil {
		return fmt.Errorf("failed to write CID mapping: %w", err)
	}

	// Save chains metadata
	chainsPath := filepath.Join(backupPath, "chains.json")
	chainsData, err := json.Marshal(is.chains)
	if err != nil {
		return fmt.Errorf("failed to marshal chains: %w", err)
	}

	if err := os.WriteFile(chainsPath, chainsData, 0644); err != nil {
		return fmt.Errorf("failed to write chains: %w", err)
	}

	// Save blocks
	blocksDir := filepath.Join(backupPath, "blocks")
	if err := os.MkdirAll(blocksDir, 0755); err != nil {
		return fmt.Errorf("failed to create blocks directory: %w", err)
	}

	for chainKey, blockMap := range is.blocks {
		chainDir := filepath.Join(blocksDir, chainKey)
		if err := os.MkdirAll(chainDir, 0755); err != nil {
			return fmt.Errorf("failed to create chain directory: %w", err)
		}

		for blockHash, block := range blockMap {
			blockPath := filepath.Join(chainDir, blockHash+".json")
			blockData, err := json.Marshal(block)
			if err != nil {
				return fmt.Errorf("failed to marshal block: %w", err)
			}

			if err := os.WriteFile(blockPath, blockData, 0644); err != nil {
				return fmt.Errorf("failed to write block: %w", err)
			}
		}
	}

	return nil
}

// RestoreStorage restores the storage from a backup
func (is *IPFSStorage) RestoreStorage(ctx context.Context, backupPath string) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if !is.isOpen {
		return storage.ErrStorageClosed
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup path does not exist: %w", err)
	}

	// In a real implementation, this would import all chains from the backup
	// to IPFS

	// For the mock implementation, we'll just load the CID mapping
	cidMappingPath := filepath.Join(backupPath, "cid_mapping.json")
	cidMappingData, err := os.ReadFile(cidMappingPath)
	if err != nil {
		return fmt.Errorf("failed to read CID mapping: %w", err)
	}

	if err := json.Unmarshal(cidMappingData, &is.cidMapping); err != nil {
		return fmt.Errorf("failed to unmarshal CID mapping: %w", err)
	}

	// Load chains metadata
	chainsPath := filepath.Join(backupPath, "chains.json")
	chainsData, err := os.ReadFile(chainsPath)
	if err != nil {
		return fmt.Errorf("failed to read chains: %w", err)
	}

	if err := json.Unmarshal(chainsData, &is.chains); err != nil {
		return fmt.Errorf("failed to unmarshal chains: %w", err)
	}

	// Load blocks
	blocksDir := filepath.Join(backupPath, "blocks")
	if _, err := os.Stat(blocksDir); err != nil {
		return fmt.Errorf("blocks directory does not exist: %w", err)
	}

	// Clear existing blocks
	is.blocks = make(map[string]map[string]*block.Block)

	// Read chain directories
	chainDirs, err := os.ReadDir(blocksDir)
	if err != nil {
		return fmt.Errorf("failed to read blocks directory: %w", err)
	}

	for _, chainDir := range chainDirs {
		if !chainDir.IsDir() {
			continue
		}

		chainKey := chainDir.Name()
		is.blocks[chainKey] = make(map[string]*block.Block)

		// Read block files
		blockFiles, err := os.ReadDir(filepath.Join(blocksDir, chainKey))
		if err != nil {
			return fmt.Errorf("failed to read chain directory: %w", err)
		}

		for _, blockFile := range blockFiles {
			if blockFile.IsDir() {
				continue
			}

			blockPath := filepath.Join(blocksDir, chainKey, blockFile.Name())
			blockData, err := os.ReadFile(blockPath)
			if err != nil {
				return fmt.Errorf("failed to read block: %w", err)
			}

			var b block.Block
			if err := json.Unmarshal(blockData, &b); err != nil {
				return fmt.Errorf("failed to unmarshal block: %w", err)
			}

			is.blocks[chainKey][b.Hash] = &b
		}
	}

	return nil
}

// GetStorageInfo returns information about the storage
func (is *IPFSStorage) GetStorageInfo(ctx context.Context) (map[string]interface{}, error) {
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
		"type":         "ipfs",
		"host":         is.config.IPFSHost,
		"port":         is.config.IPFSPort,
		"protocol":     is.config.IPFSProtocol,
		"cache_dir":    is.cacheDir,
		"chains_count": len(is.chains),
		"blocks_count": blockCount,
		"is_open":      is.isOpen,
	}, nil
}

// SearchBlocks searches for blocks across all chains
func (is *IPFSStorage) SearchBlocks(ctx context.Context, query string, options *storage.QueryOptions) ([]*block.Block, error) {
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
func (is *IPFSStorage) GetIPFSCID(ctx context.Context, chainType, repoID string) (string, error) {
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
func (is *IPFSStorage) SetIPFSCID(ctx context.Context, chainType, repoID, cid string) error {
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
