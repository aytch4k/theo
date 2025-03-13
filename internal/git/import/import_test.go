package gitimport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gold2th/theo/internal/blockchain/action"
)

func TestGitRepoImporter_InitializeChains(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "theo-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a GitRepoImporter
	importer := NewGitRepoImporter(tempDir, "test-repo", "test-owner", "test-user", "test-source")

	// Initialize chains
	err = importer.initializeChains()
	if err != nil {
		t.Fatalf("Failed to initialize chains: %v", err)
	}

	// Verify that all chains were initialized
	if importer.CommitChain == nil {
		t.Error("CommitChain was not initialized")
	}
	if importer.MergeChain == nil {
		t.Error("MergeChain was not initialized")
	}
	if importer.BranchChain == nil {
		t.Error("BranchChain was not initialized")
	}
	if importer.BranchDeleteChain == nil {
		t.Error("BranchDeleteChain was not initialized")
	}
	if importer.TagChain == nil {
		t.Error("TagChain was not initialized")
	}
	if importer.RepoCreationChain == nil {
		t.Error("RepoCreationChain was not initialized")
	}
	if importer.PRCreateChain == nil {
		t.Error("PRCreateChain was not initialized")
	}
	if importer.PRMergeChain == nil {
		t.Error("PRMergeChain was not initialized")
	}
	if importer.PRCloseChain == nil {
		t.Error("PRCloseChain was not initialized")
	}
	if importer.ForkChain == nil {
		t.Error("ForkChain was not initialized")
	}
	if importer.StarChain == nil {
		t.Error("StarChain was not initialized")
	}
	if importer.WatchChain == nil {
		t.Error("WatchChain was not initialized")
	}
	if importer.RepoChain == nil {
		t.Error("RepoChain was not initialized")
	}

	// Verify chain types and repo IDs
	if importer.CommitChain.ChainType != "commit" || importer.CommitChain.RepoID != "test-repo" {
		t.Errorf("CommitChain has incorrect type or repo ID: %s, %s", importer.CommitChain.ChainType, importer.CommitChain.RepoID)
	}
	if importer.MergeChain.ChainType != "merge" || importer.MergeChain.RepoID != "test-repo" {
		t.Errorf("MergeChain has incorrect type or repo ID: %s, %s", importer.MergeChain.ChainType, importer.MergeChain.RepoID)
	}
	if importer.BranchChain.ChainType != "branch" || importer.BranchChain.RepoID != "test-repo" {
		t.Errorf("BranchChain has incorrect type or repo ID: %s, %s", importer.BranchChain.ChainType, importer.BranchChain.RepoID)
	}
	if importer.BranchDeleteChain.ChainType != "branch-delete" || importer.BranchDeleteChain.RepoID != "test-repo" {
		t.Errorf("BranchDeleteChain has incorrect type or repo ID: %s, %s", importer.BranchDeleteChain.ChainType, importer.BranchDeleteChain.RepoID)
	}
	if importer.TagChain.ChainType != "tag" || importer.TagChain.RepoID != "test-repo" {
		t.Errorf("TagChain has incorrect type or repo ID: %s, %s", importer.TagChain.ChainType, importer.TagChain.RepoID)
	}
}

func TestGitRepoImporter_AddTagToChain(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "theo-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a GitRepoImporter
	importer := NewGitRepoImporter(tempDir, "test-repo", "test-owner", "test-user", "test-source")

	// Initialize chains
	err = importer.initializeChains()
	if err != nil {
		t.Fatalf("Failed to initialize chains: %v", err)
	}

	// Create a test tag
	tagInfo := &TagInfo{
		Name:      "v1.0.0",
		CommitRef: "abcdef1234567890",
		Author:    "Test Author",
		Email:     "test@example.com",
		Message:   "Test tag message",
		Timestamp: time.Now(),
	}

	// Add tag to chain
	err = importer.addTagToChain(tagInfo)
	if err != nil {
		t.Fatalf("Failed to add tag to chain: %v", err)
	}

	// Verify that the tag was added to the tag chain
	if len(importer.TagChain.Blocks) != 1 {
		t.Errorf("Expected 1 block in tag chain, got %d", len(importer.TagChain.Blocks))
	}

	// Verify that the tag was added to the repository chain
	// Note: There should be 2 blocks - one for repo creation and one for the tag
	if len(importer.RepoChain.Blocks) != 2 {
		t.Errorf("Expected 2 blocks in repository chain, got %d", len(importer.RepoChain.Blocks))
	}

	// Verify the tag data
	var tagData struct {
		TagID     string `json:"tag_id"`
		Name      string `json:"name"`
		CommitRef string `json:"commit_ref"`
		Author    string `json:"author"`
		Email     string `json:"email"`
		Message   string `json:"message"`
		Timestamp int64  `json:"timestamp"`
	}
	err = json.Unmarshal(importer.TagChain.Blocks[0].Data, &tagData)
	if err != nil {
		t.Fatalf("Failed to unmarshal tag data: %v", err)
	}

	if tagData.Name != tagInfo.Name {
		t.Errorf("Expected tag name %s, got %s", tagInfo.Name, tagData.Name)
	}
	if tagData.CommitRef != tagInfo.CommitRef {
		t.Errorf("Expected commit ref %s, got %s", tagInfo.CommitRef, tagData.CommitRef)
	}
	if tagData.Author != tagInfo.Author {
		t.Errorf("Expected author %s, got %s", tagInfo.Author, tagData.Author)
	}
	if tagData.Email != tagInfo.Email {
		t.Errorf("Expected email %s, got %s", tagInfo.Email, tagData.Email)
	}
	if tagData.Message != tagInfo.Message {
		t.Errorf("Expected message %s, got %s", tagInfo.Message, tagData.Message)
	}
	if tagData.Timestamp != tagInfo.Timestamp.Unix() {
		t.Errorf("Expected timestamp %d, got %d", tagInfo.Timestamp.Unix(), tagData.Timestamp)
	}
}

