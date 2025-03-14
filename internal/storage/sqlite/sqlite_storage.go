package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
)

// SQLiteStorage implements the Storage interface using SQLite
type SQLiteStorage struct {
	config     *storage.StorageConfig
	db         *sql.DB
	mutex      sync.RWMutex
	isOpen     bool
	cidMapping map[string]string // Maps chainType+repoID to IPFS CID
}

// NewSQLiteStorage creates a new SQLite storage backend
func NewSQLiteStorage(config *storage.StorageConfig) (*SQLiteStorage, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Path == "" && config.ConnectionString == "" {
		return nil, fmt.Errorf("either path or connection string must be provided")
	}

	return &SQLiteStorage{
		config:     config,
		isOpen:     false,
		cidMapping: make(map[string]string),
	}, nil
}

// Open initializes the SQLite storage backend
func (ss *SQLiteStorage) Open(ctx context.Context) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if ss.isOpen {
		return nil
	}

	// Determine connection string
	connectionString := ss.config.ConnectionString
	if connectionString == "" {
		// Create directory if it doesn't exist
		dbDir := filepath.Dir(ss.config.Path)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
		connectionString = ss.config.Path
	}

	// Open database
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	if err := ss.createTables(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Load CID mapping
	if err := ss.loadCIDMapping(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to load CID mapping: %w", err)
	}

	ss.db = db
	ss.isOpen = true
	return nil
}

// createTables creates the necessary tables in the database
func (ss *SQLiteStorage) createTables(db *sql.DB) error {
	// Create chains table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chain_type TEXT NOT NULL,
			repo_id TEXT NOT NULL,
			block_count INTEGER NOT NULL DEFAULT 0,
			last_updated TIMESTAMP NOT NULL,
			ipfs_cid TEXT,
			UNIQUE(chain_type, repo_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create chains table: %w", err)
	}

	// Create blocks table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chain_type TEXT NOT NULL,
			repo_id TEXT NOT NULL,
			block_hash TEXT NOT NULL,
			block_index INTEGER NOT NULL,
			previous_hash TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			data TEXT NOT NULL,
			signature BLOB,
			metadata TEXT,
			context TEXT NOT NULL,
			block_id TEXT NOT NULL,
			UNIQUE(chain_type, repo_id, block_hash),
			FOREIGN KEY(chain_type, repo_id) REFERENCES chains(chain_type, repo_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create blocks table: %w", err)
	}

	// Create CID mapping table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cid_mapping (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chain_key TEXT NOT NULL,
			ipfs_cid TEXT NOT NULL,
			UNIQUE(chain_key)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create CID mapping table: %w", err)
	}

	// Create indexes
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_blocks_chain ON blocks(chain_type, repo_id);
		CREATE INDEX IF NOT EXISTS idx_blocks_hash ON blocks(block_hash);
		CREATE INDEX IF NOT EXISTS idx_blocks_index ON blocks(block_index);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// loadCIDMapping loads the CID mapping from the database
func (ss *SQLiteStorage) loadCIDMapping(db *sql.DB) error {
	rows, err := db.Query("SELECT chain_key, ipfs_cid FROM cid_mapping")
	if err != nil {
		return fmt.Errorf("failed to query CID mapping: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var chainKey, ipfsCID string
		if err := rows.Scan(&chainKey, &ipfsCID); err != nil {
			return fmt.Errorf("failed to scan CID mapping row: %w", err)
		}
		ss.cidMapping[chainKey] = ipfsCID
	}

	return rows.Err()
}

// Close closes the SQLite storage backend
func (ss *SQLiteStorage) Close() error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
		return nil
	}

	// Save CID mapping
	if err := ss.saveCIDMapping(ss.db); err != nil {
		return fmt.Errorf("failed to save CID mapping: %w", err)
	}

	// Close database
	if err := ss.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	ss.isOpen = false
	return nil
}

