package action

import (
	"testing"
	"time"
)

func TestNewWatchChain(t *testing.T) {
	chain := NewWatchChain("test-repo")
	if chain == nil {
		t.Fatal("Failed to create watch chain")
	}
	if chain.ChainType != "watch" {
		t.Errorf("Expected chain type 'watch', got '%s'", chain.ChainType)
	}
	if chain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", chain.RepoID)
	}
}

func TestWatchChain_AddWatch(t *testing.T) {
	chain := NewWatchChain("test-repo")

	// Add a watch
	hash, err := chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Verify the chain has one block
	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}
}

func TestWatchChain_GetWatches(t *testing.T) {
	chain := NewWatchChain("test-repo")

	// Add watches
	_, err := chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	_, err = chain.AddWatch("bob")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Get all watches
	watches, err := chain.GetWatches()
	if err != nil {
		t.Fatalf("Failed to get watches: %v", err)
	}

	// Verify we have two watches
	if len(watches) != 2 {
		t.Errorf("Expected 2 watches, got %d", len(watches))
	}

	// Verify watch data
	if watches[0].User != "alice" {
		t.Errorf("Expected user 'alice', got '%s'", watches[0].User)
	}

	if watches[1].User != "bob" {
		t.Errorf("Expected user 'bob', got '%s'", watches[1].User)
	}
}

func TestWatchChain_GetWatchByIndex(t *testing.T) {
	chain := NewWatchChain("test-repo")

	// Add watches
	_, err := chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	_, err = chain.AddWatch("bob")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Get watch by index
	watch, err := chain.GetWatchByIndex(1)
	if err != nil {
		t.Fatalf("Failed to get watch by index: %v", err)
	}

	// Verify watch data
	if watch.User != "bob" {
		t.Errorf("Expected user 'bob', got '%s'", watch.User)
	}

	// Try to get a non-existent watch
	_, err = chain.GetWatchByIndex(2)
	if err != ErrBlockNotFound {
		t.Errorf("Expected ErrBlockNotFound, got %v", err)
	}
}

func TestWatchChain_GetWatchesByUser(t *testing.T) {
	chain := NewWatchChain("test-repo")

	// Add watches
	_, err := chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	_, err = chain.AddWatch("bob")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	_, err = chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Get watches by user
	watches, err := chain.GetWatchesByUser("alice")
	if err != nil {
		t.Fatalf("Failed to get watches by user: %v", err)
	}

	// Verify we have two watches by alice
	if len(watches) != 2 {
		t.Errorf("Expected 2 watches by alice, got %d", len(watches))
	}
}

func TestWatchChain_GetLatestWatch(t *testing.T) {
	chain := NewWatchChain("test-repo")

	// Try to get the latest watch from an empty chain
	_, err := chain.GetLatestWatch()
	if err != ErrEmptyChain {
		t.Errorf("Expected ErrEmptyChain, got %v", err)
	}

	// Add watches
	_, err = chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	_, err = chain.AddWatch("bob")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Get the latest watch
	watch, err := chain.GetLatestWatch()
	if err != nil {
		t.Fatalf("Failed to get latest watch: %v", err)
	}

	// Verify watch data
	if watch.User != "bob" {
		t.Errorf("Expected user 'bob', got '%s'", watch.User)
	}
}

func TestWatchChain_GetWatchCount(t *testing.T) {
	chain := NewWatchChain("test-repo")

	// Verify empty chain has 0 watches
	if count := chain.GetWatchCount(); count != 0 {
		t.Errorf("Expected 0 watches, got %d", count)
	}

	// Add watches
	_, err := chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	_, err = chain.AddWatch("bob")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Verify we have 2 watches
	if count := chain.GetWatchCount(); count != 2 {
		t.Errorf("Expected 2 watches, got %d", count)
	}
}

func TestWatchChain_IsUserWatching(t *testing.T) {
	chain := NewWatchChain("test-repo")

	// Verify user is not watching an empty chain
	if chain.IsUserWatching("alice") {
		t.Error("Expected alice to not be watching the repo")
	}

	// Add watches
	_, err := chain.AddWatch("alice")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	_, err = chain.AddWatch("bob")
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Verify alice is watching the repo
	if !chain.IsUserWatching("alice") {
		t.Error("Expected alice to be watching the repo")
	}

	// Verify charlie is not watching the repo
	if chain.IsUserWatching("charlie") {
		t.Error("Expected charlie to not be watching the repo")
	}
}
