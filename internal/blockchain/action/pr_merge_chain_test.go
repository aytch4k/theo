package action

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewPRMergeChain(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	if chain.ChainType != "pr-merge" {
		t.Errorf("Expected chain type 'pr-merge', got '%s'", chain.ChainType)
	}

	if chain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", chain.RepoID)
	}

	if len(chain.Blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(chain.Blocks))
	}
}

func TestPRMergeChain_AddPRMerge(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	hash, err := chain.AddPRMerge(
		"PR-123",
		"abcdef1234567890",
		"user1",
		"merge",
		"Merge pull request #123",
	)

	if err != nil {
		t.Fatalf("Failed to add PR merge: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}

	// Verify block data
	var data PRMergeData
	err = json.Unmarshal(chain.Blocks[0].Data, &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal block data: %v", err)
	}

	if data.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", data.PRNumber)
	}

	if data.MergeCommitHash != "abcdef1234567890" {
		t.Errorf("Expected merge commit hash 'abcdef1234567890', got '%s'", data.MergeCommitHash)
	}

	if data.Merger != "user1" {
		t.Errorf("Expected merger 'user1', got '%s'", data.Merger)
	}

	if data.MergeMethod != "merge" {
		t.Errorf("Expected merge method 'merge', got '%s'", data.MergeMethod)
	}

	if data.MergeMessage != "Merge pull request #123" {
		t.Errorf("Expected merge message 'Merge pull request #123', got '%s'", data.MergeMessage)
	}

	// Verify timestamp is reasonable (within the last minute)
	now := time.Now().Unix()
	if data.Timestamp > now || data.Timestamp < now-60 {
		t.Errorf("Expected timestamp to be within the last minute, got %d (now: %d)", data.Timestamp, now)
	}
}

func TestPRMergeChain_GetPRMerges(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	// Add two PR merges
	_, err := chain.AddPRMerge("PR-123", "hash1", "user1", "merge", "Merge PR 123")
	if err != nil {
		t.Fatalf("Failed to add first PR merge: %v", err)
	}

	_, err = chain.AddPRMerge("PR-456", "hash2", "user2", "squash", "Merge PR 456")
	if err != nil {
		t.Fatalf("Failed to add second PR merge: %v", err)
	}

	// Get all PR merges
	merges, err := chain.GetPRMerges()
	if err != nil {
		t.Fatalf("Failed to get PR merges: %v", err)
	}

	if len(merges) != 2 {
		t.Errorf("Expected 2 PR merges, got %d", len(merges))
	}

	if merges[0].PRNumber != "PR-123" {
		t.Errorf("Expected first PR number 'PR-123', got '%s'", merges[0].PRNumber)
	}

	if merges[1].PRNumber != "PR-456" {
		t.Errorf("Expected second PR number 'PR-456', got '%s'", merges[1].PRNumber)
	}
}

func TestPRMergeChain_GetPRMergeByNumber(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	// Add two PR merges
	_, err := chain.AddPRMerge("PR-123", "hash1", "user1", "merge", "Merge PR 123")
	if err != nil {
		t.Fatalf("Failed to add first PR merge: %v", err)
	}

	_, err = chain.AddPRMerge("PR-456", "hash2", "user2", "squash", "Merge PR 456")
	if err != nil {
		t.Fatalf("Failed to add second PR merge: %v", err)
	}

	// Get PR merge by number
	pr, err := chain.GetPRMergeByNumber("PR-123")
	if err != nil {
		t.Fatalf("Failed to get PR merge by number: %v", err)
	}

	if pr.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", pr.PRNumber)
	}

	if pr.MergeCommitHash != "hash1" {
		t.Errorf("Expected merge commit hash 'hash1', got '%s'", pr.MergeCommitHash)
	}

	// Try to get non-existent PR
	_, err = chain.GetPRMergeByNumber("PR-999")
	if err != ErrPRMergeNotFound {
		t.Errorf("Expected error ErrPRMergeNotFound, got %v", err)
	}
}

