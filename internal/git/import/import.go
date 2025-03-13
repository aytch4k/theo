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
	DataDir           string
	RepoID            string
	OwnerID           string
	UserID            string
	SourceRepo        string
	WorkingDir        string
	CommitChain       *action.ActionChain
	MergeChain        *action.ActionChain
	BranchChain       *action.ActionChain
	BranchDeleteChain *action.ActionChain
	TagChain          *action.ActionChain
	RepoCreationChain *action.RepoCreationChain
	PRCreateChain     *action.PRCreateChain
	PRMergeChain      *action.PRMergeChain
	PRCloseChain      *action.PRCloseChain
	RepoChain         *master.RepoMasterChain
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

// TagInfo represents Git tag information
type TagInfo struct {
	Name      string
	CommitRef string
	Author    string
	Email     string
	Message   string
	Timestamp time.Time
}

// BranchDeleteInfo represents Git branch deletion information
type BranchDeleteInfo struct {
	Name      string
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

	// Import tags
	if err := g.importTags(); err != nil {
		return err
	}

	// Import deleted branches
	if err := g.importDeletedBranches(); err != nil {
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
	g.BranchDeleteChain = action.NewActionChain("branch-delete", g.RepoID)
	g.TagChain = action.NewActionChain("tag", g.RepoID)
	g.RepoCreationChain = action.NewRepoCreationChain(g.RepoID)
	g.PRCreateChain = action.NewPRCreateChain(g.RepoID)
	g.PRMergeChain = action.NewPRMergeChain(g.RepoID)
	g.PRCloseChain = action.NewPRCloseChain(g.RepoID)

	// Add repository creation to chain
	if err := g.addRepoCreationToChain(); err != nil {
		return fmt.Errorf("failed to add repository creation to chain: %w", err)
	}

	return nil
}

// addRepoCreationToChain adds the repository creation to the chain
func (g *GitRepoImporter) addRepoCreationToChain() error {
	// Determine repository visibility (public by default)
	visibility := "public"

	// Add repository creation to chain
	hash, err := g.RepoCreationChain.AddRepoCreation(g.RepoID, g.OwnerID, visibility)
	if err != nil {
		return fmt.Errorf("failed to add repository creation to chain: %w", err)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: g.OwnerID,
	}
	_, err = g.RepoChain.AddActionHash("repo-creation", hash, userData)
	if err != nil {
		return fmt.Errorf("failed to add repository creation to repository chain: %w", err)
	}

	fmt.Printf("Added repository creation: %s\n", g.RepoID)
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

// importTags imports all tags from the Git repository
func (g *GitRepoImporter) importTags() error {
	fmt.Println("Importing tags...")

	// Get all tags
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}

	tags := strings.Split(string(output), "\n")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		// Get tag info
		tagInfo, err := g.getTagInfo(tag)
		if err != nil {
			fmt.Printf("Warning: Failed to get info for tag %s: %v\n", tag, err)
			continue
		}

		// Add tag to chain
		if err := g.addTagToChain(tagInfo); err != nil {
			fmt.Printf("Warning: Failed to add tag %s to chain: %v\n", tag, err)
			continue
		}
	}

	return nil
}

// getTagInfo gets information about a tag
func (g *GitRepoImporter) getTagInfo(tag string) (*TagInfo, error) {
	// Get commit hash for tag
	cmd := exec.Command("git", "rev-parse", tag)
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hash for tag: %w", err)
	}
	commitRef := strings.TrimSpace(string(output))

	// Get tag message
	cmd = exec.Command("git", "tag", "-n", tag)
	cmd.Dir = g.WorkingDir
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag message: %w", err)
	}
	message := strings.TrimSpace(string(output))
	// Remove tag name from the beginning of the message
	if strings.HasPrefix(message, tag) {
		message = strings.TrimSpace(message[len(tag):])
	}

	// Get author and timestamp from the tag
	cmd = exec.Command("git", "for-each-ref", "--format=%(taggername)|%(taggeremail)|%(taggerdate:unix)", "refs/tags/"+tag)
	cmd.Dir = g.WorkingDir
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get author info for tag: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "|")

	// If the tag is not an annotated tag, get the author from the commit
	if len(parts) != 3 || parts[0] == "" {
		cmd = exec.Command("git", "log", "-1", "--format=%an|%ae|%at", tag)
		cmd.Dir = g.WorkingDir
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get author info for tag commit: %w", err)
		}
		parts = strings.Split(strings.TrimSpace(string(output)), "|")
	}

	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected format for tag author info")
	}

	author := parts[0]
	email := parts[1]
	timestamp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return &TagInfo{
		Name:      tag,
		CommitRef: commitRef,
		Author:    author,
		Email:     email,
		Message:   message,
		Timestamp: time.Unix(timestamp, 0),
	}, nil
}

