package gitimport

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gold2th/theo/internal/blockchain/action"
	"github.com/gold2th/theo/internal/blockchain/master"
	"github.com/google/uuid"
)

// GitRepoImporter handles importing Git repositories into Theo
type GitRepoImporter struct {
	DataDir     string
	RepoID      string
	OwnerID     string
	UserID      string
	SourceRepo  string
	WorkingDir  string
	CommitChain *action.ActionChain
	MergeChain  *action.ActionChain
	BranchChain *action.ActionChain
	RepoChain   *master.RepoMasterChain
}

// CommitInfo represents Git commit information
type CommitInfo struct {
	Hash      string
	Author    string
	Email     string
	Message   string
	Timestamp time.Time
	Branch    string
}

// MergeInfo represents Git merge information
type MergeInfo struct {
	Hash         string
	Author       string
	Email        string
	Message      string
	Timestamp    time.Time
	SourceBranch string
	TargetBranch string
}

// BranchInfo represents Git branch information
type BranchInfo struct {
	Name      string
	CommitRef string
	Author    string
	Email     string
	Timestamp time.Time
}

// NewGitRepoImporter creates a new GitRepoImporter
func NewGitRepoImporter(dataDir, repoID, ownerID, userID, sourceRepo string) *GitRepoImporter {
	return &GitRepoImporter{
		DataDir:    dataDir,
		RepoID:     repoID,
		OwnerID:    ownerID,
		UserID:     userID,
		SourceRepo: sourceRepo,
	}
}

// Import imports a Git repository into Theo
func (g *GitRepoImporter) Import() error {
	// Create a temporary working directory
	workingDir, err := os.MkdirTemp("", "theo-import-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	g.WorkingDir = workingDir
	defer os.RemoveAll(workingDir)

	// Clone the repository
	if err := g.cloneRepository(); err != nil {
		return err
	}

	// Initialize Theo chains
	if err := g.initializeChains(); err != nil {
		return err
	}

	// Import branches
	if err := g.importBranches(); err != nil {
		return err
	}

	// Import commits
	if err := g.importCommits(); err != nil {
		return err
	}

	// Import merges
	if err := g.importMerges(); err != nil {
		return err
	}

	// Save chains
	if err := g.saveChains(); err != nil {
		return err
	}

	return nil
}

// cloneRepository clones the source Git repository
func (g *GitRepoImporter) cloneRepository() error {
	fmt.Printf("Cloning repository %s...\n", g.SourceRepo)
	cmd := exec.Command("git", "clone", "--mirror", g.SourceRepo, g.WorkingDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// initializeChains initializes the Theo blockchain chains
func (g *GitRepoImporter) initializeChains() error {
	fmt.Println("Initializing Theo chains...")

	// Create repository directory
	repoDir := filepath.Join(g.DataDir, g.RepoID)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Initialize repository master chain
	g.RepoChain = master.NewRepoMasterChain(g.RepoID, g.OwnerID)

	// Initialize action chains
	g.CommitChain = action.NewActionChain("commit", g.RepoID)
	g.MergeChain = action.NewActionChain("merge", g.RepoID)
	g.BranchChain = action.NewActionChain("branch", g.RepoID)

	return nil
}

// importBranches imports all branches from the Git repository
func (g *GitRepoImporter) importBranches() error {
	fmt.Println("Importing branches...")

	// Get all branches
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	branches := strings.Split(string(output), "\n")
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}

		// Remove leading * and spaces
		if strings.HasPrefix(branch, "*") {
			branch = strings.TrimSpace(branch[1:])
		}

		// Skip remote branches
		if strings.HasPrefix(branch, "remotes/") {
			continue
		}

		// Get branch info
		branchInfo, err := g.getBranchInfo(branch)
		if err != nil {
			fmt.Printf("Warning: Failed to get info for branch %s: %v\n", branch, err)
			continue
		}

		// Add branch to chain
		if err := g.addBranchToChain(branchInfo); err != nil {
			fmt.Printf("Warning: Failed to add branch %s to chain: %v\n", branch, err)
			continue
		}
	}

	return nil
}

// getBranchInfo gets information about a branch
func (g *GitRepoImporter) getBranchInfo(branch string) (*BranchInfo, error) {
	// Get commit hash for branch
	cmd := exec.Command("git", "rev-parse", branch)
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hash for branch: %w", err)
	}
	commitRef := strings.TrimSpace(string(output))

	// Get author and timestamp from the first commit on the branch
	cmd = exec.Command("git", "log", "-1", "--format=%an|%ae|%at", branch)
	cmd.Dir = g.WorkingDir
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get author info for branch: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected format for branch author info")
	}

	author := parts[0]
	email := parts[1]
	timestamp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return &BranchInfo{
		Name:      branch,
		CommitRef: commitRef,
		Author:    author,
		Email:     email,
		Timestamp: time.Unix(timestamp, 0),
	}, nil
}

