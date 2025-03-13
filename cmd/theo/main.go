package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gold2th/theo/internal/blockchain/action"
	"github.com/gold2th/theo/internal/blockchain/master"
	gitimport "github.com/gold2th/theo/internal/git/import"
)

// Command-line flags
var (
	repoID     = flag.String("repo", "", "Repository ID")
	ownerID    = flag.String("owner", "", "Owner ID")
	userID     = flag.String("user", "", "User ID")
	dataDir    = flag.String("data-dir", "./data", "Data directory")
	command    = flag.String("cmd", "help", "Command to execute (init, commit, merge, status, export, import-git)")
	message    = flag.String("message", "", "Commit message")
	branch     = flag.String("branch", "main", "Branch name")
	jsonlOut   = flag.Bool("jsonl", false, "Export in JSONL format")
	sourceRepo = flag.String("source", "", "Source Git repository URL or path for import")
)

func main() {
	flag.Parse()

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	// Execute the command
	switch *command {
	case "init":
		initRepo()
	case "commit":
		addCommit()
	case "merge":
		addMerge()
	case "status":
		showStatus()
	case "export":
		exportChains()
	case "import-git":
		importGitRepo()
	case "help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		showHelp()
		os.Exit(1)
	}
}

// initRepo initializes a new repository
func initRepo() {
	if *repoID == "" || *ownerID == "" {
		fmt.Println("Error: Repository ID and Owner ID are required")
		os.Exit(1)
	}

	// Check if repository already exists
	repoDir := filepath.Join(*dataDir, *repoID)
	if _, err := os.Stat(repoDir); err == nil {
		fmt.Printf("Repository %s already exists\n", *repoID)
		os.Exit(1)
	}

	// Create repository directory
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		fmt.Printf("Error creating repository directory: %v\n", err)
		os.Exit(1)
	}

	// Create repository master chain
	repoChain := master.NewRepoMasterChain(*repoID, *ownerID)

	// Save the chain
	if err := saveRepoChain(repoChain); err != nil {
		fmt.Printf("Error saving repository chain: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Repository %s initialized with owner %s\n", *repoID, *ownerID)
}