// addTagToChain adds a tag to the tag chain
func (g *GitRepoImporter) addTagToChain(tagInfo *TagInfo) error {
	// Create tag data
	tagData := struct {
		TagID     string `json:"tag_id"`
		Name      string `json:"name"`
		CommitRef string `json:"commit_ref"`
		Author    string `json:"author"`
		Email     string `json:"email"`
		Message   string `json:"message"`
		Timestamp int64  `json:"timestamp"`
	}{
		TagID:     uuid.New().String(),
		Name:      tagInfo.Name,
		CommitRef: tagInfo.CommitRef,
		Author:    tagInfo.Author,
		Email:     tagInfo.Email,
		Message:   tagInfo.Message,
		Timestamp: tagInfo.Timestamp.Unix(),
	}

	// Convert to JSON
	tagJSON, err := json.Marshal(tagData)
	if err != nil {
		return fmt.Errorf("failed to marshal tag data: %w", err)
	}

	// Add to tag chain
	tagHash, err := g.TagChain.AddBlock(tagJSON)
	if err != nil {
		return fmt.Errorf("failed to add tag to chain: %w", err)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: g.UserID,
	}
	_, err = g.RepoChain.AddActionHash("tag", tagHash, userData)
	if err != nil {
		return fmt.Errorf("failed to add tag to repository chain: %w", err)
	}

	fmt.Printf("Added tag: %s\n", tagInfo.Name)
	return nil
}

// importDeletedBranches imports all deleted branches from the Git repository
func (g *GitRepoImporter) importDeletedBranches() error {
	fmt.Println("Importing deleted branches...")

	// Get all deleted branches
	cmd := exec.Command("git", "log", "--diff-filter=D", "--summary")
	cmd.Dir = g.WorkingDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get deleted branches: %w", err)
	}

	// Parse the output to find deleted branches
	lines := strings.Split(string(output), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, "delete mode") && strings.Contains(line, "refs/heads/") {
			// Extract branch name
			parts := strings.Split(line, "refs/heads/")
			if len(parts) < 2 {
				continue
			}
			branchName := strings.TrimSpace(parts[1])

			// Get author and timestamp from the commit
			var author, email string
			var timestamp time.Time

			// Look for the commit info in the previous lines
			for j := i - 1; j >= 0 && j >= i-5; j-- {
				if strings.HasPrefix(lines[j], "Author:") {
					authorLine := strings.TrimPrefix(lines[j], "Author:")
					authorParts := strings.Split(authorLine, "<")
					if len(authorParts) >= 2 {
						author = strings.TrimSpace(authorParts[0])
						email = strings.TrimSuffix(strings.TrimSpace(authorParts[1]), ">")
					}
				} else if strings.HasPrefix(lines[j], "Date:") {
					dateLine := strings.TrimPrefix(lines[j], "Date:")
					dateLine = strings.TrimSpace(dateLine)
					// Parse the date
					t, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", dateLine)
					if err == nil {
						timestamp = t
					}
				}

				// If we found both author and timestamp, break
				if author != "" && !timestamp.IsZero() {
					break
				}
			}

			// If we couldn't find the author or timestamp, use defaults
			if author == "" {
				author = "unknown"
			}
			if email == "" {
				email = "unknown"
			}
			if timestamp.IsZero() {
				timestamp = time.Now()
			}

			// Create branch delete info
			branchDeleteInfo := &BranchDeleteInfo{
				Name:      branchName,
				Author:    author,
				Email:     email,
				Timestamp: timestamp,
			}

			// Add branch delete to chain
			if err := g.addBranchDeleteToChain(branchDeleteInfo); err != nil {
				fmt.Printf("Warning: Failed to add deleted branch %s to chain: %v\n", branchName, err)
				continue
			}
		}
	}

	return nil
}