// addBranchToChain adds a branch to the branch chain
func (g *GitRepoImporter) addBranchToChain(branchInfo *BranchInfo) error {
	// Create branch data
	branchData := struct {
		BranchID  string `json:"branch_id"`
		Name      string `json:"name"`
		CommitRef string `json:"commit_ref"`
		Author    string `json:"author"`
		Email     string `json:"email"`
		Timestamp int64  `json:"timestamp"`
	}{
		BranchID:  uuid.New().String(),
		Name:      branchInfo.Name,
		CommitRef: branchInfo.CommitRef,
		Author:    branchInfo.Author,
		Email:     branchInfo.Email,
		Timestamp: branchInfo.Timestamp.Unix(),
	}

	// Convert to JSON
	branchJSON, err := json.Marshal(branchData)
	if err != nil {
		return fmt.Errorf("failed to marshal branch data: %w", err)
	}

	// Add to branch chain
	branchHash, err := g.BranchChain.AddBlock(branchJSON)
	if err != nil {
		return fmt.Errorf("failed to add branch to chain: %w", err)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: g.UserID,
	}
	_, err = g.RepoChain.AddActionHash("branch", branchHash, userData)
	if err != nil {
		return fmt.Errorf("failed to add branch to repository chain: %w", err)
	}

	fmt.Printf("Added branch: %s\n", branchInfo.Name)
	return nil
}

// importCommits imports all commits from the Git repository
func (g *GitRepoImporter) importCommits() error {
	fmt.Println("Importing commits...")

	// Get all commits
	cmd := exec.Command("git", "log", "--all", "--format=%H|%an|%ae|%s|%at|%P")
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	commits := strings.Split(string(output), "\n")
	for _, commit := range commits {
		commit = strings.TrimSpace(commit)
		if commit == "" {
			continue
		}

		// Parse commit info
		parts := strings.SplitN(commit, "|", 6)
		if len(parts) != 6 {
			fmt.Printf("Warning: Unexpected format for commit: %s\n", commit)
			continue
		}

		hash := parts[0]
		author := parts[1]
		email := parts[2]
		message := parts[3]
		timestamp, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			fmt.Printf("Warning: Failed to parse timestamp for commit %s: %v\n", hash, err)
			continue
		}
		parentHashes := parts[5]

		// Skip merge commits (they will be handled separately)
		if strings.Contains(parentHashes, " ") {
			continue
		}

		// Get branch for commit
		branch, err := g.getBranchForCommit(hash)
		if err != nil {
			fmt.Printf("Warning: Failed to get branch for commit %s: %v\n", hash, err)
			branch = "unknown"
		}

		commitInfo := &CommitInfo{
			Hash:      hash,
			Author:    author,
			Email:     email,
			Message:   message,
			Timestamp: time.Unix(timestamp, 0),
			Branch:    branch,
		}

		// Add commit to chain
		if err := g.addCommitToChain(commitInfo); err != nil {
			fmt.Printf("Warning: Failed to add commit %s to chain: %v\n", hash, err)
			continue
		}
	}

	return nil
}

// getBranchForCommit gets the branch for a commit
func (g *GitRepoImporter) getBranchForCommit(hash string) (string, error) {
	cmd := exec.Command("git", "branch", "--contains", hash)
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get branch for commit: %w", err)
	}

	branches := strings.Split(string(output), "\n")
	if len(branches) == 0 {
		return "", fmt.Errorf("no branches found for commit")
	}

	// Use the first branch
	branch := strings.TrimSpace(branches[0])
	if strings.HasPrefix(branch, "*") {
		branch = strings.TrimSpace(branch[1:])
	}

	return branch, nil
}

// addCommitToChain adds a commit to the commit chain
func (g *GitRepoImporter) addCommitToChain(commitInfo *CommitInfo) error {
	// Create commit data
	commitData := struct {
		CommitHash string `json:"commit_hash"`
		Author     string `json:"author"`
		Email      string `json:"email"`
		Message    string `json:"message"`
		Timestamp  int64  `json:"timestamp"`
		Branch     string `json:"branch"`
	}{
		CommitHash: commitInfo.Hash,
		Author:     commitInfo.Author,
		Email:      commitInfo.Email,
		Message:    commitInfo.Message,
		Timestamp:  commitInfo.Timestamp.Unix(),
		Branch:     commitInfo.Branch,
	}

	// Convert to JSON
	commitJSON, err := json.Marshal(commitData)
	if err != nil {
		return fmt.Errorf("failed to marshal commit data: %w", err)
	}

	// Add to commit chain
	commitHash, err := g.CommitChain.AddBlock(commitJSON)
	if err != nil {
		return fmt.Errorf("failed to add commit to chain: %w", err)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: g.UserID,
	}
	_, err = g.RepoChain.AddActionHash("commit", commitHash, userData)
	if err != nil {
		return fmt.Errorf("failed to add commit to repository chain: %w", err)
	}

	fmt.Printf("Added commit: %s - %s\n", commitInfo.Hash[:8], commitInfo.Message)
	return nil
}

