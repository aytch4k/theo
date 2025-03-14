package file

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

// FileStorage implements the Storage interface using the local filesystem
type FileStorage struct {
	config     *storage.StorageConfig
	basePath   string
	mutex      sync.RWMutex
	isOpen     bool
	cidMapping map[string]string // Maps chainType+repoID to IPFS CID
}

// NewFileStorage creates a new file storage backend
func NewFileStorage(config *storage.StorageConfig) (*FileStorage, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	return &FileStorage{
		config:     config,
		basePath:   config.Path,
		isOpen:     false,
		cidMapping: make(map[string]string),
	}, nil
}

// Open initializes the file storage backend
func (fs *FileStorage) Open(ctx context.Context) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.isOpen {
		return nil
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(fs.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create chains directory
	chainsDir := filepath.Join(fs.basePath, "chains")
	if err := os.MkdirAll(chainsDir, 0755); err != nil {
		return fmt.Errorf("failed to create chains directory: %w", err)
	}

	// Create backups directory
	backupsDir := filepath.Join(fs.basePath, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return fmt.Errorf("failed to create backups directory: %w", err)
	}

	// Load CID mapping if it exists
	cidMappingPath := filepath.Join(fs.basePath, "cid_mapping.json")
	if _, err := os.Stat(cidMappingPath); err == nil {
		data, err := os.ReadFile(cidMappingPath)
		if err != nil {
			return fmt.Errorf("failed to read CID mapping: %w", err)
		}

		if err := json.Unmarshal(data, &fs.cidMapping); err != nil {
			return fmt.Errorf("failed to unmarshal CID mapping: %w", err)
		}
	}

	fs.isOpen = true
	return nil
}

// Close closes the file storage backend
func (fs *FileStorage) Close() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
		return nil
	}

	// Save CID mapping
	cidMappingPath := filepath.Join(fs.basePath, "cid_mapping.json")
	data, err := json.Marshal(fs.cidMapping)
	if err != nil {
		return fmt.Errorf("failed to marshal CID mapping: %w", err)
	}

	if err := os.WriteFile(cidMappingPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write CID mapping: %w", err)
	}

	fs.isOpen = false
	return nil
}

// getChainPath returns the path to a chain's directory
func (fs *FileStorage) getChainPath(chainType, repoID string) string {
	return filepath.Join(fs.basePath, "chains", repoID, chainType)
}

// getChainFilePath returns the path to a chain's file
func (fs *FileStorage) getChainFilePath(chainType, repoID string) string {
	return filepath.Join(fs.getChainPath(chainType, repoID), "chain.json")
}

// getBlockFilePath returns the path to a block's file
func (fs *FileStorage) getBlockFilePath(chainType, repoID string, blockHash string) string {
	return filepath.Join(fs.getChainPath(chainType, repoID), "blocks", blockHash+".json")
}

// getMetadataFilePath returns the path to a chain's metadata file
func (fs *FileStorage) getMetadataFilePath(chainType, repoID string) string {
	return filepath.Join(fs.getChainPath(chainType, repoID), "metadata.json")
}

