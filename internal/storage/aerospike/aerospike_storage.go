package aerospike

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v6"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
)

// AerospikeStorage implements the Storage interface using Aerospike
type AerospikeStorage struct {
	config     *storage.StorageConfig
	client     *aero.Client
	policy     *aero.WritePolicy
	readPolicy *aero.BasePolicy
	mutex      sync.RWMutex
	isOpen     bool
	namespace  string
	set        string
}

// Record keys
const (
	chainMetadataKey = "chain_metadata"
	blockKey         = "block"
	cidMappingKey    = "cid_mapping"
)

// NewAerospikeStorage creates a new Aerospike storage backend
func NewAerospikeStorage(config *storage.StorageConfig) (*AerospikeStorage, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.AerospikeHost == "" {
		return nil, fmt.Errorf("Aerospike host cannot be empty")
	}

	// Default port if not specified
	if config.AerospikePort == 0 {
		config.AerospikePort = 3000
	}

	// Default namespace if not specified
	namespace := config.AerospikeNamespace
	if namespace == "" {
		namespace = "test"
	}

	// Default set if not specified
	set := config.AerospikeSet
	if set == "" {
		set = "theo"
	}

	return &AerospikeStorage{
		config:    config,
		isOpen:    false,
		namespace: namespace,
		set:       set,
	}, nil
}

// Open initializes the Aerospike storage backend
func (as *AerospikeStorage) Open(ctx context.Context) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if as.isOpen {
		return nil
	}

	// Create client policy
	clientPolicy := aero.NewClientPolicy()

	// Set authentication if provided
	if as.config.AerospikeUsername != "" && as.config.AerospikePassword != "" {
		clientPolicy.User = as.config.AerospikeUsername
		clientPolicy.Password = as.config.AerospikePassword
	}

	// Connect to Aerospike server
	client, err := aero.NewClientWithPolicy(clientPolicy, as.config.AerospikeHost, as.config.AerospikePort)
	if err != nil {
		return fmt.Errorf("failed to connect to Aerospike: %w", err)
	}

	// Create write policy
	writePolicy := aero.NewWritePolicy(0, 0)
	writePolicy.SendKey = true

	// Create read policy
	readPolicy := aero.NewPolicy()

	as.client = client
	as.policy = writePolicy
	as.readPolicy = readPolicy
	as.isOpen = true

	return nil
}

// Close closes the Aerospike storage backend
func (as *AerospikeStorage) Close() error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return nil
	}

	as.client.Close()
	as.isOpen = false
	return nil
}

// getChainKey returns a unique key for a chain
func (as *AerospikeStorage) getChainKey(chainType, repoID string) string {
	return fmt.Sprintf("%s:%s", chainType, repoID)
}

// getBlockKey returns a unique key for a block
func (as *AerospikeStorage) getBlockKey(chainType, repoID, blockHash string) string {
	return fmt.Sprintf("%s:%s:%s", chainType, repoID, blockHash)
}

// SaveChain saves a chain to storage
func (as *AerospikeStorage) SaveChain(ctx context.Context, chainType, repoID string, blocks []*block.Block) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := as.getChainKey(chainType, repoID)

	// Create chain metadata
	metadata := storage.ChainMetadata{
		ChainType:   chainType,
		RepoID:      repoID,
		BlockCount:  len(blocks),
		LastUpdated: time.Now(),
	}

	// Get IPFS CID if it exists
	cid, err := as.GetIPFSCID(ctx, chainType, repoID)
	if err == nil {
		metadata.IPFSCID = cid
	}

	// Save chain metadata
	metadataKey, err := aero.NewKey(as.namespace, as.set, chainMetadataKey+":"+chainKey)
	if err != nil {
		return fmt.Errorf("failed to create metadata key: %w", err)
	}

	// Marshal metadata to JSON
	metadataBytes, jsonErr := json.Marshal(metadata)
	if jsonErr != nil {
		return fmt.Errorf("failed to marshal metadata: %w", jsonErr)
	}

	// Create bins map for metadata
	bins := aero.BinMap{
		"metadata": string(metadataBytes),
	}

	if err := as.client.Put(as.policy, metadataKey, bins); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// Save blocks
	for _, b := range blocks {
		if err := as.saveBlock(chainType, repoID, b); err != nil {
			return fmt.Errorf("failed to save block: %w", err)
		}
	}

	return nil
}