// addCommit adds a commit to the repository
func addCommit() {
	if *repoID == "" || *userID == "" || *message == "" {
		fmt.Println("Error: Repository ID, User ID, and message are required")
		os.Exit(1)
	}

	// Load repository chain
	repoChain, err := loadRepoChain()
	if err != nil {
		fmt.Printf("Error loading repository chain: %v\n", err)
		os.Exit(1)
	}

	// Load or create commit chain
	commitChain, err := loadActionChain("commit")
	if err != nil {
		commitChain = action.NewActionChain("commit", *repoID)
	}

	// Create commit data
	commitData := struct {
		CommitHash string `json:"commit_hash"`
		Author     string `json:"author"`
		Message    string `json:"message"`
		Timestamp  int64  `json:"timestamp"`
		Branch     string `json:"branch"`
	}{
		CommitHash: fmt.Sprintf("commit-%d", time.Now().UnixNano()),
		Author:     *userID,
		Message:    *message,
		Timestamp:  time.Now().Unix(),
		Branch:     *branch,
	}

	// Convert to JSON
	commitJSON, err := json.Marshal(commitData)
	if err != nil {
		fmt.Printf("Error marshaling commit data: %v\n", err)
		os.Exit(1)
	}

	// Add to commit chain
	commitHash, err := commitChain.AddBlock(commitJSON)
	if err != nil {
		fmt.Printf("Error adding commit to chain: %v\n", err)
		os.Exit(1)
	}

	// Save commit chain
	if err := saveActionChain(commitChain); err != nil {
		fmt.Printf("Error saving commit chain: %v\n", err)
		os.Exit(1)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: *userID,
	}
	_, err = repoChain.AddActionHash("commit", commitHash, userData)
	if err != nil {
		fmt.Printf("Error adding commit to repository chain: %v\n", err)
		os.Exit(1)
	}

	// Save repository chain
	if err := saveRepoChain(repoChain); err != nil {
		fmt.Printf("Error saving repository chain: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Commit added: %s\n", *message)
}

// addMerge adds a merge to the repository
func addMerge() {
	if *repoID == "" || *userID == "" || *message == "" {
		fmt.Println("Error: Repository ID, User ID, and message are required")
		os.Exit(1)
	}

	// Load repository chain
	repoChain, err := loadRepoChain()
	if err != nil {
		fmt.Printf("Error loading repository chain: %v\n", err)
		os.Exit(1)
	}

	// Load or create merge chain
	mergeChain, err := loadActionChain("merge")
	if err != nil {
		mergeChain = action.NewActionChain("merge", *repoID)
	}

	// Create merge data
	mergeData := struct {
		MergeHash    string `json:"merge_hash"`
		Author       string `json:"author"`
		Message      string `json:"message"`
		Timestamp    int64  `json:"timestamp"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
	}{
		MergeHash:    fmt.Sprintf("merge-%d", time.Now().UnixNano()),
		Author:       *userID,
		Message:      *message,
		Timestamp:    time.Now().Unix(),
		SourceBranch: flag.Arg(0),
		TargetBranch: *branch,
	}

	// Convert to JSON
	mergeJSON, err := json.Marshal(mergeData)
	if err != nil {
		fmt.Printf("Error marshaling merge data: %v\n", err)
		os.Exit(1)
	}

	// Add to merge chain
	mergeHash, err := mergeChain.AddBlock(mergeJSON)
	if err != nil {
		fmt.Printf("Error adding merge to chain: %v\n", err)
		os.Exit(1)
	}

	// Save merge chain
	if err := saveActionChain(mergeChain); err != nil {
		fmt.Printf("Error saving merge chain: %v\n", err)
		os.Exit(1)
	}

	// Add to repository master chain
	userData := master.UserData{
		ID: *userID,
	}
	_, err = repoChain.AddActionHash("merge", mergeHash, userData)
	if err != nil {
		fmt.Printf("Error adding merge to repository chain: %v\n", err)
		os.Exit(1)
	}

	// Save repository chain
	if err := saveRepoChain(repoChain); err != nil {
		fmt.Printf("Error saving repository chain: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Merge added: %s\n", *message)
}

// showStatus shows the status of the repository
func showStatus() {
	if *repoID == "" {
		fmt.Println("Error: Repository ID is required")
		os.Exit(1)
	}

	// Load repository chain
	repoChain, err := loadRepoChain()
	if err != nil {
		fmt.Printf("Error loading repository chain: %v\n", err)
		os.Exit(1)
	}

	// Get repository info
	fmt.Printf("Repository: %s\n", repoChain.RepoID)
	fmt.Printf("Owner: %s\n", repoChain.OwnerID)
	fmt.Printf("Blocks: %d\n", len(repoChain.Blocks))
	fmt.Printf("Verified: %v\n\n", repoChain.VerifyChain())

	// Load action chains
	actionTypes := []string{"commit", "merge", "branch"}
	for _, actionType := range actionTypes {
		actionChain, err := loadActionChain(actionType)
		if err != nil {
			continue
		}

		fmt.Printf("%s Chain:\n", capitalize(actionType))
		fmt.Printf("  Blocks: %d\n", len(actionChain.Blocks))
		fmt.Printf("  Verified: %v\n", actionChain.VerifyChain())

		// Show the latest 5 blocks
		fmt.Printf("  Latest Blocks:\n")
		start := len(actionChain.Blocks) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(actionChain.Blocks); i++ {
			block := actionChain.Blocks[i]
			fmt.Printf("    %d: %s\n", block.Index, block.Hash[:8])

			// Parse and display block data
			var data map[string]interface{}
			if err := json.Unmarshal(block.Data, &data); err == nil {
				if message, ok := data["message"].(string); ok {
					fmt.Printf("       Message: %s\n", message)
				}
				if author, ok := data["author"].(string); ok {
					fmt.Printf("       Author: %s\n", author)
				}
				if timestamp, ok := data["timestamp"].(float64); ok {
					fmt.Printf("       Time: %s\n", time.Unix(int64(timestamp), 0).Format(time.RFC3339))
				}
			}
		}
		fmt.Println()
	}
}

// exportChains exports the chains to files
func exportChains() {
	if *repoID == "" {
		fmt.Println("Error: Repository ID is required")
		os.Exit(1)
	}

	// Load repository chain
	repoChain, err := loadRepoChain()
	if err != nil {
		fmt.Printf("Error loading repository chain: %v\n", err)
		os.Exit(1)
	}

	// Export repository chain
	var repoData []byte
	var exportErr error
	if *jsonlOut {
		jsonl, err := repoChain.ExportToJSONL()
		if err != nil {
			fmt.Printf("Error exporting repository chain to JSONL: %v\n", err)
			os.Exit(1)
		}
		repoData = []byte(jsonl)
	} else {
		repoData, exportErr = repoChain.Export()
		if exportErr != nil {
			fmt.Printf("Error exporting repository chain: %v\n", exportErr)
			os.Exit(1)
		}
	}

	// Write to file
	repoFile := filepath.Join(*dataDir, *repoID, "export-repo.json")
	if *jsonlOut {
		repoFile = filepath.Join(*dataDir, *repoID, "export-repo.jsonl")
	}
	if err := os.WriteFile(repoFile, repoData, 0644); err != nil {
		fmt.Printf("Error writing repository chain to file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Repository chain exported to %s\n", repoFile)

	// Export action chains
	actionTypes := []string{"commit", "merge", "branch"}
	for _, actionType := range actionTypes {
		actionChain, err := loadActionChain(actionType)
		if err != nil {
			continue
		}

		var actionData []byte
		if *jsonlOut {
			jsonl, err := actionChain.ExportToJSONL()
			if err != nil {
				fmt.Printf("Error exporting %s chain to JSONL: %v\n", actionType, err)
				continue
			}
			actionData = []byte(jsonl)
		} else {
			actionData, exportErr = actionChain.Export()
			if exportErr != nil {
				fmt.Printf("Error exporting %s chain: %v\n", actionType, exportErr)
				continue
			}
		}

		// Write to file
		actionFile := filepath.Join(*dataDir, *repoID, fmt.Sprintf("export-%s.json", actionType))
		if *jsonlOut {
			actionFile = filepath.Join(*dataDir, *repoID, fmt.Sprintf("export-%s.jsonl", actionType))
		}
		if err := os.WriteFile(actionFile, actionData, 0644); err != nil {
			fmt.Printf("Error writing %s chain to file: %v\n", actionType, err)
			continue
		}

		fmt.Printf("%s chain exported to %s\n", capitalize(actionType), actionFile)
	}
}

// importGitRepo imports a Git repository into Theo
func importGitRepo() {
	if *repoID == "" || *ownerID == "" || *userID == "" || *sourceRepo == "" {
		fmt.Println("Error: Repository ID, Owner ID, User ID, and Source Repository are required")
		os.Exit(1)
	}

	// Create Git repository importer
	importer := gitimport.NewGitRepoImporter(*dataDir, *repoID, *ownerID, *userID, *sourceRepo)

	// Import the repository
	fmt.Printf("Importing Git repository %s into Theo repository %s...\n", *sourceRepo, *repoID)
	if err := importer.Import(); err != nil {
		fmt.Printf("Error importing repository: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Repository %s imported successfully\n", *sourceRepo)
}

// showHelp shows the help message
func showHelp() {
	fmt.Println("Theo - A blockchain-based Git system")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  theo [flags] [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init        Initialize a new repository")
	fmt.Println("  commit      Add a commit to the repository")
	fmt.Println("  merge       Add a merge to the repository")
	fmt.Println("  status      Show the status of the repository")
	fmt.Println("  export      Export the chains to files")
	fmt.Println("  import-git  Import a Git repository into Theo")
	fmt.Println("  help        Show this help message")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

// Helper functions

// loadRepoChain loads the repository master chain
func loadRepoChain() (*master.RepoMasterChain, error) {
	repoFile := filepath.Join(*dataDir, *repoID, "repo.json")
	data, err := os.ReadFile(repoFile)
	if err != nil {
		return nil, err
	}

	repoChain := master.NewRepoMasterChain("", "")
	if err := repoChain.Import(data); err != nil {
		return nil, err
	}

	return repoChain, nil
}

// saveRepoChain saves the repository master chain
func saveRepoChain(repoChain *master.RepoMasterChain) error {
	data, err := repoChain.Export()
	if err != nil {
		return err
	}

	repoDir := filepath.Join(*dataDir, repoChain.RepoID)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return err
	}

	repoFile := filepath.Join(repoDir, "repo.json")
	return os.WriteFile(repoFile, data, 0644)
}

// loadActionChain loads an action chain
func loadActionChain(actionType string) (*action.ActionChain, error) {
	actionFile := filepath.Join(*dataDir, *repoID, fmt.Sprintf("%s.json", actionType))
	data, err := os.ReadFile(actionFile)
	if err != nil {
		return nil, err
	}

	actionChain := action.NewActionChain("", "")
	if err := actionChain.Import(data); err != nil {
		return nil, err
	}

	return actionChain, nil
}

// saveActionChain saves an action chain
func saveActionChain(actionChain *action.ActionChain) error {
	data, err := actionChain.Export()
	if err != nil {
		return err
	}

	repoDir := filepath.Join(*dataDir, actionChain.RepoID)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return err
	}

	actionFile := filepath.Join(repoDir, fmt.Sprintf("%s.json", actionChain.ChainType))
	return os.WriteFile(actionFile, data, 0644)
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return string(s[0]-32) + s[1:]
}