// SaveChain saves a chain to storage
func (fs *FileStorage) SaveChain(ctx context.Context, chainType, repoID string, blocks []*block.Block) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
		return storage.ErrStorageClosed
	}

	// Create chain directory
	chainPath := fs.getChainPath(chainType, repoID)
	if err := os.MkdirAll(chainPath, 0755); err != nil {
		return fmt.Errorf("failed to create chain directory: %w", err)
	}

	// Create blocks directory
	blocksPath := filepath.Join(chainPath, "blocks")
	if err := os.MkdirAll(blocksPath, 0755); err != nil {
		return fmt.Errorf("failed to create blocks directory: %w", err)
	}

	// Save each block individually
	for _, b := range blocks {
		blockData, err := json.Marshal(b)
		if err != nil {
			return fmt.Errorf("failed to marshal block: %w", err)
		}

		blockPath := fs.getBlockFilePath(chainType, repoID, b.Hash)
		if err := os.WriteFile(blockPath, blockData, 0644); err != nil {
			return fmt.Errorf("failed to write block file: %w", err)
		}
	}

	// Save chain metadata
	metadata := storage.ChainMetadata{
		ChainType:   chainType,
		RepoID:      repoID,
		BlockCount:  len(blocks),
		LastUpdated: time.Now(),
		IPFSCID:     fs.cidMapping[chainType+repoID],
	}

	metadataData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := fs.getMetadataFilePath(chainType, repoID)
	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	// Save chain file with block hashes
	type chainFile struct {
		ChainType   string   `json:"chain_type"`
		RepoID      string   `json:"repo_id"`
		BlockHashes []string `json:"block_hashes"`
	}

	blockHashes := make([]string, len(blocks))
	for i, b := range blocks {
		blockHashes[i] = b.Hash
	}

	chainData := chainFile{
		ChainType:   chainType,
		RepoID:      repoID,
		BlockHashes: blockHashes,
	}

	chainDataBytes, err := json.Marshal(chainData)
	if err != nil {
		return fmt.Errorf("failed to marshal chain data: %w", err)
	}

	chainFilePath := fs.getChainFilePath(chainType, repoID)
	if err := os.WriteFile(chainFilePath, chainDataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write chain file: %w", err)
	}

	return nil
}

// LoadChain loads a chain from storage
func (fs *FileStorage) LoadChain(ctx context.Context, chainType, repoID string) ([]*block.Block, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Read chain file
	chainFilePath := fs.getChainFilePath(chainType, repoID)
	chainData, err := os.ReadFile(chainFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read chain file: %w", err)
	}

	// Parse chain file
	type chainFile struct {
		ChainType   string   `json:"chain_type"`
		RepoID      string   `json:"repo_id"`
		BlockHashes []string `json:"block_hashes"`
	}

	var chain chainFile
	if err := json.Unmarshal(chainData, &chain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chain data: %w", err)
	}

	// Load each block
	blocks := make([]*block.Block, len(chain.BlockHashes))
	for i, hash := range chain.BlockHashes {
		blockPath := fs.getBlockFilePath(chainType, repoID, hash)
		blockData, err := os.ReadFile(blockPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read block file: %w", err)
		}

		var b block.Block
		if err := json.Unmarshal(blockData, &b); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
		}

		blocks[i] = &b
	}

	return blocks, nil
}

// DeleteChain deletes a chain from storage
func (fs *FileStorage) DeleteChain(ctx context.Context, chainType, repoID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
		return storage.ErrStorageClosed
	}

	// Delete chain directory
	chainPath := fs.getChainPath(chainType, repoID)
	if err := os.RemoveAll(chainPath); err != nil {
		return fmt.Errorf("failed to delete chain directory: %w", err)
	}

	// Remove from CID mapping
	delete(fs.cidMapping, chainType+repoID)

	return nil
}