func TestGitRepoImporter_AddBranchDeleteToChain(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "theo-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a GitRepoImporter
	importer := NewGitRepoImporter(tempDir, "test-repo", "test-owner", "test-user", "test-source")

	// Initialize chains
	err = importer.initializeChains()
	if err != nil {
		t.Fatalf("Failed to initialize chains: %v", err)
	}

	// Create a test branch delete
	branchDeleteInfo := &BranchDeleteInfo{
		Name:      "feature-123",
		Author:    "Test Author",
		Email:     "test@example.com",
		Timestamp: time.Now(),
	}

	// Add branch delete to chain
	err = importer.addBranchDeleteToChain(branchDeleteInfo)
	if err != nil {
		t.Fatalf("Failed to add branch delete to chain: %v", err)
	}

	// Verify that the branch delete was added to the branch delete chain
	if len(importer.BranchDeleteChain.Blocks) != 1 {
		t.Errorf("Expected 1 block in branch delete chain, got %d", len(importer.BranchDeleteChain.Blocks))
	}

	// Verify that the branch delete was added to the repository chain
	// Note: There should be 2 blocks - one for repo creation and one for the branch delete
	if len(importer.RepoChain.Blocks) != 2 {
		t.Errorf("Expected 2 blocks in repository chain, got %d", len(importer.RepoChain.Blocks))
	}

	// Verify the branch delete data
	var branchDeleteData struct {
		BranchDeleteID string `json:"branch_delete_id"`
		Name           string `json:"name"`
		Author         string `json:"author"`
		Email          string `json:"email"`
		Timestamp      int64  `json:"timestamp"`
	}
	err = json.Unmarshal(importer.BranchDeleteChain.Blocks[0].Data, &branchDeleteData)
	if err != nil {
		t.Fatalf("Failed to unmarshal branch delete data: %v", err)
	}

	if branchDeleteData.Name != branchDeleteInfo.Name {
		t.Errorf("Expected branch name %s, got %s", branchDeleteInfo.Name, branchDeleteData.Name)
	}
	if branchDeleteData.Author != branchDeleteInfo.Author {
		t.Errorf("Expected author %s, got %s", branchDeleteInfo.Author, branchDeleteData.Author)
	}
	if branchDeleteData.Email != branchDeleteInfo.Email {
		t.Errorf("Expected email %s, got %s", branchDeleteInfo.Email, branchDeleteData.Email)
	}
	if branchDeleteData.Timestamp != branchDeleteInfo.Timestamp.Unix() {
		t.Errorf("Expected timestamp %d, got %d", branchDeleteInfo.Timestamp.Unix(), branchDeleteData.Timestamp)
	}
}

