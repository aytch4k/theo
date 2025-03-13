package action

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewPRCreateChain(t *testing.T) {
	chain := NewPRCreateChain("test-repo")

	if chain.ChainType != "pr-create" {
		t.Errorf("Expected chain type 'pr-create', got '%s'", chain.ChainType)
	}

	if chain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", chain.RepoID)
	}

	if len(chain.Blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(chain.Blocks))
	}
}

func TestPRCreateChain_AddPRCreate(t *testing.T) {
	chain := NewPRCreateChain("test-repo")

	hash, err := chain.AddPRCreate(
		"PR-123",
		"Add new feature",
		"feature-branch",
		"main",
		"user1",
		"This PR adds a new feature",
	)

	if err != nil {
		t.Fatalf("Failed to add PR create: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}

	// Verify block data
	var data PRCreateData
	err = json.Unmarshal(chain.Blocks[0].Data, &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal block data: %v", err)
	}

	if data.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", data.PRNumber)
	}

	if data.Title != "Add new feature" {
		t.Errorf("Expected title 'Add new feature', got '%s'", data.Title)
	}

	if data.SourceBranch != "feature-branch" {
		t.Errorf("Expected source branch 'feature-branch', got '%s'", data.SourceBranch)
	}

	if data.TargetBranch != "main" {
		t.Errorf("Expected target branch 'main', got '%s'", data.TargetBranch)
	}

	if data.Creator != "user1" {
		t.Errorf("Expected creator 'user1', got '%s'", data.Creator)
	}

	if data.Description != "This PR adds a new feature" {
		t.Errorf("Expected description 'This PR adds a new feature', got '%s'", data.Description)
	}

	// Verify timestamp is reasonable (within the last minute)
	now := time.Now().Unix()
	if data.Timestamp > now || data.Timestamp < now-60 {
		t.Errorf("Expected timestamp to be within the last minute, got %d (now: %d)", data.Timestamp, now)
	}
}

func TestPRCreateChain_GetPRCreations(t *testing.T) {
	chain := NewPRCreateChain("test-repo")

	// Add two PR creations
	_, err := chain.AddPRCreate("PR-123", "First PR", "feature1", "main", "user1", "First PR description")
	if err != nil {
		t.Fatalf("Failed to add first PR create: %v", err)
	}

	_, err = chain.AddPRCreate("PR-456", "Second PR", "feature2", "main", "user2", "Second PR description")
	if err != nil {
		t.Fatalf("Failed to add second PR create: %v", err)
	}

	// Get all PR creations
	creations, err := chain.GetPRCreations()
	if err != nil {
		t.Fatalf("Failed to get PR creations: %v", err)
	}

	if len(creations) != 2 {
		t.Errorf("Expected 2 PR creations, got %d", len(creations))
	}

	if creations[0].PRNumber != "PR-123" {
		t.Errorf("Expected first PR number 'PR-123', got '%s'", creations[0].PRNumber)
	}

	if creations[1].PRNumber != "PR-456" {
		t.Errorf("Expected second PR number 'PR-456', got '%s'", creations[1].PRNumber)
	}
}

func TestPRCreateChain_GetPRCreationByNumber(t *testing.T) {
	chain := NewPRCreateChain("test-repo")

	// Add two PR creations
	_, err := chain.AddPRCreate("PR-123", "First PR", "feature1", "main", "user1", "First PR description")
	if err != nil {
		t.Fatalf("Failed to add first PR create: %v", err)
	}

	_, err = chain.AddPRCreate("PR-456", "Second PR", "feature2", "main", "user2", "Second PR description")
	if err != nil {
		t.Fatalf("Failed to add second PR create: %v", err)
	}

	// Get PR creation by number
	pr, err := chain.GetPRCreationByNumber("PR-123")
	if err != nil {
		t.Fatalf("Failed to get PR creation by number: %v", err)
	}

	if pr.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", pr.PRNumber)
	}

	if pr.Title != "First PR" {
		t.Errorf("Expected title 'First PR', got '%s'", pr.Title)
	}

	// Try to get non-existent PR
	_, err = chain.GetPRCreationByNumber("PR-999")
	if err != ErrPRNotFound {
		t.Errorf("Expected error ErrPRNotFound, got %v", err)
	}
}