// ListChains lists all chains in storage
func (fs *FileStorage) ListChains(ctx context.Context, options *storage.QueryOptions) ([]storage.ChainMetadata, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Get all repository directories
	reposPath := filepath.Join(fs.basePath, "chains")
	repoDirs, err := os.ReadDir(reposPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []storage.ChainMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read repos directory: %w", err)
	}

	var chains []storage.ChainMetadata

	// Iterate through each repository
	for _, repoDir := range repoDirs {
		if !repoDir.IsDir() {
			continue
		}

		repoID := repoDir.Name()
		repoPath := filepath.Join(reposPath, repoID)

		// Get all chain directories
		chainDirs, err := os.ReadDir(repoPath)
		if err != nil {
			continue
		}

		// Iterate through each chain
		for _, chainDir := range chainDirs {
			if !chainDir.IsDir() {
				continue
			}

			chainType := chainDir.Name()
			metadataPath := fs.getMetadataFilePath(chainType, repoID)

			// Read metadata file
			metadataData, err := os.ReadFile(metadataPath)
			if err != nil {
				continue
			}

			var metadata storage.ChainMetadata
			if err := json.Unmarshal(metadataData, &metadata); err != nil {
				continue
			}

			chains = append(chains, metadata)
		}
	}

	// Apply filters if options are provided
	if options != nil && len(options.Filters) > 0 {
		filteredChains := make([]storage.ChainMetadata, 0)
		for _, chain := range chains {
			if applyFilters(chain, options.Filters) {
				filteredChains = append(filteredChains, chain)
			}
		}
		chains = filteredChains
	}

	return chains, nil
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
		return contains(value, filterValue)
	case "startswith":
		return startsWith(value, filterValue)
	case "endswith":
		return endsWith(value, filterValue)
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

// contains checks if a string contains another string
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// startsWith checks if a string starts with another string
func startsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// endsWith checks if a string ends with another string
func endsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// SaveBlock saves a block to a chain
func (fs *FileStorage) SaveBlock(ctx context.Context, chainType, repoID string, b *block.Block) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
		return storage.ErrStorageClosed
	}

	// Create chain directory
	chainPath := fs.getChainPath(chainType, repoID)
	if err := os.MkdirAll(chainPath, 0755); err != nil {
		return fmt.Errorf("failed to create chain directory: %w", err)
	}

	// Create blocks directory
	blocksPath := filepath.Join(chainPath, "blocks")
	if err := os.MkdirAll(blocksPath, 0755); err != nil {
		return fmt.Errorf("failed to create blocks directory: %w", err)
	}

	// Save block
	blockData, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	blockPath := fs.getBlockFilePath(chainType, repoID, b.Hash)
	if err := os.WriteFile(blockPath, blockData, 0644); err != nil {
		return fmt.Errorf("failed to write block file: %w", err)
	}

	// Update chain file
	chainFilePath := fs.getChainFilePath(chainType, repoID)
	var chainData []byte
	var chain struct {
		ChainType   string   `json:"chain_type"`
		RepoID      string   `json:"repo_id"`
		BlockHashes []string `json:"block_hashes"`
	}

	// Read existing chain file if it exists
	if _, err := os.Stat(chainFilePath); err == nil {
		chainData, err = os.ReadFile(chainFilePath)
		if err != nil {
			return fmt.Errorf("failed to read chain file: %w", err)
		}

		if err := json.Unmarshal(chainData, &chain); err != nil {
			return fmt.Errorf("failed to unmarshal chain data: %w", err)
		}
	} else {
		// Create new chain file
		chain = struct {
			ChainType   string   `json:"chain_type"`
			RepoID      string   `json:"repo_id"`
			BlockHashes []string `json:"block_hashes"`
		}{
			ChainType:   chainType,
			RepoID:      repoID,
			BlockHashes: []string{},
		}
	}

	// Add block hash to chain
	chain.BlockHashes = append(chain.BlockHashes, b.Hash)

	// Save chain file
	chainData, err = json.Marshal(chain)
	if err != nil {
		return fmt.Errorf("failed to marshal chain data: %w", err)
	}

	if err := os.WriteFile(chainFilePath, chainData, 0644); err != nil {
		return fmt.Errorf("failed to write chain file: %w", err)
	}

	// Update metadata
	metadataPath := fs.getMetadataFilePath(chainType, repoID)
	var metadata storage.ChainMetadata

	// Read existing metadata if it exists
	if _, err := os.Stat(metadataPath); err == nil {
		metadataData, err := os.ReadFile(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to read metadata file: %w", err)
		}

		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		// Create new metadata
		metadata = storage.ChainMetadata{
			ChainType:   chainType,
			RepoID:      repoID,
			BlockCount:  0,
			LastUpdated: time.Now(),
			IPFSCID:     fs.cidMapping[chainType+repoID],
		}
	}

	// Update metadata
	metadata.BlockCount++
	metadata.LastUpdated = time.Now()

	// Save metadata
	metadataData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// LoadBlock loads a block from a chain
