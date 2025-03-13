package action

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewRepoCreationChain(t *testing.T) {
	// Create a new repository creation chain
	repoID := "test-repo"
	chain := NewRepoCreationChain(repoID)

	// Verify chain properties
	if chain.ChainType != "repo-creation" {
		t.Errorf("Expected chain type 'repo-creation', got '%s'", chain.ChainType)
	}
	if chain.RepoID != repoID {
		t.Errorf("Expected repo ID '%s', got '%s'", repoID, chain.RepoID)
	}
	if len(chain.Blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(chain.Blocks))
	}
}

func TestRepoCreationChain_AddRepoCreation(t *testing.T) {
	// Create a new repository creation chain
	chain := NewRepoCreationChain("test-repo")

	// Add a repository creation
	repoName := "new-project"
	owner := "alice"
	visibility := "public"
	hash, err := chain.AddRepoCreation(repoName, owner, visibility)
	if err != nil {
		t.Fatalf("Failed to add repository creation: %v", err)
	}

	// Verify block was added
	if len(chain.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(chain.Blocks))
	}

	// Verify block hash
	if chain.Blocks[0].Hash != hash {
		t.Errorf("Expected hash '%s', got '%s'", hash, chain.Blocks[0].Hash)
	}

	// Verify block data
	var data RepoCreationData
	if err := json.Unmarshal(chain.Blocks[0].Data, &data); err != nil {
		t.Fatalf("Failed to unmarshal block data: %v", err)
	}
	if data.RepoName != repoName {
		t.Errorf("Expected repo name '%s', got '%s'", repoName, data.RepoName)
	}
	if data.Owner != owner {
		t.Errorf("Expected owner '%s', got '%s'", owner, data.Owner)
	}
	if data.Visibility != visibility {
		t.Errorf("Expected visibility '%s', got '%s'", visibility, data.Visibility)
	}

	// Verify timestamp is reasonable (within the last minute)
	now := time.Now().Unix()
	if data.CreationTimestamp > now || data.CreationTimestamp < now-60 {
		t.Errorf("Expected timestamp to be within the last minute, got %d (now: %d)", data.CreationTimestamp, now)
	}
}

func TestRepoCreationChain_GetRepoCreations(t *testing.T) {
	// Create a new repository creation chain
	chain := NewRepoCreationChain("test-repo")

	// Add multiple repository creations
	numCreations := 3
	for i := 0; i < numCreations; i++ {
		repoName := "project-" + string(rune('A'+i))
		owner := "user-" + string(rune('A'+i))
		visibility := "public"
		_, err := chain.AddRepoCreation(repoName, owner, visibility)
		if err != nil {
			t.Fatalf("Failed to add repository creation: %v", err)
		}
	}

	// Get all repository creations
	creations, err := chain.GetRepoCreations()
	if err != nil {
		t.Fatalf("Failed to get repository creations: %v", err)
	}

	// Verify number of creations
	if len(creations) != numCreations {
		t.Errorf("Expected %d creations, got %d", numCreations, len(creations))
	}

	// Verify creation data
	for i, creation := range creations {
		expectedRepoName := "project-" + string(rune('A'+i))
		expectedOwner := "user-" + string(rune('A'+i))
		if creation.RepoName != expectedRepoName {
			t.Errorf("Creation %d: Expected repo name '%s', got '%s'", i, expectedRepoName, creation.RepoName)
		}
		if creation.Owner != expectedOwner {
			t.Errorf("Creation %d: Expected owner '%s', got '%s'", i, expectedOwner, creation.Owner)
		}
	}
}

func TestRepoCreationChain_GetRepoCreationByIndex(t *testing.T) {
	// Create a new repository creation chain
	chain := NewRepoCreationChain("test-repo")

	// Add multiple repository creations
	numCreations := 3
	for i := 0; i < numCreations; i++ {
		repoName := "project-" + string(rune('A'+i))
		owner := "user-" + string(rune('A'+i))
		visibility := "public"
		_, err := chain.AddRepoCreation(repoName, owner, visibility)
		if err != nil {
			t.Fatalf("Failed to add repository creation: %v", err)
		}
	}

	// Get repository creation by index
	index := uint64(1)
	creation, err := chain.GetRepoCreationByIndex(index)
	if err != nil {
		t.Fatalf("Failed to get repository creation by index: %v", err)
	}

	// Verify creation data
	expectedRepoName := "project-B"
	expectedOwner := "user-B"
	if creation.RepoName != expectedRepoName {
		t.Errorf("Expected repo name '%s', got '%s'", expectedRepoName, creation.RepoName)
	}
	if creation.Owner != expectedOwner {
		t.Errorf("Expected owner '%s', got '%s'", expectedOwner, creation.Owner)
	}

	// Try to get a non-existent creation
	_, err = chain.GetRepoCreationByIndex(uint64(numCreations))
	if err != ErrBlockNotFound {
		t.Errorf("Expected error '%v', got '%v'", ErrBlockNotFound, err)
	}
}

func TestRepoCreationChain_GetLatestRepoCreation(t *testing.T) {
	// Create a new repository creation chain
	chain := NewRepoCreationChain("test-repo")

	// Try to get the latest creation from an empty chain
	_, err := chain.GetLatestRepoCreation()
	if err != ErrEmptyChain {
		t.Errorf("Expected error '%v', got '%v'", ErrEmptyChain, err)
	}

	// Add multiple repository creations
	numCreations := 3
	for i := 0; i < numCreations; i++ {
		repoName := "project-" + string(rune('A'+i))
		owner := "user-" + string(rune('A'+i))
		visibility := "public"
		_, err := chain.AddRepoCreation(repoName, owner, visibility)
		if err != nil {
			t.Fatalf("Failed to add repository creation: %v", err)
		}
	}

	// Get the latest repository creation
	creation, err := chain.GetLatestRepoCreation()
	if err != nil {
		t.Fatalf("Failed to get latest repository creation: %v", err)
	}

	// Verify creation data
	expectedRepoName := "project-C"
	expectedOwner := "user-C"
	if creation.RepoName != expectedRepoName {
		t.Errorf("Expected repo name '%s', got '%s'", expectedRepoName, creation.RepoName)
	}
	if creation.Owner != expectedOwner {
		t.Errorf("Expected owner '%s', got '%s'", expectedOwner, creation.Owner)
	}
}
