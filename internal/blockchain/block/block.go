package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Block represents a basic block in a blockchain with linked-JSON support
type Block struct {
	Context      string          `json:"@context"`
	ID           string          `json:"@id"`
	Index        uint64          `json:"index"`
	PreviousHash string          `json:"previousHash"`
	Timestamp    int64           `json:"timestamp"`
	Data         json.RawMessage `json:"data"`
	Hash         string          `json:"hash"`
	Signature    []byte          `json:"signature,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// Reference represents a reference to another block or resource
type Reference struct {
	ID string `json:"@id"`
}

// NewBlock creates a new block with the given parameters
func NewBlock(index uint64, previousHash string, data json.RawMessage, chainType, repoID string) (*Block, error) {
	if len(previousHash) == 0 && index > 0 {
		return nil, fmt.Errorf("previous hash is required for non-genesis blocks")
	}

	// Generate a unique ID for the block
	blockID := fmt.Sprintf("chain://%s/%s/block%s", chainType, repoID, uuid.New().String())

	// Create the block
	block := &Block{
		Context:      "https://devhub-git.org/contexts/block.jsonld",
		ID:           blockID,
		Index:        index,
		PreviousHash: previousHash,
		Timestamp:    time.Now().Unix(),
		Data:         data,
	}

	// Calculate the hash
	hash, err := block.CalculateHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}
	block.Hash = hash

	return block, nil
}

// CalculateHash calculates the hash of the block
func (b *Block) CalculateHash() (string, error) {
	// Create a copy of the block without the hash and signature
	blockCopy := &Block{
		Context:      b.Context,
		ID:           b.ID,
		Index:        b.Index,
		PreviousHash: b.PreviousHash,
		Timestamp:    b.Timestamp,
		Data:         b.Data,
		Metadata:     b.Metadata,
	}

	// Marshal the block to JSON
	blockJSON, err := json.Marshal(blockCopy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal block: %w", err)
	}

	// Calculate the SHA-256 hash
	hash := sha256.Sum256(blockJSON)
	return hex.EncodeToString(hash[:]), nil
}

// Verify verifies the integrity of the block
func (b *Block) Verify() (bool, error) {
	// Calculate the hash
	hash, err := b.CalculateHash()
	if err != nil {
		return false, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Compare with the stored hash
	return hash == b.Hash, nil
}

// Sign signs the block with the given private key
func (b *Block) Sign(privateKey []byte, signFunc func(data, privateKey []byte) ([]byte, error)) error {
	// Calculate the hash if not already calculated
	if b.Hash == "" {
		hash, err := b.CalculateHash()
		if err != nil {
			return fmt.Errorf("failed to calculate hash: %w", err)
		}
		b.Hash = hash
	}

	// Sign the hash
	signature, err := signFunc([]byte(b.Hash), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign block: %w", err)
	}

	b.Signature = signature
	return nil
}

// VerifySignature verifies the signature of the block
func (b *Block) VerifySignature(publicKey []byte, verifyFunc func(data, signature, publicKey []byte) (bool, error)) (bool, error) {
	if len(b.Signature) == 0 {
		return false, fmt.Errorf("block is not signed")
	}

	// Verify the signature
	return verifyFunc([]byte(b.Hash), b.Signature, publicKey)
}

// MarshalJSON marshals the block to JSON
func (b *Block) MarshalJSON() ([]byte, error) {
	type BlockAlias Block
	return json.Marshal(&struct {
		*BlockAlias
	}{
		BlockAlias: (*BlockAlias)(b),
	})
}

// UnmarshalJSON unmarshals the block from JSON
func (b *Block) UnmarshalJSON(data []byte) error {
	type BlockAlias Block
	aux := &struct {
		*BlockAlias
	}{
		BlockAlias: (*BlockAlias)(b),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	return nil
}

// ToJSONL converts the block to JSONL format
func (b *Block) ToJSONL() (string, error) {
	blockJSON, err := json.Marshal(b)
	if err != nil {
		return "", fmt.Errorf("failed to marshal block: %w", err)
	}
	return string(blockJSON), nil
}

// FromJSONL converts JSONL to a block
func FromJSONL(jsonl string) (*Block, error) {
	var block Block
	if err := json.Unmarshal([]byte(jsonl), &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}
	return &block, nil
}