func (fs *FileStorage) LoadBlock(ctx context.Context, chainType, repoID string, blockHash string) (*block.Block, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Read block file
	blockPath := fs.getBlockFilePath(chainType, repoID, blockHash)
	blockData, err := os.ReadFile(blockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read block file: %w", err)
	}

	// Parse block
	var b block.Block
	if err := json.Unmarshal(blockData, &b); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
	}

	return &b, nil
}

// LoadBlockByIndex loads a block from a chain by its index
func (fs *FileStorage) LoadBlockByIndex(ctx context.Context, chainType, repoID string, blockIndex uint64) (*block.Block, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Load chain
	blocks, err := fs.LoadChain(ctx, chainType, repoID)
	if err != nil {
		return nil, err
	}

	// Find block by index
	for _, b := range blocks {
		if b.Index == blockIndex {
			return b, nil
		}
	}

	return nil, storage.ErrNotFound
}

// DeleteBlock deletes a block from a chain
func (fs *FileStorage) DeleteBlock(ctx context.Context, chainType, repoID string, blockHash string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
		return storage.ErrStorageClosed
	}

	// Delete block file
	blockPath := fs.getBlockFilePath(chainType, repoID, blockHash)
	if err := os.Remove(blockPath); err != nil {
		if os.IsNotExist(err) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("failed to delete block file: %w", err)
	}

	// Update chain file
	chainFilePath := fs.getChainFilePath(chainType, repoID)
	chainData, err := os.ReadFile(chainFilePath)
	if err != nil {
		return fmt.Errorf("failed to read chain file: %w", err)
	}

	var chain struct {
		ChainType   string   `json:"chain_type"`
		RepoID      string   `json:"repo_id"`
		BlockHashes []string `json:"block_hashes"`
	}

	if err := json.Unmarshal(chainData, &chain); err != nil {
		return fmt.Errorf("failed to unmarshal chain data: %w", err)
	}

	// Remove block hash from chain
	newBlockHashes := make([]string, 0, len(chain.BlockHashes))
	for _, hash := range chain.BlockHashes {
		if hash != blockHash {
			newBlockHashes = append(newBlockHashes, hash)
		}
	}
	chain.BlockHashes = newBlockHashes

	// Save chain file
	chainData, err = json.Marshal(chain)
	if err != nil {
		return fmt.Errorf("failed to marshal chain data: %w", err)
	}

	if err := os.WriteFile(chainFilePath, chainData, 0644); err != nil {
		return fmt.Errorf("failed to write chain file: %w", err)
	}

	// Update metadata
	metadataPath := fs.getMetadataFilePath(chainType, repoID)
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata storage.ChainMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Update metadata
	metadata.BlockCount--
	metadata.LastUpdated = time.Now()

	// Save metadata
	metadataData, err = json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// ExportChain exports a chain to a byte array
