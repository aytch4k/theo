package storage

import (
	"context"
	"errors"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
)

// Common errors
var (
	ErrNotFound      = errors.New("item not found")
	ErrAlreadyExists = errors.New("item already exists")
	ErrInvalidKey    = errors.New("invalid key")
	ErrInvalidValue  = errors.New("invalid value")
	ErrStorageClosed = errors.New("storage is closed")
)

// StorageType represents the type of storage backend
type StorageType string

// Storage backend types
const (
	StorageTypeFile      StorageType = "file"
	StorageTypeSQLite    StorageType = "sqlite"
	StorageTypeIndexedDB StorageType = "indexeddb"
	StorageTypeIPFS      StorageType = "ipfs"
	StorageTypeAerospike StorageType = "aerospike"
)

// StorageConfig contains configuration options for storage backends
type StorageConfig struct {
	// Common options
	Type        StorageType `json:"type"`
	Path        string      `json:"path"`
	Compression bool        `json:"compression"`
	Encryption  bool        `json:"encryption"`

	// SQLite specific options
	ConnectionString string `json:"connection_string,omitempty"`

	// IPFS specific options
	IPFSHost     string `json:"ipfs_host,omitempty"`
	IPFSPort     int    `json:"ipfs_port,omitempty"`
	IPFSProtocol string `json:"ipfs_protocol,omitempty"`

	// IndexedDB specific options
	DBName     string   `json:"db_name,omitempty"`
	DBVersion  int      `json:"db_version,omitempty"`
	StoreNames []string `json:"store_names,omitempty"`

	// Aerospike specific options
	AerospikeHost      string `json:"aerospike_host,omitempty"`
	AerospikePort      int    `json:"aerospike_port,omitempty"`
	AerospikeNamespace string `json:"aerospike_namespace,omitempty"`
	AerospikeSet       string `json:"aerospike_set,omitempty"`
	AerospikeUsername  string `json:"aerospike_username,omitempty"`
	AerospikePassword  string `json:"aerospike_password,omitempty"`
}

// ChainMetadata contains metadata about a chain
type ChainMetadata struct {
	ChainType   string    `json:"chain_type"`
	RepoID      string    `json:"repo_id"`
	BlockCount  int       `json:"block_count"`
	LastUpdated time.Time `json:"last_updated"`
	IPFSCID     string    `json:"ipfs_cid,omitempty"`
}

// QueryOptions contains options for querying data
type QueryOptions struct {
	Limit          int      `json:"limit"`
	Offset         int      `json:"offset"`
	SortBy         string   `json:"sort_by"`
	SortOrder      string   `json:"sort_order"` // "asc" or "desc"
	Filters        []Filter `json:"filters"`
	UseBloomFilter bool     `json:"use_bloom_filter"`
}

// Filter represents a filter condition for queries
type Filter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // "eq", "neq", "gt", "gte", "lt", "lte", "contains", "startswith", "endswith"
	Value    interface{} `json:"value"`
}

// Storage defines the interface for all storage backends
type Storage interface {
	// Open initializes the storage backend
	Open(ctx context.Context) error

	// Close closes the storage backend
	Close() error

	// SaveChain saves a chain to storage
	SaveChain(ctx context.Context, chainType, repoID string, blocks []*block.Block) error

	// LoadChain loads a chain from storage
	LoadChain(ctx context.Context, chainType, repoID string) ([]*block.Block, error)

	// DeleteChain deletes a chain from storage
	DeleteChain(ctx context.Context, chainType, repoID string) error

	// ListChains lists all chains in storage
	ListChains(ctx context.Context, options *QueryOptions) ([]ChainMetadata, error)

	// SaveBlock saves a block to a chain
	SaveBlock(ctx context.Context, chainType, repoID string, block *block.Block) error

	// LoadBlock loads a block from a chain
	LoadBlock(ctx context.Context, chainType, repoID string, blockHash string) (*block.Block, error)

	// LoadBlockByIndex loads a block from a chain by its index
	LoadBlockByIndex(ctx context.Context, chainType, repoID string, blockIndex uint64) (*block.Block, error)

	// DeleteBlock deletes a block from a chain
	DeleteBlock(ctx context.Context, chainType, repoID string, blockHash string) error

	// ExportChain exports a chain to a byte array
	ExportChain(ctx context.Context, chainType, repoID string, format string) ([]byte, error)

	// ImportChain imports a chain from a byte array
	ImportChain(ctx context.Context, chainType, repoID string, data []byte, format string) error

	// BackupStorage creates a backup of the entire storage
	BackupStorage(ctx context.Context, backupPath string) error

	// RestoreStorage restores the storage from a backup
	RestoreStorage(ctx context.Context, backupPath string) error

	// GetStorageInfo returns information about the storage
	GetStorageInfo(ctx context.Context) (map[string]interface{}, error)

	// SearchBlocks searches for blocks across all chains
	SearchBlocks(ctx context.Context, query string, options *QueryOptions) ([]*block.Block, error)

	// GetIPFSCID returns the IPFS CID for a chain
	GetIPFSCID(ctx context.Context, chainType, repoID string) (string, error)

	// SetIPFSCID sets the IPFS CID for a chain
	SetIPFSCID(ctx context.Context, chainType, repoID, cid string) error
}

// StorageFactory creates a new storage backend based on the provided configuration
func NewStorage(config *StorageConfig) (Storage, error) {
	// This function should be implemented in a separate package to avoid import cycles
	// For now, return an error
	return nil, errors.New("storage factory not implemented yet")
}
