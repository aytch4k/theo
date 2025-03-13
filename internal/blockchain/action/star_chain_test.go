package action

import (
	"testing"
	"time"
)

func TestNewStarChain(t *testing.T) {
	chain := NewStarChain("test-repo")
	if chain == nil {
		t.Fatal("Failed to create star chain")
	}
	if chain.ChainType != "star" {
		t.Errorf("Expected chain type 'star', got '%s'", chain.ChainType)
	}
	if chain.RepoID != "test-repo" {
		t.Errorf("Expected repo ID 'test-repo', got '%s'", chain.RepoID)
	}
}

func TestStarChain_AddStar(t *testing.T) {
	chain := NewStarChain("test-repo")

	// Add a star
	hash, err := chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}
	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Verify the chain has one block
	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}
}

func TestStarChain_GetStars(t *testing.T) {
	chain := NewStarChain("test-repo")

	// Add stars
	_, err := chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	_, err = chain.AddStar("bob")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	// Get all stars
	stars, err := chain.GetStars()
	if err != nil {
		t.Fatalf("Failed to get stars: %v", err)
	}

	// Verify we have two stars
	if len(stars) != 2 {
		t.Errorf("Expected 2 stars, got %d", len(stars))
	}

	// Verify star data
	if stars[0].User != "alice" {
		t.Errorf("Expected user 'alice', got '%s'", stars[0].User)
	}

	if stars[1].User != "bob" {
		t.Errorf("Expected user 'bob', got '%s'", stars[1].User)
	}
}

func TestStarChain_GetStarByIndex(t *testing.T) {
	chain := NewStarChain("test-repo")

	// Add stars
	_, err := chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	_, err = chain.AddStar("bob")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	// Get star by index
	star, err := chain.GetStarByIndex(1)
	if err != nil {
		t.Fatalf("Failed to get star by index: %v", err)
	}

	// Verify star data
	if star.User != "bob" {
		t.Errorf("Expected user 'bob', got '%s'", star.User)
	}

	// Try to get a non-existent star
	_, err = chain.GetStarByIndex(2)
	if err != ErrBlockNotFound {
		t.Errorf("Expected ErrBlockNotFound, got %v", err)
	}
}

func TestStarChain_GetStarsByUser(t *testing.T) {
	chain := NewStarChain("test-repo")

	// Add stars
	_, err := chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	_, err = chain.AddStar("bob")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	_, err = chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	// Get stars by user
	stars, err := chain.GetStarsByUser("alice")
	if err != nil {
		t.Fatalf("Failed to get stars by user: %v", err)
	}

	// Verify we have two stars by alice
	if len(stars) != 2 {
		t.Errorf("Expected 2 stars by alice, got %d", len(stars))
	}
}

func TestStarChain_GetLatestStar(t *testing.T) {
	chain := NewStarChain("test-repo")

	// Try to get the latest star from an empty chain
	_, err := chain.GetLatestStar()
	if err != ErrEmptyChain {
		t.Errorf("Expected ErrEmptyChain, got %v", err)
	}

	// Add stars
	_, err = chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	_, err = chain.AddStar("bob")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	// Get the latest star
	star, err := chain.GetLatestStar()
	if err != nil {
		t.Fatalf("Failed to get latest star: %v", err)
	}

	// Verify star data
	if star.User != "bob" {
		t.Errorf("Expected user 'bob', got '%s'", star.User)
	}
}

func TestStarChain_GetStarCount(t *testing.T) {
	chain := NewStarChain("test-repo")

	// Verify empty chain has 0 stars
	if count := chain.GetStarCount(); count != 0 {
		t.Errorf("Expected 0 stars, got %d", count)
	}

	// Add stars
	_, err := chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	_, err = chain.AddStar("bob")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	// Verify we have 2 stars
	if count := chain.GetStarCount(); count != 2 {
		t.Errorf("Expected 2 stars, got %d", count)
	}
}

func TestStarChain_HasUserStarred(t *testing.T) {
	chain := NewStarChain("test-repo")

	// Verify user has not starred an empty chain
	if chain.HasUserStarred("alice") {
		t.Error("Expected alice to not have starred the repo")
	}

	// Add stars
	_, err := chain.AddStar("alice")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	_, err = chain.AddStar("bob")
	if err != nil {
		t.Fatalf("Failed to add star: %v", err)
	}

	// Verify alice has starred the repo
	if !chain.HasUserStarred("alice") {
		t.Error("Expected alice to have starred the repo")
	}

	// Verify charlie has not starred the repo
	if chain.HasUserStarred("charlie") {
		t.Error("Expected charlie to not have starred the repo")
	}
}