func TestGitRepoImporter_AddRepoCreationToChain(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "theo-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a GitRepoImporter
	importer := NewGitRepoImporter(tempDir, "test-repo", "test-owner", "test-user", "test-source")

	// Initialize chains
	err = importer.initializeChains()
	if err != nil {
		t.Fatalf("Failed to initialize chains: %v", err)
	}

	// Verify that the repository creation was added to the repository creation chain
	if len(importer.RepoCreationChain.Blocks) != 1 {
		t.Errorf("Expected 1 block in repository creation chain, got %d", len(importer.RepoCreationChain.Blocks))
	}

	// Verify that the repository creation was added to the repository chain
	if len(importer.RepoChain.Blocks) != 1 {
		t.Errorf("Expected 1 block in repository chain, got %d", len(importer.RepoChain.Blocks))
	}

	// Verify the repository creation data
	var repoCreationData struct {
		RepoName          string `json:"repo_name"`
		Owner             string `json:"owner"`
		CreationTimestamp int64  `json:"creation_timestamp"`
		Visibility        string `json:"visibility"`
	}
	err = json.Unmarshal(importer.RepoCreationChain.Blocks[0].Data, &repoCreationData)
	if err != nil {
		t.Fatalf("Failed to unmarshal repository creation data: %v", err)
	}

	if repoCreationData.RepoName != "test-repo" {
		t.Errorf("Expected repo name 'test-repo', got '%s'", repoCreationData.RepoName)
	}
	if repoCreationData.Owner != "test-owner" {
		t.Errorf("Expected owner 'test-owner', got '%s'", repoCreationData.Owner)
	}
	if repoCreationData.Visibility != "public" {
		t.Errorf("Expected visibility 'public', got '%s'", repoCreationData.Visibility)
	}

	// Verify timestamp is reasonable (within the last minute)
	now := time.Now().Unix()
	if repoCreationData.CreationTimestamp > now || repoCreationData.CreationTimestamp < now-60 {
		t.Errorf("Expected timestamp to be within the last minute, got %d (now: %d)", repoCreationData.CreationTimestamp, now)
	}
}

func TestGitRepoImporter_SaveChains(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "theo-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a GitRepoImporter
	importer := NewGitRepoImporter(tempDir, "test-repo", "test-owner", "test-user", "test-source")

	// Initialize chains
	err = importer.initializeChains()
	if err != nil {
		t.Fatalf("Failed to initialize chains: %v", err)
	}

	// Add some data to the chains
	tagInfo := &TagInfo{
		Name:      "v1.0.0",
		CommitRef: "abcdef1234567890",
		Author:    "Test Author",
		Email:     "test@example.com",
		Message:   "Test tag message",
		Timestamp: time.Now(),
	}
	err = importer.addTagToChain(tagInfo)
	if err != nil {
		t.Fatalf("Failed to add tag to chain: %v", err)
	}

	branchDeleteInfo := &BranchDeleteInfo{
		Name:      "feature-123",
		Author:    "Test Author",
		Email:     "test@example.com",
		Timestamp: time.Now(),
	}
	err = importer.addBranchDeleteToChain(branchDeleteInfo)
	if err != nil {
		t.Fatalf("Failed to add branch delete to chain: %v", err)
	}

	// Save chains
	err = importer.saveChains()
	if err != nil {
		t.Fatalf("Failed to save chains: %v", err)
	}

	// Verify that the chain files were created
	repoDir := filepath.Join(tempDir, "test-repo")
	files := []string{
		"repo.json",
		"commit.json",
		"merge.json",
		"branch.json",
		"branch-delete.json",
		"tag.json",
		"repo-creation.json",
		"pr-create.json",
		"pr-merge.json",
		"pr-close.json",
		"fork.json",
		"star.json",
		"watch.json",
	}
	for _, file := range files {
		filePath := filepath.Join(repoDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", filePath)
		}
	}

	// Verify that the tag chain file contains the correct data
	tagChainData, err := os.ReadFile(filepath.Join(repoDir, "tag.json"))
	if err != nil {
		t.Fatalf("Failed to read tag chain file: %v", err)
	}
	tagChain := action.NewActionChain("", "")
	err = tagChain.Import(tagChainData)
	if err != nil {
		t.Fatalf("Failed to import tag chain data: %v", err)
	}
	if len(tagChain.Blocks) != 1 {
		t.Errorf("Expected 1 block in tag chain, got %d", len(tagChain.Blocks))
	}

	// Verify that the branch delete chain file contains the correct data
	branchDeleteChainData, err := os.ReadFile(filepath.Join(repoDir, "branch-delete.json"))
	if err != nil {
		t.Fatalf("Failed to read branch delete chain file: %v", err)
	}
	branchDeleteChain := action.NewActionChain("", "")
	err = branchDeleteChain.Import(branchDeleteChainData)
	if err != nil {
		t.Fatalf("Failed to import branch delete chain data: %v", err)
	}
	if len(branchDeleteChain.Blocks) != 1 {
		t.Errorf("Expected 1 block in branch delete chain, got %d", len(branchDeleteChain.Blocks))
	}

	// Verify that the repo creation chain file contains the correct data
	repoCreationChainData, err := os.ReadFile(filepath.Join(repoDir, "repo-creation.json"))
	if err != nil {
		t.Fatalf("Failed to read repo creation chain file: %v", err)
	}
	repoCreationChain := action.NewActionChain("", "")
	err = repoCreationChain.Import(repoCreationChainData)
	if err != nil {
		t.Fatalf("Failed to import repo creation chain data: %v", err)
	}
	if len(repoCreationChain.Blocks) != 1 {
		t.Errorf("Expected 1 block in repo creation chain, got %d", len(repoCreationChain.Blocks))
	}
}