// saveBlock saves a block to Aerospike
func (as *AerospikeStorage) saveBlock(chainType, repoID string, b *block.Block) error {
	blockKey := as.getBlockKey(chainType, repoID, b.Hash)
	key, err := aero.NewKey(as.namespace, as.set, blockKey+":"+blockKey)
	if err != nil {
		return fmt.Errorf("failed to create block key: %w", err)
	}

	// Marshal block to JSON
	blockBytes, jsonErr := json.Marshal(b)
	if jsonErr != nil {
		return fmt.Errorf("failed to marshal block: %w", jsonErr)
	}

	// Create bins map for block
	bins := aero.BinMap{
		"block": string(blockBytes),
	}

	if err := as.client.Put(as.policy, key, bins); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	return nil
}

// LoadChain loads a chain from storage
func (as *AerospikeStorage) LoadChain(ctx context.Context, chainType, repoID string) ([]*block.Block, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return nil, storage.ErrStorageClosed
	}

	chainKey := as.getChainKey(chainType, repoID)

	// Check if chain exists
	metadataKey, err := aero.NewKey(as.namespace, as.set, chainMetadataKey+":"+chainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata key: %w", err)
	}

	record, err := as.client.Get(as.readPolicy, metadataKey)
	if err != nil {
		// Check if key not found
		if err == aero.ErrKeyNotFound {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	// Parse metadata
	var metadata storage.ChainMetadata
	metadataJSON, ok := record.Bins["metadata"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid metadata format")
	}

	if jsonErr := json.Unmarshal([]byte(metadataJSON), &metadata); jsonErr != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", jsonErr)
	}

	// Query blocks - use scan and filter manually
	recordset, err := as.client.ScanAll(nil, as.namespace, as.set)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}

	var blocks []*block.Block
	blockKeyPrefix := blockKey + ":" + chainKey

	for res := range recordset.Results() {
		if res.Err != nil {
			return nil, fmt.Errorf("error in query results: %w", res.Err)
		}

		// Check if this is a block for our chain
		keyStr := res.Record.Key.String()
		if !strings.Contains(keyStr, blockKeyPrefix) {
			continue
		}

		blockJSON, ok := res.Record.Bins["block"].(string)
		if !ok {
			continue
		}

		var b block.Block
		if jsonErr := json.Unmarshal([]byte(blockJSON), &b); jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal block: %w", jsonErr)
		}

		blocks = append(blocks, &b)
	}

	// Sort blocks by index
	sortBlocksByIndex(blocks)

	return blocks, nil
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
func (as *AerospikeStorage) DeleteChain(ctx context.Context, chainType, repoID string) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := as.getChainKey(chainType, repoID)

	// Check if chain exists
	metadataKey, err := aero.NewKey(as.namespace, as.set, chainMetadataKey+":"+chainKey)
	if err != nil {
		return fmt.Errorf("failed to create metadata key: %w", err)
	}

	exists, err := as.client.Exists(as.readPolicy, metadataKey)
	if err != nil {
		return fmt.Errorf("failed to check if chain exists: %w", err)
	}

	if !exists {
		return storage.ErrNotFound
	}

	// Delete chain metadata
	if _, err := as.client.Delete(as.policy, metadataKey); err != nil {
		return fmt.Errorf("failed to delete chain metadata: %w", err)
	}

	// Delete blocks - use scan and filter manually
	recordset, err := as.client.ScanAll(nil, as.namespace, as.set)
	if err != nil {
		return fmt.Errorf("failed to query blocks: %w", err)
	}

	blockKeyPrefix := blockKey + ":" + chainKey
	for res := range recordset.Results() {
		if res.Err != nil {
			return fmt.Errorf("error in query results: %w", res.Err)
		}

		// Check if this is a block for our chain
		keyStr := res.Record.Key.String()
		if !strings.Contains(keyStr, blockKeyPrefix) {
			continue
		}

		if _, err := as.client.Delete(as.policy, res.Record.Key); err != nil {
			return fmt.Errorf("failed to delete block: %w", err)
		}
	}

	// Delete CID mapping
	cidKey, err := aero.NewKey(as.namespace, as.set, cidMappingKey+":"+chainKey)
	if err != nil {
		return fmt.Errorf("failed to create CID key: %w", err)
	}

	// Ignore error if key doesn't exist
	as.client.Delete(as.policy, cidKey)

	return nil
}

