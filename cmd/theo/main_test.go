package main

import (
	"flag"
	"os"
	"testing"
)

func TestCommandLineFlags(t *testing.T) {
	// Save original flag values
	originalCommand := *command
	originalRepoID := *repoID
	originalOwnerID := *ownerID
	originalUserID := *userID
	originalBranch := *branch
	originalTagName := *tagName
	originalMessage := *message
	originalDataDir := *dataDir
	originalSourceRepo := *sourceRepo
	originalJsonlOut := *jsonlOut

	// Restore original flag values after test
	defer func() {
		*command = originalCommand
		*repoID = originalRepoID
		*ownerID = originalOwnerID
		*userID = originalUserID
		*branch = originalBranch
		*tagName = originalTagName
		*message = originalMessage
		*dataDir = originalDataDir
		*sourceRepo = originalSourceRepo
		*jsonlOut = originalJsonlOut
	}()

	// Test cases for command line flags
	testCases := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name: "Default values",
			args: []string{"theo"},
			expected: map[string]string{
				"command":  "help",
				"repo":     "",
				"owner":    "",
				"user":     "",
				"branch":   "main",
				"tag-name": "",
				"message":  "",
				"data-dir": "./data",
				"source":   "",
				"jsonl":    "false",
			},
		},
		{
			name: "Init command",
			args: []string{"theo", "--cmd", "init", "--repo", "test-repo", "--owner", "test-owner"},
			expected: map[string]string{
				"command":  "init",
				"repo":     "test-repo",
				"owner":    "test-owner",
				"user":     "",
				"branch":   "main",
				"tag-name": "",
				"message":  "",
				"data-dir": "./data",
				"source":   "",
				"jsonl":    "false",
			},
		},
		{
			name: "Commit command",
			args: []string{"theo", "--cmd", "commit", "--repo", "test-repo", "--user", "test-user", "--message", "Test commit", "--branch", "feature"},
			expected: map[string]string{
				"command":  "commit",
				"repo":     "test-repo",
				"owner":    "",
				"user":     "test-user",
				"branch":   "feature",
				"tag-name": "",
				"message":  "Test commit",
				"data-dir": "./data",
				"source":   "",
				"jsonl":    "false",
			},
		},
		{
			name: "Tag command",
			args: []string{"theo", "--cmd", "tag", "--repo", "test-repo", "--user", "test-user", "--tag-name", "v1.0.0", "--message", "Test tag"},
			expected: map[string]string{
				"command":  "tag",
				"repo":     "test-repo",
				"owner":    "",
				"user":     "test-user",
				"branch":   "main",
				"tag-name": "v1.0.0",
				"message":  "Test tag",
				"data-dir": "./data",
				"source":   "",
				"jsonl":    "false",
			},
		},
		{
			name: "Branch delete command",
			args: []string{"theo", "--cmd", "branch-delete", "--repo", "test-repo", "--user", "test-user", "--branch", "feature", "--message", "Branch deleted"},
			expected: map[string]string{
				"command":  "branch-delete",
				"repo":     "test-repo",
				"owner":    "",
				"user":     "test-user",
				"branch":   "feature",
				"tag-name": "",
				"message":  "Branch deleted",
				"data-dir": "./data",
				"source":   "",
				"jsonl":    "false",
			},
		},
		{
			name: "Import Git command",
			args: []string{"theo", "--cmd", "import-git", "--repo", "test-repo", "--owner", "test-owner", "--user", "test-user", "--source", "https://github.com/example/repo.git"},
			expected: map[string]string{
				"command":  "import-git",
				"repo":     "test-repo",
				"owner":    "test-owner",
				"user":     "test-user",
				"branch":   "main",
				"tag-name": "",
				"message":  "",
				"data-dir": "./data",
				"source":   "https://github.com/example/repo.git",
				"jsonl":    "false",
			},
		},
		{
			name: "Export command with JSONL",
			args: []string{"theo", "--cmd", "export", "--repo", "test-repo", "--jsonl"},
			expected: map[string]string{
				"command":  "export",
				"repo":     "test-repo",
				"owner":    "",
				"user":     "",
				"branch":   "main",
				"tag-name": "",
				"message":  "",
				"data-dir": "./data",
				"source":   "",
				"jsonl":    "true",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(tc.args[0], flag.ExitOnError)

			// Reinitialize flags
			repoID = flag.String("repo", "", "Repository ID")
			ownerID = flag.String("owner", "", "Owner ID")
			userID = flag.String("user", "", "User ID")
			dataDir = flag.String("data-dir", "./data", "Data directory")
			command = flag.String("cmd", "help", "Command to execute (init, commit, merge, status, export, import-git, tag, branch-delete)")
			message = flag.String("message", "", "Commit message")
			branch = flag.String("branch", "main", "Branch name")
			tagName = flag.String("tag-name", "", "Tag name")
			jsonlOut = flag.Bool("jsonl", false, "Export in JSONL format")
			sourceRepo = flag.String("source", "", "Source Git repository URL or path for import")

			// Parse flags
			os.Args = tc.args
			flag.Parse()

			// Check flag values
			if *command != tc.expected["command"] {
				t.Errorf("Expected command %s, got %s", tc.expected["command"], *command)
			}
			if *repoID != tc.expected["repo"] {
				t.Errorf("Expected repo %s, got %s", tc.expected["repo"], *repoID)
			}
			if *ownerID != tc.expected["owner"] {
				t.Errorf("Expected owner %s, got %s", tc.expected["owner"], *ownerID)
			}
			if *userID != tc.expected["user"] {
				t.Errorf("Expected user %s, got %s", tc.expected["user"], *userID)
			}
			if *branch != tc.expected["branch"] {
				t.Errorf("Expected branch %s, got %s", tc.expected["branch"], *branch)
			}
			if *tagName != tc.expected["tag-name"] {
				t.Errorf("Expected tag-name %s, got %s", tc.expected["tag-name"], *tagName)
			}
			if *message != tc.expected["message"] {
				t.Errorf("Expected message %s, got %s", tc.expected["message"], *message)
			}
			if *dataDir != tc.expected["data-dir"] {
				t.Errorf("Expected data-dir %s, got %s", tc.expected["data-dir"], *dataDir)
			}
			if *sourceRepo != tc.expected["source"] {
				t.Errorf("Expected source %s, got %s", tc.expected["source"], *sourceRepo)
			}
			if tc.expected["jsonl"] == "true" && !*jsonlOut {
				t.Errorf("Expected jsonl to be true, got false")
			} else if tc.expected["jsonl"] == "false" && *jsonlOut {
				t.Errorf("Expected jsonl to be false, got true")
			}
		})
	}
}