// addBranchDeleteToChain adds a branch deletion to the branch delete chain
func (g *GitRepoImporter) addBranchDeleteToChain(branchDeleteInfo *BranchDeleteInfo) error {
	// Create branch delete data
	branchDeleteData := struct {
		BranchDeleteID string `json:"branch_delete_id"`
		Name           string `json:"name"`
		Author         string `json:"author"`
		Email          string `json:"email"`
		Timestamp      int64  `json:"timestamp"`
	}{
		BranchDeleteID: uuid.New().String(),
		Name:           branchDeleteInfo.Name,
		Author:         branchDeleteInfo.Author,
		Email:          branchDeleteInfo.Email,
		Timestamp:      branchDeleteInfo.Timestamp.Unix(),
	}

	// Convert to JSON
	branchDeleteJSON, err := json.Marshal(branchDeleteData)
	if err != nil {
		return fmt.Errorf("failed to marshal branch delete data: %w", err)
	}

	// Add to branch delete chain
	branchDeleteHash, err := g.BranchDeleteChain.AddBlock(branchDeleteJSON)
	if err != nil {
		return fmt.Errorf("failed to add branch delete to chain: %w", err)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: g.UserID,
	}
	_, err = g.RepoChain.AddActionHash("branch-delete", branchDeleteHash, userData)
	if err != nil {
		return fmt.Errorf("failed to add branch delete to repository chain: %w", err)
	}

	fmt.Printf("Added deleted branch: %s\n", branchDeleteInfo.Name)
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

	// Save tag chain
	tagData, err := g.TagChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export tag chain: %w", err)
	}
	tagFile := filepath.Join(repoDir, "tag.json")
	if err := os.WriteFile(tagFile, tagData, 0644); err != nil {
		return fmt.Errorf("failed to write tag chain to file: %w", err)
	}

	// Save branch delete chain
	branchDeleteData, err := g.BranchDeleteChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export branch delete chain: %w", err)
	}
	branchDeleteFile := filepath.Join(repoDir, "branch-delete.json")
	if err := os.WriteFile(branchDeleteFile, branchDeleteData, 0644); err != nil {
		return fmt.Errorf("failed to write branch delete chain to file: %w", err)
	}

	// Save repository creation chain
	repoCreationData, err := g.RepoCreationChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export repository creation chain: %w", err)
	}
	repoCreationFile := filepath.Join(repoDir, "repo-creation.json")
	if err := os.WriteFile(repoCreationFile, repoCreationData, 0644); err != nil {
		return fmt.Errorf("failed to write repository creation chain to file: %w", err)
	}

	// Save PR create chain
	prCreateData, err := g.PRCreateChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export PR create chain: %w", err)
	}
	prCreateFile := filepath.Join(repoDir, "pr-create.json")
	if err := os.WriteFile(prCreateFile, prCreateData, 0644); err != nil {
		return fmt.Errorf("failed to write PR create chain to file: %w", err)
	}

	// Save PR merge chain
	prMergeData, err := g.PRMergeChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export PR merge chain: %w", err)
	}
	prMergeFile := filepath.Join(repoDir, "pr-merge.json")
	if err := os.WriteFile(prMergeFile, prMergeData, 0644); err != nil {
		return fmt.Errorf("failed to write PR merge chain to file: %w", err)
	}

	// Save PR close chain
	prCloseData, err := g.PRCloseChain.Export()
	if err != nil {
		return fmt.Errorf("failed to export PR close chain: %w", err)
	}
	prCloseFile := filepath.Join(repoDir, "pr-close.json")
	if err := os.WriteFile(prCloseFile, prCloseData, 0644); err != nil {
		return fmt.Errorf("failed to write PR close chain to file: %w", err)
	}

	fmt.Println("Chains saved successfully")
	return nil
}
