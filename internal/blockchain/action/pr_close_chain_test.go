package action

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewPRCloseChain(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	if chain.ChainType != "pr-close" {
		t.Errorf("Expected chain type 'pr-close', got '%s'", chain.ChainType)
	}

	if chain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", chain.RepoID)
	}

	if len(chain.Blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(chain.Blocks))
	}
}

func TestPRCloseChain_AddPRClose(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	hash, err := chain.AddPRClose(
		"PR-123",
		"user1",
		"abandoned",
		"Closing this PR as it's no longer needed",
	)

	if err != nil {
		t.Fatalf("Failed to add PR close: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}

	// Verify block data
	var data PRCloseData
	err = json.Unmarshal(chain.Blocks[0].Data, &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal block data: %v", err)
	}

	if data.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", data.PRNumber)
	}

	if data.Closer != "user1" {
		t.Errorf("Expected closer 'user1', got '%s'", data.Closer)
	}

	if data.CloseReason != "abandoned" {
		t.Errorf("Expected close reason 'abandoned', got '%s'", data.CloseReason)
	}

	if data.Comment != "Closing this PR as it's no longer needed" {
		t.Errorf("Expected comment 'Closing this PR as it's no longer needed', got '%s'", data.Comment)
	}

	// Verify timestamp is reasonable (within the last minute)
	now := time.Now().Unix()
	if data.Timestamp > now || data.Timestamp < now-60 {
		t.Errorf("Expected timestamp to be within the last minute, got %d (now: %d)", data.Timestamp, now)
	}
}

func TestPRCloseChain_GetPRCloses(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	// Add two PR closes
	_, err := chain.AddPRClose("PR-123", "user1", "abandoned", "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add first PR close: %v", err)
	}

	_, err = chain.AddPRClose("PR-456", "user2", "superseded", "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add second PR close: %v", err)
	}

	// Get all PR closes
	closes, err := chain.GetPRCloses()
	if err != nil {
		t.Fatalf("Failed to get PR closes: %v", err)
	}

	if len(closes) != 2 {
		t.Errorf("Expected 2 PR closes, got %d", len(closes))
	}

	if closes[0].PRNumber != "PR-123" {
		t.Errorf("Expected first PR number 'PR-123', got '%s'", closes[0].PRNumber)
	}

	if closes[1].PRNumber != "PR-456" {
		t.Errorf("Expected second PR number 'PR-456', got '%s'", closes[1].PRNumber)
	}
}

func TestPRCloseChain_GetPRCloseByNumber(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	// Add two PR closes
	_, err := chain.AddPRClose("PR-123", "user1", "abandoned", "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add first PR close: %v", err)
	}

	_, err = chain.AddPRClose("PR-456", "user2", "superseded", "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add second PR close: %v", err)
	}

	// Get PR close by number
	pr, err := chain.GetPRCloseByNumber("PR-123")
	if err != nil {
		t.Fatalf("Failed to get PR close by number: %v", err)
	}

	if pr.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", pr.PRNumber)
	}

	if pr.CloseReason != "abandoned" {
		t.Errorf("Expected close reason 'abandoned', got '%s'", pr.CloseReason)
	}

	// Try to get non-existent PR
	_, err = chain.GetPRCloseByNumber("PR-999")
	if err != ErrPRCloseNotFound {
		t.Errorf("Expected error ErrPRCloseNotFound, got %v", err)
	}
}

func TestPRCloseChain_GetPRCloseByIndex(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	// Add two PR closes
	_, err := chain.AddPRClose("PR-123", "user1", "abandoned", "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add first PR close: %v", err)
	}

	_, err = chain.AddPRClose("PR-456", "user2", "superseded", "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add second PR close: %v", err)
	}

	// Get PR close by index
	pr, err := chain.GetPRCloseByIndex(0)
	if err != nil {
		t.Fatalf("Failed to get PR close by index: %v", err)
	}

	if pr.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", pr.PRNumber)
	}

	// Try to get non-existent PR
	_, err = chain.GetPRCloseByIndex(999)
	if err != ErrBlockNotFound {
		t.Errorf("Expected error ErrBlockNotFound, got %v", err)
	}
}