// ListChains lists all chains in storage
func (as *AerospikeStorage) ListChains(ctx context.Context, options *storage.QueryOptions) ([]storage.ChainMetadata, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Query chain metadata - use scan and filter manually
	recordset, err := as.client.ScanAll(nil, as.namespace, as.set)
	if err != nil {
		return nil, fmt.Errorf("failed to query chains: %w", err)
	}

	var chains []storage.ChainMetadata
	for res := range recordset.Results() {
		if res.Err != nil {
			return nil, fmt.Errorf("error in query results: %w", res.Err)
		}

		// Check if this is a chain metadata record
		keyStr := res.Record.Key.String()
		if !strings.Contains(keyStr, chainMetadataKey) {
			continue
		}

		metadataJSON, ok := res.Record.Bins["metadata"].(string)
		if !ok {
			continue
		}

		var metadata storage.ChainMetadata
		if jsonErr := json.Unmarshal([]byte(metadataJSON), &metadata); jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", jsonErr)
		}

		chains = append(chains, metadata)
	}

	// Apply filters if options are provided
	if options != nil && len(options.Filters) > 0 {
		filteredChains := make([]storage.ChainMetadata, 0)
		for _, metadata := range chains {
			if applyFilters(metadata, options.Filters) {
				filteredChains = append(filteredChains, metadata)
			}
		}
		chains = filteredChains
	}

	// Apply sorting if options are provided
	if options != nil && options.SortBy != "" {
		sortChainMetadata(chains, options.SortBy, options.SortOrder)
	}

	// Apply limit and offset if options are provided
	if options != nil {
		// Apply offset
		if options.Offset > 0 && options.Offset < len(chains) {
			chains = chains[options.Offset:]
		}

		// Apply limit
		if options.Limit > 0 && options.Limit < len(chains) {
			chains = chains[:options.Limit]
		}
	}

	return chains, nil
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
func (as *AerospikeStorage) SaveBlock(ctx context.Context, chainType, repoID string, b *block.Block) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := as.getChainKey(chainType, repoID)

	// Check if chain exists
	metadataKey, err := aero.NewKey(as.namespace, as.set, chainMetadataKey+":"+chainKey)
	if err != nil {
		return fmt.Errorf("failed to create metadata key: %w", err)
	}

	record, err := as.client.Get(as.readPolicy, metadataKey)
	if err != nil {
		// Check if key not found
		if err == aero.ErrKeyNotFound {
			// Create chain metadata
			metadata := storage.ChainMetadata{
				ChainType:   chainType,
				RepoID:      repoID,
				BlockCount:  0,
				LastUpdated: time.Now(),
			}

			// Marshal metadata to JSON
			metadataBytes, jsonErr := json.Marshal(metadata)
			if jsonErr != nil {
				return fmt.Errorf("failed to marshal metadata: %w", jsonErr)
			}

			bins := aero.BinMap{
				"metadata": string(metadataBytes),
			}

			if err := as.client.Put(as.policy, metadataKey, bins); err != nil {
				return fmt.Errorf("failed to save metadata: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get metadata: %w", err)
		}
	}

	// Check if block already exists
	blockKey := as.getBlockKey(chainType, repoID, b.Hash)
	key, err := aero.NewKey(as.namespace, as.set, blockKey+":"+blockKey)
	if err != nil {
		return fmt.Errorf("failed to create block key: %w", err)
	}

	exists, err := as.client.Exists(as.readPolicy, key)
	if err != nil {
		return fmt.Errorf("failed to check if block exists: %w", err)
	}

	if exists {
		return storage.ErrAlreadyExists
	}

	// Save block
	if err := as.saveBlock(chainType, repoID, b); err != nil {
		return err
	}

	// Update chain metadata
	var metadata storage.ChainMetadata
	if record != nil {
		metadataJSON, ok := record.Bins["metadata"].(string)
		if !ok {
			return fmt.Errorf("invalid metadata format")
		}

		if jsonErr := json.Unmarshal([]byte(metadataJSON), &metadata); jsonErr != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", jsonErr)
		}
	} else {
		metadata = storage.ChainMetadata{
			ChainType:   chainType,
			RepoID:      repoID,
			BlockCount:  0,
			LastUpdated: time.Now(),
		}
	}

	metadata.BlockCount++
	metadata.LastUpdated = time.Now()

	// Marshal metadata to JSON
	metadataBytes, jsonErr := json.Marshal(metadata)
	if jsonErr != nil {
		return fmt.Errorf("failed to marshal metadata: %w", jsonErr)
	}

	bins := aero.BinMap{
		"metadata": string(metadataBytes),
	}

	if err := as.client.Put(as.policy, metadataKey, bins); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// LoadBlock loads a block from a chain