func TestPRCreateChain_GetPRCreationByIndex(t *testing.T) {
	chain := NewPRCreateChain("test-repo")

	// Add two PR creations
	_, err := chain.AddPRCreate("PR-123", "First PR", "feature1", "main", "user1", "First PR description")
	if err != nil {
		t.Fatalf("Failed to add first PR create: %v", err)
	}

	_, err = chain.AddPRCreate("PR-456", "Second PR", "feature2", "main", "user2", "Second PR description")
	if err != nil {
		t.Fatalf("Failed to add second PR create: %v", err)
	}

	// Get PR creation by index
	pr, err := chain.GetPRCreationByIndex(0)
	if err != nil {
		t.Fatalf("Failed to get PR creation by index: %v", err)
	}

	if pr.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", pr.PRNumber)
	}

	// Try to get non-existent PR
	_, err = chain.GetPRCreationByIndex(999)
	if err != ErrBlockNotFound {
		t.Errorf("Expected error ErrBlockNotFound, got %v", err)
	}
}

func TestPRCreateChain_GetLatestPRCreation(t *testing.T) {
	chain := NewPRCreateChain("test-repo")

	// Try to get latest PR creation from empty chain
	_, err := chain.GetLatestPRCreation()
	if err != ErrEmptyChain {
		t.Errorf("Expected error ErrEmptyChain, got %v", err)
	}

	// Add two PR creations
	_, err = chain.AddPRCreate("PR-123", "First PR", "feature1", "main", "user1", "First PR description")
	if err != nil {
		t.Fatalf("Failed to add first PR create: %v", err)
	}

	_, err = chain.AddPRCreate("PR-456", "Second PR", "feature2", "main", "user2", "Second PR description")
	if err != nil {
		t.Fatalf("Failed to add second PR create: %v", err)
	}

	// Get latest PR creation
	pr, err := chain.GetLatestPRCreation()
	if err != nil {
		t.Fatalf("Failed to get latest PR creation: %v", err)
	}

	if pr.PRNumber != "PR-456" {
		t.Errorf("Expected PR number 'PR-456', got '%s'", pr.PRNumber)
	}

	if pr.Title != "Second PR" {
		t.Errorf("Expected title 'Second PR', got '%s'", pr.Title)
	}
}

func TestPRCreateChain_Export(t *testing.T) {
	chain := NewPRCreateChain("test-repo")

	// Add a PR creation
	_, err := chain.AddPRCreate("PR-123", "Test PR", "feature", "main", "user1", "Test PR description")
	if err != nil {
		t.Fatalf("Failed to add PR create: %v", err)
	}

	// Export chain
	data, err := chain.Export()
	if err != nil {
		t.Fatalf("Failed to export chain: %v", err)
	}

	// Import chain
	newChain := NewPRCreateChain("")
	err = newChain.Import(data)
	if err != nil {
		t.Fatalf("Failed to import chain: %v", err)
	}

	// Verify imported chain
	if newChain.ChainType != "pr-create" {
		t.Errorf("Expected chain type 'pr-create', got '%s'", newChain.ChainType)
	}

	if newChain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", newChain.RepoID)
	}

	if len(newChain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(newChain.Blocks))
	}

	// Verify block data
	var prData PRCreateData
	err = json.Unmarshal(newChain.Blocks[0].Data, &prData)
	if err != nil {
		t.Fatalf("Failed to unmarshal block data: %v", err)
	}

	if prData.PRNumber != "PR-123" {
		t.Errorf("Expected PR number 'PR-123', got '%s'", prData.PRNumber)
	}
}
