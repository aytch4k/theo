package gitimport

import (
	"fmt"
	"time"

	"github.com/gold2th/theo/internal/blockchain/master"
)

// ForkInfo represents information about a repository fork
type ForkInfo struct {
	ParentRepo   string
	ForkRepoName string
	Forker       string
	Timestamp    time.Time
}

// addForkToChain adds a fork to the fork chain
func (g *GitRepoImporter) addForkToChain(forkInfo *ForkInfo) error {
	// Add fork to fork chain
	_, err := g.ForkChain.AddFork(
		forkInfo.ParentRepo,
		forkInfo.ForkRepoName,
		forkInfo.Forker,
	)
	if err != nil {
		return fmt.Errorf("failed to add fork to fork chain: %w", err)
	}

	fmt.Printf("Added fork: %s -> %s by %s\n", forkInfo.ParentRepo, forkInfo.ForkRepoName, forkInfo.Forker)
	// Add the fork action hash to the repo chain
	latestBlock := g.ForkChain.Blocks[len(g.ForkChain.Blocks)-1]
	userData := master.UserData{
		ID: forkInfo.Forker,
	}
	_, err = g.RepoChain.AddActionHash("fork", latestBlock.Hash, userData)
	if err != nil {
		return fmt.Errorf("failed to add fork action hash to repo chain: %w", err)
	}

	return nil
}