func (as *AerospikeStorage) LoadBlock(ctx context.Context, chainType, repoID string, blockHash string) (*block.Block, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return nil, storage.ErrStorageClosed
	}

	blockKey := as.getBlockKey(chainType, repoID, blockHash)
	key, err := aero.NewKey(as.namespace, as.set, blockKey+":"+blockKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create block key: %w", err)
	}

	record, err := as.client.Get(as.readPolicy, key)
	if err != nil {
		// Check if key not found
		if err == aero.ErrKeyNotFound {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	blockJSON, ok := record.Bins["block"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid block format")
	}

	var b block.Block
	if jsonErr := json.Unmarshal([]byte(blockJSON), &b); jsonErr != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", jsonErr)
	}

	return &b, nil
}

// LoadBlockByIndex loads a block from a chain by its index
func (as *AerospikeStorage) LoadBlockByIndex(ctx context.Context, chainType, repoID string, blockIndex uint64) (*block.Block, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Load all blocks and find the one with the matching index
	blocks, err := as.LoadChain(ctx, chainType, repoID)
	if err != nil {
		return nil, err
	}

	for _, b := range blocks {
		if b.Index == blockIndex {
			return b, nil
		}
	}

	return nil, storage.ErrNotFound
}

// DeleteBlock deletes a block from a chain
func (as *AerospikeStorage) DeleteBlock(ctx context.Context, chainType, repoID string, blockHash string) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := as.getChainKey(chainType, repoID)

	// Check if chain exists
	metadataKey, err := aero.NewKey(as.namespace, as.set, chainMetadataKey+":"+chainKey)
	if err != nil {
		return fmt.Errorf("failed to create metadata key: %w", err)
	}

	record, err := as.client.Get(as.readPolicy, metadataKey)
	if err != nil {
		// Check if key not found
		if err == aero.ErrKeyNotFound {
			return storage.ErrNotFound
		}
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Check if block exists
	blockKey := as.getBlockKey(chainType, repoID, blockHash)
	key, err := aero.NewKey(as.namespace, as.set, blockKey+":"+blockKey)
	if err != nil {
		return fmt.Errorf("failed to create block key: %w", err)
	}

	exists, err := as.client.Exists(as.readPolicy, key)
	if err != nil {
		return fmt.Errorf("failed to check if block exists: %w", err)
	}

	if !exists {
		return storage.ErrNotFound
	}

	// Delete block
	if _, err := as.client.Delete(as.policy, key); err != nil {
		return fmt.Errorf("failed to delete block: %w", err)
	}

	// Update chain metadata
	var metadata storage.ChainMetadata
	metadataJSON, ok := record.Bins["metadata"].(string)
	if !ok {
		return fmt.Errorf("invalid metadata format")
	}

	if jsonErr := json.Unmarshal([]byte(metadataJSON), &metadata); jsonErr != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", jsonErr)
	}

	metadata.BlockCount--
	metadata.LastUpdated = time.Now()

	// Marshal metadata to JSON
	updatedMetadataBytes, jsonErr := json.Marshal(metadata)
	if jsonErr != nil {
		return fmt.Errorf("failed to marshal metadata: %w", jsonErr)
	}

	bins := aero.BinMap{
		"metadata": string(updatedMetadataBytes),
	}

	if err := as.client.Put(as.policy, metadataKey, bins); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// ExportChain exports a chain to a byte array