func (fs *FileStorage) ExportChain(ctx context.Context, chainType, repoID string, format string) ([]byte, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Load chain
	blocks, err := fs.LoadChain(ctx, chainType, repoID)
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

		chain := exportedChain{
			ChainType: chainType,
			RepoID:    repoID,
			Blocks:    blocks,
			IPFSCID:   fs.cidMapping[chainType+repoID],
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
func (fs *FileStorage) ImportChain(ctx context.Context, chainType, repoID string, data []byte, format string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
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
		if chain.IPFSCID != "" {
			fs.cidMapping[chainType+repoID] = chain.IPFSCID
		}
	case "jsonl":
		// Import from JSONL
		lines := splitLines(string(data))
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
	return fs.SaveChain(ctx, chainType, repoID, blocks)
}

// BackupStorage creates a backup of the entire storage
func (fs *FileStorage) BackupStorage(ctx context.Context, backupPath string) error {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return storage.ErrStorageClosed
	}

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy chains directory
	chainsDir := filepath.Join(fs.basePath, "chains")
	backupChainsDir := filepath.Join(backupPath, "chains")
	if err := copyDir(chainsDir, backupChainsDir); err != nil {
		return fmt.Errorf("failed to copy chains directory: %w", err)
	}

	// Save CID mapping
	cidMappingPath := filepath.Join(backupPath, "cid_mapping.json")
	data, err := json.Marshal(fs.cidMapping)
	if err != nil {
		return fmt.Errorf("failed to marshal CID mapping: %w", err)
	}

	if err := os.WriteFile(cidMappingPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write CID mapping: %w", err)
	}

	return nil
}

// RestoreStorage restores the storage from a backup
func (fs *FileStorage) RestoreStorage(ctx context.Context, backupPath string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
		return storage.ErrStorageClosed
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup path does not exist: %w", err)
	}

	// Clear current storage
	chainsDir := filepath.Join(fs.basePath, "chains")
	if err := os.RemoveAll(chainsDir); err != nil {
		return fmt.Errorf("failed to clear chains directory: %w", err)
	}

	// Copy backup chains directory
	backupChainsDir := filepath.Join(backupPath, "chains")
	if err := copyDir(backupChainsDir, chainsDir); err != nil {
		return fmt.Errorf("failed to copy backup chains directory: %w", err)
	}

	// Load CID mapping
	cidMappingPath := filepath.Join(backupPath, "cid_mapping.json")
	data, err := os.ReadFile(cidMappingPath)
	if err != nil {
		return fmt.Errorf("failed to read CID mapping: %w", err)
	}

	if err := json.Unmarshal(data, &fs.cidMapping); err != nil {
		return fmt.Errorf("failed to unmarshal CID mapping: %w", err)
	}

	return nil
}

// GetStorageInfo returns information about the storage
func (fs *FileStorage) GetStorageInfo(ctx context.Context) (map[string]interface{}, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Get chains count
	chains, err := fs.ListChains(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Get total size
	var totalSize int64
	err = filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate storage size: %w", err)
	}

	return map[string]interface{}{
		"type":         "file",
		"path":         fs.basePath,
		"chains_count": len(chains),
		"total_size":   totalSize,
		"is_open":      fs.isOpen,
	}, nil
}

// SearchBlocks searches for blocks across all chains
func (fs *FileStorage) SearchBlocks(ctx context.Context, query string, options *storage.QueryOptions) ([]*block.Block, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Get all chains
	chains, err := fs.ListChains(ctx, nil)
	if err != nil {
		return nil, err
	}

	var results []*block.Block

	// Search in each chain
	for _, chain := range chains {
		// Load chain
		blocks, err := fs.LoadChain(ctx, chain.ChainType, chain.RepoID)
		if err != nil {
			continue
		}

		// Search in blocks
		for _, b := range blocks {
			// Convert block to JSON for searching
			blockData, err := json.Marshal(b)
			if err != nil {
				continue
			}

			// Check if query is in block data
			if contains(string(blockData), query) {
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
func (fs *FileStorage) GetIPFSCID(ctx context.Context, chainType, repoID string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if !fs.isOpen {
		return "", storage.ErrStorageClosed
	}

	cid, ok := fs.cidMapping[chainType+repoID]
	if !ok {
		return "", storage.ErrNotFound
	}

	return cid, nil
}

// SetIPFSCID sets the IPFS CID for a chain
func (fs *FileStorage) SetIPFSCID(ctx context.Context, chainType, repoID, cid string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if !fs.isOpen {
		return storage.ErrStorageClosed
	}

	fs.cidMapping[chainType+repoID] = cid

	// Update metadata
	metadataPath := fs.getMetadataFilePath(chainType, repoID)
	if _, err := os.Stat(metadataPath); err == nil {
		metadataData, err := os.ReadFile(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to read metadata file: %w", err)
		}

		var metadata storage.ChainMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		metadata.IPFSCID = cid
		metadata.LastUpdated = time.Now()

		metadataData, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
			return fmt.Errorf("failed to write metadata file: %w", err)
		}
	}

	return nil
}

// Helper functions

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursive copy for directories
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a file
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	_, err = dstFile.ReadFrom(srcFile)
	return err
}

// splitLines splits a string by newlines
func splitLines(s string) []string {
	var lines []string
	var line string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
