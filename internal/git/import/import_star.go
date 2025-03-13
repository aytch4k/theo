package gitimport

import (
	"fmt"
	"time"

	"github.com/gold2th/theo/internal/blockchain/master"
)

// StarInfo represents information about a repository star
type StarInfo struct {
	User      string
	Timestamp time.Time
}

// addStarToChain adds a star to the star chain
func (g *GitRepoImporter) addStarToChain(starInfo *StarInfo) error {
	// Add star to star chain
	_, err := g.StarChain.AddStar(
		starInfo.User,
	)
	if err != nil {
		return fmt.Errorf("failed to add star to star chain: %w", err)
	}

	fmt.Printf("Added star by %s\n", starInfo.User)

	// Add the star action hash to the repo chain
	latestBlock := g.StarChain.Blocks[len(g.StarChain.Blocks)-1]
	userData := master.UserData{
		ID: starInfo.User,
	}
	_, err = g.RepoChain.AddActionHash("star", latestBlock.Hash, userData)
	if err != nil {
		return fmt.Errorf("failed to add star action hash to repo chain: %w", err)
	}

	return nil
}