func (as *AerospikeStorage) ExportChain(ctx context.Context, chainType, repoID string, format string) ([]byte, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Load chain
	blocks, err := as.LoadChain(ctx, chainType, repoID)
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

		// Get IPFS CID if it exists
		cid, err := as.GetIPFSCID(ctx, chainType, repoID)
		if err != nil {
			cid = ""
		}

		chain := exportedChain{
			ChainType: chainType,
			RepoID:    repoID,
			Blocks:    blocks,
			IPFSCID:   cid,
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
func (as *AerospikeStorage) ImportChain(ctx context.Context, chainType, repoID string, data []byte, format string) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	var blocks []*block.Block
	var cid string

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
		if jsonErr := json.Unmarshal(data, &chain); jsonErr != nil {
			return fmt.Errorf("failed to unmarshal chain data: %w", jsonErr)
		}

		blocks = chain.Blocks
		cid = chain.IPFSCID
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
	if err := as.SaveChain(ctx, chainType, repoID, blocks); err != nil {
		return err
	}

	// Set IPFS CID if provided
	if cid != "" {
		if err := as.SetIPFSCID(ctx, chainType, repoID, cid); err != nil {
			return fmt.Errorf("failed to set IPFS CID: %w", err)
		}
	}

	return nil
}

// BackupStorage creates a backup of the entire storage
func (as *AerospikeStorage) BackupStorage(ctx context.Context, backupPath string) error {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	// This would typically use Aerospike's backup tools
	// For now, we'll export all chains to JSON files
	return fmt.Errorf("not implemented: use Aerospike's backup tools instead")
}

// RestoreStorage restores the storage from a backup
func (as *AerospikeStorage) RestoreStorage(ctx context.Context, backupPath string) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	// This would typically use Aerospike's restore tools
	// For now, we'll just import all chains from JSON files
	return fmt.Errorf("not implemented: use Aerospike's restore tools instead")
}

// GetStorageInfo returns information about the storage
func (as *AerospikeStorage) GetStorageInfo(ctx context.Context) (map[string]interface{}, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Get node statistics
	node := as.client.GetNodes()[0]
	infoPolicy := aero.NewInfoPolicy()
	stats, err := node.RequestInfo(infoPolicy, "statistics")
	if err != nil {
		return nil, fmt.Errorf("failed to get node statistics: %w", err)
	}

	// Count chains and blocks using scan
	recordset, err := as.client.ScanAll(nil, as.namespace, as.set)
	if err != nil {
		return nil, fmt.Errorf("failed to scan records: %w", err)
	}

	var chainsCount, blocksCount int
	for res := range recordset.Results() {
		if res.Err != nil {
			return nil, fmt.Errorf("error in scan results: %w", res.Err)
		}

		keyStr := res.Record.Key.String()
		if strings.Contains(keyStr, chainMetadataKey) {
			chainsCount++
		} else if strings.Contains(keyStr, blockKey) {
			blocksCount++
		}
	}

	return map[string]interface{}{
		"type":         "aerospike",
		"host":         as.config.AerospikeHost,
		"port":         as.config.AerospikePort,
		"namespace":    as.namespace,
		"set":          as.set,
		"chains_count": chainsCount,
		"blocks_count": blocksCount,
		"is_open":      as.isOpen,
		"statistics":   stats,
	}, nil
}