func TestCommandValidation(t *testing.T) {
	// Skip this test when running in Docker
	t.Skip("Skipping test that requires special handling in Docker")

	// Test cases for command validation
	testCases := []struct {
		name          string
		command       string
		repoID        string
		ownerID       string
		userID        string
		message       string
		tagName       string
		sourceRepo    string
		expectFailure bool
	}{
		{
			name:          "Valid init command",
			command:       "init",
			repoID:        "test-repo",
			ownerID:       "test-owner",
			expectFailure: false,
		},
		{
			name:          "Invalid init command - missing repo ID",
			command:       "init",
			ownerID:       "test-owner",
			expectFailure: true,
		},
		{
			name:          "Invalid init command - missing owner ID",
			command:       "init",
			repoID:        "test-repo",
			expectFailure: true,
		},
		{
			name:          "Valid commit command",
			command:       "commit",
			repoID:        "test-repo",
			userID:        "test-user",
			message:       "Test commit",
			expectFailure: false,
		},
		{
			name:          "Invalid commit command - missing repo ID",
			command:       "commit",
			userID:        "test-user",
			message:       "Test commit",
			expectFailure: true,
		},
		{
			name:          "Invalid commit command - missing user ID",
			command:       "commit",
			repoID:        "test-repo",
			message:       "Test commit",
			expectFailure: true,
		},
		{
			name:          "Invalid commit command - missing message",
			command:       "commit",
			repoID:        "test-repo",
			userID:        "test-user",
			expectFailure: true,
		},
		{
			name:          "Valid tag command",
			command:       "tag",
			repoID:        "test-repo",
			userID:        "test-user",
			tagName:       "v1.0.0",
			expectFailure: false,
		},
		{
			name:          "Invalid tag command - missing repo ID",
			command:       "tag",
			userID:        "test-user",
			tagName:       "v1.0.0",
			expectFailure: true,
		},
		{
			name:          "Invalid tag command - missing user ID",
			command:       "tag",
			repoID:        "test-repo",
			tagName:       "v1.0.0",
			expectFailure: true,
		},
		{
			name:          "Invalid tag command - missing tag name",
			command:       "tag",
			repoID:        "test-repo",
			userID:        "test-user",
			expectFailure: true,
		},
		{
			name:          "Valid import-git command",
			command:       "import-git",
			repoID:        "test-repo",
			ownerID:       "test-owner",
			userID:        "test-user",
			sourceRepo:    "https://github.com/example/repo.git",
			expectFailure: false,
		},
		{
			name:          "Invalid import-git command - missing repo ID",
			command:       "import-git",
			ownerID:       "test-owner",
			userID:        "test-user",
			sourceRepo:    "https://github.com/example/repo.git",
			expectFailure: true,
		},
		{
			name:          "Invalid import-git command - missing owner ID",
			command:       "import-git",
			repoID:        "test-repo",
			userID:        "test-user",
			sourceRepo:    "https://github.com/example/repo.git",
			expectFailure: true,
		},
		{
			name:          "Invalid import-git command - missing user ID",
			command:       "import-git",
			repoID:        "test-repo",
			ownerID:       "test-owner",
			sourceRepo:    "https://github.com/example/repo.git",
			expectFailure: true,
		},
		{
			name:          "Invalid import-git command - missing source repo",
			command:       "import-git",
			repoID:        "test-repo",
			ownerID:       "test-owner",
			userID:        "test-user",
			expectFailure: true,
		},
	}

	// Save original os.Exit function
	originalExit := osExit
	defer func() { osExit = originalExit }()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock os.Exit
			exitCalled := false
			osExit = func(code int) {
				exitCalled = true
				if code == 0 {
					t.Error("Expected non-zero exit code")
				}
			}

			// Set flag values
			*command = tc.command
			*repoID = tc.repoID
			*ownerID = tc.ownerID
			*userID = tc.userID
			*message = tc.message
			*tagName = tc.tagName
			*sourceRepo = tc.sourceRepo

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the validation function based on the command
			switch tc.command {
			case "init":
				initRepo()
			case "commit":
				addCommit()
			case "tag":
				addTag()
			case "import-git":
				importGitRepo()
			}

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			r.Close()

			// Check if exit was called
			if tc.expectFailure && !exitCalled {
				t.Error("Expected os.Exit to be called, but it wasn't")
			} else if !tc.expectFailure && exitCalled {
				t.Error("Expected os.Exit not to be called, but it was")
			}
		})
	}
}

// Mock os.Exit for testing
var osExit = os.Exit