func TestPRCloseChain_GetLatestPRClose(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	// Try to get latest PR close from empty chain
	_, err := chain.GetLatestPRClose()
	if err != ErrEmptyChain {
		t.Errorf("Expected error ErrEmptyChain, got %v", err)
	}

	// Add two PR closes
	_, err = chain.AddPRClose("PR-123", "user1", "abandoned", "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add first PR close: %v", err)
	}

	_, err = chain.AddPRClose("PR-456", "user2", "superseded", "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add second PR close: %v", err)
	}

	// Get latest PR close
	pr, err := chain.GetLatestPRClose()
	if err != nil {
		t.Fatalf("Failed to get latest PR close: %v", err)
	}

	if pr.PRNumber != "PR-456" {
		t.Errorf("Expected PR number 'PR-456', got '%s'", pr.PRNumber)
	}

	if pr.Closer != "user2" {
		t.Errorf("Expected closer 'user2', got '%s'", pr.Closer)
	}
}

func TestPRCloseChain_GetPRClosesByReason(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	// Add three PR closes with different reasons
	_, err := chain.AddPRClose("PR-123", "user1", "abandoned", "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add first PR close: %v", err)
	}

	_, err = chain.AddPRClose("PR-456", "user2", "superseded", "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add second PR close: %v", err)
	}

	_, err = chain.AddPRClose("PR-789", "user3", "abandoned", "Comment 3")
	if err != nil {
		t.Fatalf("Failed to add third PR close: %v", err)
	}

	// Get PR closes by reason
	abandonedPRs, err := chain.GetPRClosesByReason("abandoned")
	if err != nil {
		t.Fatalf("Failed to get PR closes by reason: %v", err)
	}

	if len(abandonedPRs) != 2 {
		t.Errorf("Expected 2 abandoned PRs, got %d", len(abandonedPRs))
	}

	supersededPRs, err := chain.GetPRClosesByReason("superseded")
	if err != nil {
		t.Fatalf("Failed to get PR closes by reason: %v", err)
	}

	if len(supersededPRs) != 1 {
		t.Errorf("Expected 1 superseded PR, got %d", len(supersededPRs))
	}

	if supersededPRs[0].PRNumber != "PR-456" {
		t.Errorf("Expected PR number 'PR-456', got '%s'", supersededPRs[0].PRNumber)
	}

	// Get PR closes by non-existent reason
	nonExistentPRs, err := chain.GetPRClosesByReason("nonexistent")
	if err != nil {
		t.Fatalf("Failed to get PR closes by reason: %v", err)
	}

	if len(nonExistentPRs) != 0 {
		t.Errorf("Expected 0 PRs with non-existent reason, got %d", len(nonExistentPRs))
	}
}

func TestPRCloseChain_Export(t *testing.T) {
	chain := NewPRCloseChain("test-repo")

	// Add a PR close
	_, err := chain.AddPRClose("PR-123", "user1", "abandoned", "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add PR close: %v", err)
	}

	// Export chain
	data, err := chain.Export()
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Import chain
	newChain := NewPRCloseChain("")
	err = newChain.Import(data)
	if err != nil {
		t.Fatalf("Failed to import chain: %v", err)
	}

	// Verify imported chain
	if newChain.ChainType != "pr-close" {
		t.Errorf("Expected chain type 'pr-close', got '%s'", newChain.ChainType)
	}

	if newChain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", newChain.RepoID)
	}

	if len(newChain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(newChain.Blocks))
	}

	// Verify block data
	var prData PRCloseData
	err = json.Unmarshal(newChain.Blocks[0].Data, &prData)
	if err != nil {
		t.Fatalf("Failed to unmarshal block data: %v", err)
	}

	if prData.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", prData.PRNumber)
	}

	if prData.CloseReason != "abandoned" {
		t.Errorf("Expected close reason 'abandoned', got '%s'", prData.CloseReason)
	}
}