// SearchBlocks searches for blocks across all chains
func (as *AerospikeStorage) SearchBlocks(ctx context.Context, query string, options *storage.QueryOptions) ([]*block.Block, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Query all blocks using scan
	recordset, err := as.client.ScanAll(nil, as.namespace, as.set)
	if err != nil {
		return nil, fmt.Errorf("failed to scan records: %w", err)
	}

	var results []*block.Block
	for res := range recordset.Results() {
		if res.Err != nil {
			return nil, fmt.Errorf("error in scan results: %w", res.Err)
		}

		// Check if this is a block record
		keyStr := res.Record.Key.String()
		if !strings.Contains(keyStr, blockKey) {
			continue
		}

		blockJSON, ok := res.Record.Bins["block"].(string)
		if !ok {
			continue
		}

		// Check if query is in block data
		if !strings.Contains(blockJSON, query) {
			continue
		}

		var b block.Block
		if jsonErr := json.Unmarshal([]byte(blockJSON), &b); jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal block: %w", jsonErr)
		}

		results = append(results, &b)
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
func (as *AerospikeStorage) GetIPFSCID(ctx context.Context, chainType, repoID string) (string, error) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if !as.isOpen {
		return "", storage.ErrStorageClosed
	}

	chainKey := as.getChainKey(chainType, repoID)
	cidKey, err := aero.NewKey(as.namespace, as.set, cidMappingKey+":"+chainKey)
	if err != nil {
		return "", fmt.Errorf("failed to create CID key: %w", err)
	}

	record, err := as.client.Get(as.readPolicy, cidKey)
	if err != nil {
		// Check if key not found
		if err == aero.ErrKeyNotFound {
			return "", storage.ErrNotFound
		}
		return "", fmt.Errorf("failed to get CID: %w", err)
	}

	cid, ok := record.Bins["cid"].(string)
	if !ok {
		return "", fmt.Errorf("invalid CID format")
	}

	return cid, nil
}

// SetIPFSCID sets the IPFS CID for a chain
func (as *AerospikeStorage) SetIPFSCID(ctx context.Context, chainType, repoID, cid string) error {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	if !as.isOpen {
		return storage.ErrStorageClosed
	}

	chainKey := as.getChainKey(chainType, repoID)
	cidKey, err := aero.NewKey(as.namespace, as.set, cidMappingKey+":"+chainKey)
	if err != nil {
		return fmt.Errorf("failed to create CID key: %w", err)
	}

	bins := aero.BinMap{
		"cid": cid,
	}

	if err := as.client.Put(as.policy, cidKey, bins); err != nil {
		return fmt.Errorf("failed to save CID: %w", err)
	}

	// Update chain metadata
	metadataKey, err := aero.NewKey(as.namespace, as.set, chainMetadataKey+":"+chainKey)
	if err != nil {
		return fmt.Errorf("failed to create metadata key: %w", err)
	}

	record, err := as.client.Get(as.readPolicy, metadataKey)
	// Only update if metadata exists and no error
	if err == nil && record != nil {
		var metadata storage.ChainMetadata
		metadataJSON, ok := record.Bins["metadata"].(string)
		if !ok {
			return fmt.Errorf("invalid metadata format")
		}

		if jsonErr := json.Unmarshal([]byte(metadataJSON), &metadata); jsonErr != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", jsonErr)
		}

		metadata.IPFSCID = cid
		metadata.LastUpdated = time.Now()

		// Marshal metadata to JSON
		updatedMetadataBytes, jsonErr := json.Marshal(metadata)
		if jsonErr != nil {
			return fmt.Errorf("failed to marshal metadata: %w", jsonErr)
		}

		bins := aero.BinMap{
			"metadata": string(updatedMetadataBytes),
		}

		if err := as.client.Put(as.policy, metadataKey, bins); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	return nil
}