// importMerges imports all merge commits from the Git repository
func (g *GitRepoImporter) importMerges() error {
	fmt.Println("Importing merges...")

	// Get all merge commits
	cmd := exec.Command("git", "log", "--all", "--merges", "--format=%H|%an|%ae|%s|%at|%P")
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get merge commits: %w", err)
	}

	merges := strings.Split(string(output), "\n")
	for _, merge := range merges {
		merge = strings.TrimSpace(merge)
		if merge == "" {
			continue
		}

		// Parse merge info
		parts := strings.SplitN(merge, "|", 6)
		if len(parts) != 6 {
			fmt.Printf("Warning: Unexpected format for merge: %s\n", merge)
			continue
		}

		hash := parts[0]
		author := parts[1]
		email := parts[2]
		message := parts[3]
		timestamp, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			fmt.Printf("Warning: Failed to parse timestamp for merge %s: %v\n", hash, err)
			continue
		}
		parentHashes := parts[5]

		// Parse parent hashes
		parents := strings.Split(parentHashes, " ")
		if len(parents) < 2 {
			fmt.Printf("Warning: Unexpected number of parents for merge %s: %s\n", hash, parentHashes)
			continue
		}

		// Get branches for parent commits
		targetBranch, err := g.getBranchForCommit(parents[0])
		if err != nil {
			fmt.Printf("Warning: Failed to get target branch for merge %s: %v\n", hash, err)
			targetBranch = "unknown"
		}

		sourceBranch, err := g.getBranchForCommit(parents[1])
		if err != nil {
			fmt.Printf("Warning: Failed to get source branch for merge %s: %v\n", hash, err)
			sourceBranch = "unknown"
		}

		mergeInfo := &MergeInfo{
			Hash:         hash,
			Author:       author,
			Email:        email,
			Message:      message,
			Timestamp:    time.Unix(timestamp, 0),
			SourceBranch: sourceBranch,
			TargetBranch: targetBranch,
		}

		// Add merge to chain
		if err := g.addMergeToChain(mergeInfo); err != nil {
			fmt.Printf("Warning: Failed to add merge %s to chain: %v\n", hash, err)
			continue
		}
	}

	return nil
}

// addMergeToChain adds a merge to the merge chain
func (g *GitRepoImporter) addMergeToChain(mergeInfo *MergeInfo) error {
	// Create merge data
	mergeData := struct {
		MergeHash    string `json:"merge_hash"`
		Author       string `json:"author"`
		Email        string `json:"email"`
		Message      string `json:"message"`
		Timestamp    int64  `json:"timestamp"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
	}{
		MergeHash:    mergeInfo.Hash,
		Author:       mergeInfo.Author,
		Email:        mergeInfo.Email,
		Message:      mergeInfo.Message,
		Timestamp:    mergeInfo.Timestamp.Unix(),
		SourceBranch: mergeInfo.SourceBranch,
		TargetBranch: mergeInfo.TargetBranch,
	}

	// Convert to JSON
	mergeJSON, err := json.Marshal(mergeData)
	if err != nil {
		return fmt.Errorf("failed to marshal merge data: %w", err)
	}

	// Add to merge chain
	mergeHash, err := g.MergeChain.AddBlock(mergeJSON)
	if err != nil {
		return fmt.Errorf("failed to add merge to chain: %w", err)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: g.UserID,
	}
	_, err = g.RepoChain.AddActionHash("merge", mergeHash, userData)
	if err != nil {
		return fmt.Errorf("failed to add merge to repository chain: %w", err)
	}

	fmt.Printf("Added merge: %s - %s\n", mergeInfo.Hash[:8], mergeInfo.Message)
	return nil
}

// saveChains saves all chains to disk
func (g *GitRepoImporter) saveChains() error {
	fmt.Println("Saving chains...")

	// Create repository directory
	repoDir := filepath.Join(g.DataDir, g.RepoID)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Save repository chain
	repoData, err := g.RepoChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export repository chain: %w", err)
	}
	repoFile := filepath.Join(repoDir, "repo.json")
	if err := os.WriteFile(repoFile, repoData, 0644); err != nil {
		return fmt.Errorf("failed to write repository chain to file: %w", err)
	}

	// Save commit chain
	commitData, err := g.CommitChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export commit chain: %w", err)
	}
	commitFile := filepath.Join(repoDir, "commit.json")
	if err := os.WriteFile(commitFile, commitData, 0644); err != nil {
		return fmt.Errorf("failed to write commit chain to file: %w", err)
	}

	// Save merge chain
	mergeData, err := g.MergeChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export merge chain: %w", err)
	}
	mergeFile := filepath.Join(repoDir, "merge.json")
	if err := os.WriteFile(mergeFile, mergeData, 0644); err != nil {
		return fmt.Errorf("failed to write merge chain to file: %w", err)
	}

	// Save branch chain
	branchData, err := g.BranchChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export branch chain: %w", err)
	}
	branchFile := filepath.Join(repoDir, "branch.json")
	if err := os.WriteFile(branchFile, branchData, 0644); err != nil {
		return fmt.Errorf("failed to write branch chain to file: %w", err)
	}

	fmt.Println("Chains saved successfully")
	return nil
}