func TestPRMergeChain_GetPRMergeByCommitHash(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	// Add two PR merges
	_, err := chain.AddPRMerge("PR-123", "hash1", "user1", "merge", "Merge PR 123")
	if err != nil {
		t.Fatalf("Failed to add first PR merge: %v", err)
	}

	_, err = chain.AddPRMerge("PR-456", "hash2", "user2", "squash", "Merge PR 456")
	if err != nil {
		t.Fatalf("Failed to add second PR merge: %v", err)
	}

	// Get PR merge by commit hash
	pr, err := chain.GetPRMergeByCommitHash("hash2")
	if err != nil {
		t.Fatalf("Failed to get PR merge by commit hash: %v", err)
	}

	if pr.PRNumber != "PR-456" {
		t.Errorf("Expected PR number 'PR-456', got '%s'", pr.PRNumber)
	}

	if pr.MergeMethod != "squash" {
		t.Errorf("Expected merge method 'squash', got '%s'", pr.MergeMethod)
	}

	// Try to get non-existent PR
	_, err = chain.GetPRMergeByCommitHash("nonexistent")
	if err != ErrPRMergeNotFound {
		t.Errorf("Expected error ErrPRMergeNotFound, got %v", err)
	}
}

func TestPRMergeChain_GetPRMergeByIndex(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	// Add two PR merges
	_, err := chain.AddPRMerge("PR-123", "hash1", "user1", "merge", "Merge PR 123")
	if err != nil {
		t.Fatalf("Failed to add first PR merge: %v", err)
	}

	_, err = chain.AddPRMerge("PR-456", "hash2", "user2", "squash", "Merge PR 456")
	if err != nil {
		t.Fatalf("Failed to add second PR merge: %v", err)
	}

	// Get PR merge by index
	pr, err := chain.GetPRMergeByIndex(0)
	if err != nil {
		t.Fatalf("Failed to get PR merge by index: %v", err)
	}

	if pr.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", pr.PRNumber)
	}

	// Try to get non-existent PR
	_, err = chain.GetPRMergeByIndex(999)
	if err != ErrBlockNotFound {
		t.Errorf("Expected error ErrBlockNotFound, got %v", err)
	}
}

func TestPRMergeChain_GetLatestPRMerge(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	// Try to get latest PR merge from empty chain
	_, err := chain.GetLatestPRMerge()
	if err != ErrEmptyChain {
		t.Errorf("Expected error ErrEmptyChain, got %v", err)
	}

	// Add two PR merges
	_, err = chain.AddPRMerge("PR-123", "hash1", "user1", "merge", "Merge PR 123")
	if err != nil {
		t.Fatalf("Failed to add first PR merge: %v", err)
	}

	_, err = chain.AddPRMerge("PR-456", "hash2", "user2", "squash", "Merge PR 456")
	if err != nil {
		t.Fatalf("Failed to add second PR merge: %v", err)
	}

	// Get latest PR merge
	pr, err := chain.GetLatestPRMerge()
	if err != nil {
		t.Fatalf("Failed to get latest PR merge: %v", err)
	}

	if pr.PRNumber != "PR-456" {
		t.Errorf("Expected PR number 'PR-456', got '%s'", pr.PRNumber)
	}

	if pr.MergeCommitHash != "hash2" {
		t.Errorf("Expected merge commit hash 'hash2', got '%s'", pr.MergeCommitHash)
	}
}

func TestPRMergeChain_Export(t *testing.T) {
	chain := NewPRMergeChain("test-repo")

	// Add a PR merge
	_, err := chain.AddPRMerge("PR-123", "hash1", "user1", "merge", "Merge PR 123")
	if err != nil {
		t.Fatalf("Failed to add PR merge: %v", err)
	}

	// Export chain
	data, err := chain.Export()
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Import chain
	newChain := NewPRMergeChain("")
	err = newChain.Import(data)
	if err != nil {
		t.Fatalf("Failed to import chain: %v", err)
	}

	// Verify imported chain
	if newChain.ChainType != "pr-merge" {
		t.Errorf("Expected chain type 'pr-merge', got '%s'", newChain.ChainType)
	}

	if newChain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", newChain.RepoID)
	}

	if len(newChain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(newChain.Blocks))
	}

	// Verify block data
	var prData PRMergeData
	err = json.Unmarshal(newChain.Blocks[0].Data, &prData)
	if err != nil {
		t.Fatalf("Failed to unmarshal block data: %v", err)
	}

	if prData.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", prData.PRNumber)
	}

	if prData.MergeCommitHash != "hash1" {
		t.Errorf("Expected merge commit hash 'hash1', got '%s'", prData.MergeCommitHash)
	}
}
