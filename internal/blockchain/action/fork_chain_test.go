package action

import (
	"testing"
	"time"
)

func TestNewForkChain(t *testing.T) {
	chain := NewForkChain("test-repo")
	if chain == nil {
		t.Fatal("Failed to create fork chain")
	}
	if chain.ChainType != "fork" {
		t.Errorf("Expected chain type 'fork', got '%s'", chain.ChainType)
	}
	if chain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", chain.RepoID)
	}
}

func TestForkChain_AddFork(t *testing.T) {
	chain := NewForkChain("test-repo")

	// Add a fork
	hash, err := chain.AddFork("original-repo", "forked-repo", "alice")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Verify the chain has one block
	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}
}

func TestForkChain_GetForks(t *testing.T) {
	chain := NewForkChain("test-repo")

	// Add forks
	_, err := chain.AddFork("original-repo", "forked-repo-1", "alice")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	_, err = chain.AddFork("original-repo", "forked-repo-2", "bob")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	// Get all forks
	forks, err := chain.GetForks()
	if err != nil {
		t.Fatalf("Failed to get forks: %v", err)
	}

	// Verify we have two forks
	if len(forks) != 2 {
		t.Errorf("Expected 2 forks, got %d", len(forks))
	}

	// Verify fork data
	if forks[0].ParentRepo != "original-repo" {
		t.Errorf("Expected parent repo 'original-repo', got '%s'", forks[0].ParentRepo)
	}
	if forks[0].ForkRepoName != "forked-repo-1" {
		t.Errorf("Expected fork repo name 'forked-repo-1', got '%s'", forks[0].ForkRepoName)
	}
	if forks[0].Forker != "alice" {
		t.Errorf("Expected forker 'alice', got '%s'", forks[0].Forker)
	}

	if forks[1].ParentRepo != "original-repo" {
		t.Errorf("Expected parent repo 'original-repo', got '%s'", forks[1].ParentRepo)
	}
	if forks[1].ForkRepoName != "forked-repo-2" {
		t.Errorf("Expected fork repo name 'forked-repo-2', got '%s'", forks[1].ForkRepoName)
	}
	if forks[1].Forker != "bob" {
		t.Errorf("Expected forker 'bob', got '%s'", forks[1].Forker)
	}
}

func TestForkChain_GetForkByIndex(t *testing.T) {
	chain := NewForkChain("test-repo")

	// Add forks
	_, err := chain.AddFork("original-repo", "forked-repo-1", "alice")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	_, err = chain.AddFork("original-repo", "forked-repo-2", "bob")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	// Get fork by index
	fork, err := chain.GetForkByIndex(1)
	if err != nil {
		t.Fatalf("Failed to get fork by index: %v", err)
	}

	// Verify fork data
	if fork.ParentRepo != "original-repo" {
		t.Errorf("Expected parent repo 'original-repo', got '%s'", fork.ParentRepo)
	}
	if fork.ForkRepoName != "forked-repo-2" {
		t.Errorf("Expected fork repo name 'forked-repo-2', got '%s'", fork.ForkRepoName)
	}
	if fork.Forker != "bob" {
		t.Errorf("Expected forker 'bob', got '%s'", fork.Forker)
	}

	// Try to get a non-existent fork
	_, err = chain.GetForkByIndex(2)
	if err != ErrBlockNotFound {
		t.Errorf("Expected ErrBlockNotFound, got %v", err)
	}
}

func TestForkChain_GetForksByForker(t *testing.T) {
	chain := NewForkChain("test-repo")

	// Add forks
	_, err := chain.AddFork("original-repo", "forked-repo-1", "alice")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	_, err = chain.AddFork("original-repo", "forked-repo-2", "bob")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	_, err = chain.AddFork("original-repo", "forked-repo-3", "alice")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	// Get forks by forker
	forks, err := chain.GetForksByForker("alice")
	if err != nil {
		t.Fatalf("Failed to get forks by forker: %v", err)
	}

	// Verify we have two forks by alice
	if len(forks) != 2 {
		t.Errorf("Expected 2 forks by alice, got %d", len(forks))
	}

	// Verify fork data
	if forks[0].ForkRepoName != "forked-repo-1" {
		t.Errorf("Expected fork repo name 'forked-repo-1', got '%s'", forks[0].ForkRepoName)
	}
	if forks[1].ForkRepoName != "forked-repo-3" {
		t.Errorf("Expected fork repo name 'forked-repo-3', got '%s'", forks[1].ForkRepoName)
	}
}

func TestForkChain_GetLatestFork(t *testing.T) {
	chain := NewForkChain("test-repo")

	// Try to get the latest fork from an empty chain
	_, err := chain.GetLatestFork()
	if err != ErrEmptyChain {
		t.Errorf("Expected ErrEmptyChain, got %v", err)
	}

	// Add forks
	_, err = chain.AddFork("original-repo", "forked-repo-1", "alice")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	_, err = chain.AddFork("original-repo", "forked-repo-2", "bob")
	if err != nil {
		t.Fatalf("Failed to add fork: %v", err)
	}

	// Get the latest fork
	fork, err := chain.GetLatestFork()
	if err != nil {
		t.Fatalf("Failed to get latest fork: %v", err)
	}

	// Verify fork data
	if fork.ForkRepoName != "forked-repo-2" {
		t.Errorf("Expected fork repo name 'forked-repo-2', got '%s'", fork.ForkRepoName)
	}
	if fork.Forker != "bob" {
		t.Errorf("Expected forker 'bob', got '%s'", fork.Forker)
	}
}