// saveCIDMapping saves the CID mapping to the database
func (ss *SQLiteStorage) saveCIDMapping(db *sql.DB) error {
	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing mapping
	_, err = tx.Exec("DELETE FROM cid_mapping")
	if err != nil {
		return fmt.Errorf("failed to clear CID mapping: %w", err)
	}

	// Insert new mapping
	stmt, err := tx.Prepare("INSERT INTO cid_mapping (chain_key, ipfs_cid) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare CID mapping statement: %w", err)
	}
	defer stmt.Close()

	for chainKey, ipfsCID := range ss.cidMapping {
		_, err := stmt.Exec(chainKey, ipfsCID)
		if err != nil {
			return fmt.Errorf("failed to insert CID mapping: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SaveChain saves a chain to storage
func (ss *SQLiteStorage) SaveChain(ctx context.Context, chainType, repoID string, blocks []*block.Block) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
		return storage.ErrStorageClosed
	}

	// Begin transaction
	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert chain
	_, err = tx.Exec(`
		INSERT INTO chains (chain_type, repo_id, block_count, last_updated, ipfs_cid)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(chain_type, repo_id) DO UPDATE SET
			block_count = ?,
			last_updated = ?,
			ipfs_cid = ?
	`,
		chainType, repoID, len(blocks), time.Now(), ss.cidMapping[chainType+repoID],
		len(blocks), time.Now(), ss.cidMapping[chainType+repoID],
	)
	if err != nil {
		return fmt.Errorf("failed to upsert chain: %w", err)
	}

	// Delete existing blocks
	_, err = tx.Exec("DELETE FROM blocks WHERE chain_type = ? AND repo_id = ?", chainType, repoID)
	if err != nil {
		return fmt.Errorf("failed to delete existing blocks: %w", err)
	}

	// Insert blocks
	stmt, err := tx.Prepare(`
		INSERT INTO blocks (
			chain_type, repo_id, block_hash, block_index, previous_hash,
			timestamp, data, signature, metadata, context, block_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare block statement: %w", err)
	}
	defer stmt.Close()

	for _, b := range blocks {
		data, err := json.Marshal(b.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal block data: %w", err)
		}

		metadata, err := json.Marshal(b.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal block metadata: %w", err)
		}

		_, err = stmt.Exec(
			chainType, repoID, b.Hash, b.Index, b.PreviousHash,
			b.Timestamp, string(data), b.Signature, string(metadata), b.Context, b.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert block: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadChain loads a chain from storage
func (ss *SQLiteStorage) LoadChain(ctx context.Context, chainType, repoID string) ([]*block.Block, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Check if chain exists
	var count int
	err := ss.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chains WHERE chain_type = ? AND repo_id = ?", chainType, repoID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check chain existence: %w", err)
	}
	if count == 0 {
		return nil, storage.ErrNotFound
	}

	// Query blocks
	rows, err := ss.db.QueryContext(ctx, `
		SELECT block_hash, block_index, previous_hash, timestamp, data, signature, metadata, context, block_id
		FROM blocks
		WHERE chain_type = ? AND repo_id = ?
		ORDER BY block_index ASC
	`, chainType, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	var blocks []*block.Block
	for rows.Next() {
		var b block.Block
		var dataStr, metadataStr string
		var signature []byte

		err := rows.Scan(
			&b.Hash, &b.Index, &b.PreviousHash, &b.Timestamp,
			&dataStr, &signature, &metadataStr, &b.Context, &b.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block row: %w", err)
		}

		// Parse data
		if err := json.Unmarshal([]byte(dataStr), &b.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
		}

		// Parse metadata
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &b.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal block metadata: %w", err)
			}
		}

		// Set signature
		b.Signature = signature

		blocks = append(blocks, &b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating block rows: %w", err)
	}

	return blocks, nil
}

// DeleteChain deletes a chain from storage
func (ss *SQLiteStorage) DeleteChain(ctx context.Context, chainType, repoID string) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
		return storage.ErrStorageClosed
	}

	// Begin transaction
	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete blocks
	_, err = tx.Exec("DELETE FROM blocks WHERE chain_type = ? AND repo_id = ?", chainType, repoID)
	if err != nil {
		return fmt.Errorf("failed to delete blocks: %w", err)
	}

	// Delete chain
	result, err := tx.Exec("DELETE FROM chains WHERE chain_type = ? AND repo_id = ?", chainType, repoID)
	if err != nil {
		return fmt.Errorf("failed to delete chain: %w", err)
	}

	// Check if chain existed
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	// Remove from CID mapping
	delete(ss.cidMapping, chainType+repoID)

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListChains lists all chains in storage
func (ss *SQLiteStorage) ListChains(ctx context.Context, options *storage.QueryOptions) ([]storage.ChainMetadata, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Build query
	query := "SELECT chain_type, repo_id, block_count, last_updated, ipfs_cid FROM chains"
	var args []interface{}

	// Apply filters if options are provided
	if options != nil && len(options.Filters) > 0 {
		whereClause, filterArgs := ss.buildWhereClause(options.Filters)
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = append(args, filterArgs...)
		}
	}

	// Apply sorting if options are provided
	if options != nil && options.SortBy != "" {
		query += " ORDER BY " + options.SortBy
		if options.SortOrder != "" {
			query += " " + options.SortOrder
		}
	} else {
		query += " ORDER BY chain_type, repo_id"
	}

	// Apply limit and offset if options are provided
	if options != nil {
		if options.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", options.Limit)
		}
		if options.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", options.Offset)
		}
	}

	// Execute query
	rows, err := ss.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query chains: %w", err)
	}
	defer rows.Close()

	var chains []storage.ChainMetadata
	for rows.Next() {
		var metadata storage.ChainMetadata
		var lastUpdated string
		var ipfsCID sql.NullString

		err := rows.Scan(
			&metadata.ChainType, &metadata.RepoID, &metadata.BlockCount,
			&lastUpdated, &ipfsCID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chain row: %w", err)
		}

		// Parse last updated
		metadata.LastUpdated, err = time.Parse(time.RFC3339, lastUpdated)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last updated: %w", err)
		}

		// Set IPFS CID
		if ipfsCID.Valid {
			metadata.IPFSCID = ipfsCID.String
		}

		chains = append(chains, metadata)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chain rows: %w", err)
	}

	return chains, nil
}

// buildWhereClause builds a WHERE clause from filters
func (ss *SQLiteStorage) buildWhereClause(filters []storage.Filter) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	for _, filter := range filters {
		switch filter.Field {
		case "chain_type", "repo_id", "block_count", "last_updated", "ipfs_cid":
			clause, arg := ss.buildFilterClause(filter)
			if clause != "" {
				clauses = append(clauses, clause)
				args = append(args, arg...)
			}
		}
	}

	return strings.Join(clauses, " AND "), args
}

// buildFilterClause builds a filter clause
func (ss *SQLiteStorage) buildFilterClause(filter storage.Filter) (string, []interface{}) {
	switch filter.Operator {
	case "eq":
		return fmt.Sprintf("%s = ?", filter.Field), []interface{}{filter.Value}
	case "neq":
		return fmt.Sprintf("%s != ?", filter.Field), []interface{}{filter.Value}
	case "gt":
		return fmt.Sprintf("%s > ?", filter.Field), []interface{}{filter.Value}
	case "gte":
		return fmt.Sprintf("%s >= ?", filter.Field), []interface{}{filter.Value}
	case "lt":
		return fmt.Sprintf("%s < ?", filter.Field), []interface{}{filter.Value}
	case "lte":
		return fmt.Sprintf("%s <= ?", filter.Field), []interface{}{filter.Value}
	case "contains":
		return fmt.Sprintf("%s LIKE ?", filter.Field), []interface{}{"%" + filter.Value.(string) + "%"}
	case "startswith":
		return fmt.Sprintf("%s LIKE ?", filter.Field), []interface{}{filter.Value.(string) + "%"}
	case "endswith":
		return fmt.Sprintf("%s LIKE ?", filter.Field), []interface{}{"%" + filter.Value.(string)}
	default:
		return "", nil
	}
}

// SaveBlock saves a block to a chain
func (ss *SQLiteStorage) SaveBlock(ctx context.Context, chainType, repoID string, b *block.Block) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
		return storage.ErrStorageClosed
	}

	// Begin transaction
	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if chain exists
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM chains WHERE chain_type = ? AND repo_id = ?", chainType, repoID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check chain existence: %w", err)
	}
	if count == 0 {
		// Create chain
		_, err = tx.Exec(`
			INSERT INTO chains (chain_type, repo_id, block_count, last_updated, ipfs_cid)
			VALUES (?, ?, 0, ?, ?)
		`, chainType, repoID, time.Now(), ss.cidMapping[chainType+repoID])
		if err != nil {
			return fmt.Errorf("failed to create chain: %w", err)
		}
	}

	// Check if block already exists
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM blocks WHERE chain_type = ? AND repo_id = ? AND block_hash = ?", chainType, repoID, b.Hash).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check block existence: %w", err)
	}
	if count > 0 {
		return storage.ErrAlreadyExists
	}

	// Insert block
	data, err := json.Marshal(b.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal block data: %w", err)
	}

	metadata, err := json.Marshal(b.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal block metadata: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO blocks (
			chain_type, repo_id, block_hash, block_index, previous_hash,
			timestamp, data, signature, metadata, context, block_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		chainType, repoID, b.Hash, b.Index, b.PreviousHash,
		b.Timestamp, string(data), b.Signature, string(metadata), b.Context, b.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert block: %w", err)
	}

	// Update chain
	_, err = tx.Exec(`
		UPDATE chains
		SET block_count = block_count + 1, last_updated = ?
		WHERE chain_type = ? AND repo_id = ?
	`, time.Now(), chainType, repoID)
	if err != nil {
		return fmt.Errorf("failed to update chain: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadBlock loads a block from a chain
func (ss *SQLiteStorage) LoadBlock(ctx context.Context, chainType, repoID string, blockHash string) (*block.Block, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Query block
	row := ss.db.QueryRowContext(ctx, `
		SELECT block_hash, block_index, previous_hash, timestamp, data, signature, metadata, context, block_id
		FROM blocks
		WHERE chain_type = ? AND repo_id = ? AND block_hash = ?
	`, chainType, repoID, blockHash)

	var b block.Block
	var dataStr, metadataStr string
	var signature []byte

	err := row.Scan(
		&b.Hash, &b.Index, &b.PreviousHash, &b.Timestamp,
		&dataStr, &signature, &metadataStr, &b.Context, &b.ID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan block row: %w", err)
	}

	// Parse data
	if err := json.Unmarshal([]byte(dataStr), &b.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
	}

	// Parse metadata
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &b.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block metadata: %w", err)
		}
	}

	// Set signature
	b.Signature = signature

	return &b, nil
}

// LoadBlockByIndex loads a block from a chain by its index
func (ss *SQLiteStorage) LoadBlockByIndex(ctx context.Context, chainType, repoID string, blockIndex uint64) (*block.Block, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Query block
	row := ss.db.QueryRowContext(ctx, `
		SELECT block_hash, block_index, previous_hash, timestamp, data, signature, metadata, context, block_id
		FROM blocks
		WHERE chain_type = ? AND repo_id = ? AND block_index = ?
	`, chainType, repoID, blockIndex)

	var b block.Block
	var dataStr, metadataStr string
	var signature []byte

	err := row.Scan(
		&b.Hash, &b.Index, &b.PreviousHash, &b.Timestamp,
		&dataStr, &signature, &metadataStr, &b.Context, &b.ID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan block row: %w", err)
	}

	// Parse data
	if err := json.Unmarshal([]byte(dataStr), &b.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
	}

	// Parse metadata
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &b.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block metadata: %w", err)
		}
	}

	// Set signature
	b.Signature = signature

	return &b, nil
}

// DeleteBlock deletes a block from a chain
func (ss *SQLiteStorage) DeleteBlock(ctx context.Context, chainType, repoID string, blockHash string) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
		return storage.ErrStorageClosed
	}

	// Begin transaction
	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete block
	result, err := tx.Exec("DELETE FROM blocks WHERE chain_type = ? AND repo_id = ? AND block_hash = ?", chainType, repoID, blockHash)
	if err != nil {
		return fmt.Errorf("failed to delete block: %w", err)
	}

	// Check if block existed
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	// Update chain
	_, err = tx.Exec(`
		UPDATE chains
		SET block_count = block_count - 1, last_updated = ?
		WHERE chain_type = ? AND repo_id = ?
	`, time.Now(), chainType, repoID)
	if err != nil {
		return fmt.Errorf("failed to update chain: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExportChain exports a chain to a byte array
func (ss *SQLiteStorage) ExportChain(ctx context.Context, chainType, repoID string, format string) ([]byte, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Load chain
	blocks, err := ss.LoadChain(ctx, chainType, repoID)
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
			IPFSCID:   ss.cidMapping[chainType+repoID],
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
func (ss *SQLiteStorage) ImportChain(ctx context.Context, chainType, repoID string, data []byte, format string) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
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
			ss.cidMapping[chainType+repoID] = chain.IPFSCID
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
	return ss.SaveChain(ctx, chainType, repoID, blocks)
}

// BackupStorage creates a backup of the entire storage
func (ss *SQLiteStorage) BackupStorage(ctx context.Context, backupPath string) error {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return storage.ErrStorageClosed
	}

	// Create backup directory
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup database
	backupDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}
	defer backupDB.Close()

	// Create tables in backup database
	if err := ss.createTables(backupDB); err != nil {
		return fmt.Errorf("failed to create tables in backup database: %w", err)
	}

	// Begin transaction
	tx, err := backupDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Copy chains
	rows, err := ss.db.QueryContext(ctx, "SELECT chain_type, repo_id, block_count, last_updated, ipfs_cid FROM chains")
	if err != nil {
		return fmt.Errorf("failed to query chains: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare("INSERT INTO chains (chain_type, repo_id, block_count, last_updated, ipfs_cid) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare chain statement: %w", err)
	}
	defer stmt.Close()

	for rows.Next() {
		var chainType, repoID, lastUpdated string
		var blockCount int
		var ipfsCID sql.NullString

		if err := rows.Scan(&chainType, &repoID, &blockCount, &lastUpdated, &ipfsCID); err != nil {
			return fmt.Errorf("failed to scan chain row: %w", err)
		}

		_, err := stmt.Exec(chainType, repoID, blockCount, lastUpdated, ipfsCID)
		if err != nil {
			return fmt.Errorf("failed to insert chain: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating chain rows: %w", err)
	}

	// Copy blocks
	rows, err = ss.db.QueryContext(ctx, `
		SELECT chain_type, repo_id, block_hash, block_index, previous_hash,
			timestamp, data, signature, metadata, context, block_id
		FROM blocks
	`)
	if err != nil {
		return fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	stmt, err = tx.Prepare(`
		INSERT INTO blocks (
			chain_type, repo_id, block_hash, block_index, previous_hash,
			timestamp, data, signature, metadata, context, block_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare block statement: %w", err)
	}
	defer stmt.Close()

	for rows.Next() {
		var chainType, repoID, blockHash, previousHash, data, metadata, context, blockID string
		var blockIndex, timestamp int64
		var signature []byte

		if err := rows.Scan(
			&chainType, &repoID, &blockHash, &blockIndex, &previousHash,
			&timestamp, &data, &signature, &metadata, &context, &blockID,
		); err != nil {
			return fmt.Errorf("failed to scan block row: %w", err)
		}

		_, err := stmt.Exec(
			chainType, repoID, blockHash, blockIndex, previousHash,
			timestamp, data, signature, metadata, context, blockID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert block: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating block rows: %w", err)
	}

	// Copy CID mapping
	for chainKey, ipfsCID := range ss.cidMapping {
		_, err := tx.Exec("INSERT INTO cid_mapping (chain_key, ipfs_cid) VALUES (?, ?)", chainKey, ipfsCID)
		if err != nil {
			return fmt.Errorf("failed to insert CID mapping: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RestoreStorage restores the storage from a backup
func (ss *SQLiteStorage) RestoreStorage(ctx context.Context, backupPath string) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
		return storage.ErrStorageClosed
	}

	// Open backup database
	backupDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}
	defer backupDB.Close()

	// Begin transaction
	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear current database
	_, err = tx.Exec("DELETE FROM blocks")
	if err != nil {
		return fmt.Errorf("failed to clear blocks: %w", err)
	}

	_, err = tx.Exec("DELETE FROM chains")
	if err != nil {
		return fmt.Errorf("failed to clear chains: %w", err)
	}

	_, err = tx.Exec("DELETE FROM cid_mapping")
	if err != nil {
		return fmt.Errorf("failed to clear CID mapping: %w", err)
	}

	// Copy chains
	rows, err := backupDB.QueryContext(ctx, "SELECT chain_type, repo_id, block_count, last_updated, ipfs_cid FROM chains")
	if err != nil {
		return fmt.Errorf("failed to query chains: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare("INSERT INTO chains (chain_type, repo_id, block_count, last_updated, ipfs_cid) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare chain statement: %w", err)
	}
	defer stmt.Close()

	for rows.Next() {
		var chainType, repoID, lastUpdated string
		var blockCount int
		var ipfsCID sql.NullString

		if err := rows.Scan(&chainType, &repoID, &blockCount, &lastUpdated, &ipfsCID); err != nil {
			return fmt.Errorf("failed to scan chain row: %w", err)
		}

		_, err := stmt.Exec(chainType, repoID, blockCount, lastUpdated, ipfsCID)
		if err != nil {
			return fmt.Errorf("failed to insert chain: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating chain rows: %w", err)
	}

	// Copy blocks
	rows, err = backupDB.QueryContext(ctx, `
		SELECT chain_type, repo_id, block_hash, block_index, previous_hash,
			timestamp, data, signature, metadata, context, block_id
		FROM blocks
	`)
	if err != nil {
		return fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	stmt, err = tx.Prepare(`
		INSERT INTO blocks (
			chain_type, repo_id, block_hash, block_index, previous_hash,
			timestamp, data, signature, metadata, context, block_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare block statement: %w", err)
	}
	defer stmt.Close()

	for rows.Next() {
		var chainType, repoID, blockHash, previousHash, data, metadata, context, blockID string
		var blockIndex, timestamp int64
		var signature []byte

		if err := rows.Scan(
			&chainType, &repoID, &blockHash, &blockIndex, &previousHash,
			&timestamp, &data, &signature, &metadata, &context, &blockID,
		); err != nil {
			return fmt.Errorf("failed to scan block row: %w", err)
		}

		_, err := stmt.Exec(
			chainType, repoID, blockHash, blockIndex, previousHash,
			timestamp, data, signature, metadata, context, blockID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert block: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating block rows: %w", err)
	}

	// Copy CID mapping
	rows, err = backupDB.QueryContext(ctx, "SELECT chain_key, ipfs_cid FROM cid_mapping")
	if err != nil {
		return fmt.Errorf("failed to query CID mapping: %w", err)
	}
	defer rows.Close()

	// Clear current CID mapping
	ss.cidMapping = make(map[string]string)

	for rows.Next() {
		var chainKey, ipfsCID string
		if err := rows.Scan(&chainKey, &ipfsCID); err != nil {
			return fmt.Errorf("failed to scan CID mapping row: %w", err)
		}

		ss.cidMapping[chainKey] = ipfsCID

		_, err := tx.Exec("INSERT INTO cid_mapping (chain_key, ipfs_cid) VALUES (?, ?)", chainKey, ipfsCID)
		if err != nil {
			return fmt.Errorf("failed to insert CID mapping: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating CID mapping rows: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetStorageInfo returns information about the storage
func (ss *SQLiteStorage) GetStorageInfo(ctx context.Context) (map[string]interface{}, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Get chains count
	var chainsCount int
	err := ss.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chains").Scan(&chainsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get chains count: %w", err)
	}

	// Get blocks count
	var blocksCount int
	err = ss.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blocks").Scan(&blocksCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocks count: %w", err)
	}

	// Get database size
	var databaseSize int64
	err = ss.db.QueryRowContext(ctx, "SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&databaseSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}

	return map[string]interface{}{
		"type":         "sqlite",
		"path":         ss.config.Path,
		"chains_count": chainsCount,
		"blocks_count": blocksCount,
		"total_size":   databaseSize,
		"is_open":      ss.isOpen,
	}, nil
}

// SearchBlocks searches for blocks across all chains
func (ss *SQLiteStorage) SearchBlocks(ctx context.Context, query string, options *storage.QueryOptions) ([]*block.Block, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return nil, storage.ErrStorageClosed
	}

	// Build query
	sqlQuery := `
		SELECT b.chain_type, b.repo_id, b.block_hash, b.block_index, b.previous_hash,
			b.timestamp, b.data, b.signature, b.metadata, b.context, b.block_id
		FROM blocks b
		WHERE b.data LIKE ? OR b.metadata LIKE ? OR b.block_hash LIKE ? OR b.block_id LIKE ?
	`
	args := []interface{}{
		"%" + query + "%",
		"%" + query + "%",
		"%" + query + "%",
		"%" + query + "%",
	}

	// Apply filters if options are provided
	if options != nil && len(options.Filters) > 0 {
		whereClause, filterArgs := ss.buildBlockWhereClause(options.Filters)
		if whereClause != "" {
			sqlQuery += " AND " + whereClause
			args = append(args, filterArgs...)
		}
	}

	// Apply sorting if options are provided
	if options != nil && options.SortBy != "" {
		sqlQuery += " ORDER BY b." + options.SortBy
		if options.SortOrder != "" {
			sqlQuery += " " + options.SortOrder
		}
	} else {
		sqlQuery += " ORDER BY b.chain_type, b.repo_id, b.block_index"
	}

	// Apply limit and offset if options are provided
	if options != nil {
		if options.Limit > 0 {
			sqlQuery += fmt.Sprintf(" LIMIT %d", options.Limit)
		}
		if options.Offset > 0 {
			sqlQuery += fmt.Sprintf(" OFFSET %d", options.Offset)
		}
	}

	// Execute query
	rows, err := ss.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	var blocks []*block.Block
	for rows.Next() {
		var chainType, repoID, blockHash, previousHash, dataStr, metadataStr, context, blockID string
		var blockIndex, timestamp int64
		var signature []byte

		err := rows.Scan(
			&chainType, &repoID, &blockHash, &blockIndex, &previousHash,
			&timestamp, &dataStr, &signature, &metadataStr, &context, &blockID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block row: %w", err)
		}

		var b block.Block
		b.Hash = blockHash
		b.Index = uint64(blockIndex)
		b.PreviousHash = previousHash
		b.Timestamp = timestamp
		b.Context = context
		b.ID = blockID
		b.Signature = signature

		// Parse data
		if err := json.Unmarshal([]byte(dataStr), &b.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
		}

		// Parse metadata
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &b.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal block metadata: %w", err)
			}
		}

		blocks = append(blocks, &b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating block rows: %w", err)
	}

	return blocks, nil
}

// buildBlockWhereClause builds a WHERE clause for blocks from filters
func (ss *SQLiteStorage) buildBlockWhereClause(filters []storage.Filter) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	for _, filter := range filters {
		switch filter.Field {
		case "chain_type", "repo_id", "block_hash", "block_index", "previous_hash", "timestamp", "block_id":
			clause, arg := ss.buildFilterClause(filter)
			if clause != "" {
				clauses = append(clauses, "b."+clause)
				args = append(args, arg...)
			}
		}
	}

	return strings.Join(clauses, " AND "), args
}

// GetIPFSCID returns the IPFS CID for a chain
func (ss *SQLiteStorage) GetIPFSCID(ctx context.Context, chainType, repoID string) (string, error) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if !ss.isOpen {
		return "", storage.ErrStorageClosed
	}

	cid, ok := ss.cidMapping[chainType+repoID]
	if !ok {
		return "", storage.ErrNotFound
	}

	return cid, nil
}

// SetIPFSCID sets the IPFS CID for a chain
func (ss *SQLiteStorage) SetIPFSCID(ctx context.Context, chainType, repoID, cid string) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isOpen {
		return storage.ErrStorageClosed
	}

	// Update CID mapping
	ss.cidMapping[chainType+repoID] = cid

	// Update chain
	_, err := ss.db.ExecContext(ctx, `
		UPDATE chains
		SET ipfs_cid = ?, last_updated = ?
		WHERE chain_type = ? AND repo_id = ?
	`, cid, time.Now(), chainType, repoID)
	if err != nil {
		return fmt.Errorf("failed to update chain: %w", err)
	}

	// Update CID mapping table
	_, err = ss.db.ExecContext(ctx, `
		INSERT INTO cid_mapping (chain_key, ipfs_cid)
		VALUES (?, ?)
		ON CONFLICT(chain_key) DO UPDATE SET
			ipfs_cid = ?
	`, chainType+repoID, cid, cid)
	if err != nil {
		return fmt.Errorf("failed to update CID mapping: %w", err)
	}

	return nil
}
