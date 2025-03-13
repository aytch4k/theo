package gitimport

import (
	"fmt"
	"time"

	"github.com/gold2th/theo/internal/blockchain/master"
)

// WatchInfo represents information about a repository watch
type WatchInfo struct {
	User      string
	Timestamp time.Time
}

// addWatchToChain adds a watch to the watch chain
func (g *GitRepoImporter) addWatchToChain(watchInfo *WatchInfo) error {
	// Add watch to watch chain
	_, err := g.WatchChain.AddWatch(
		watchInfo.User,
	)
	if err != nil {
		return fmt.Errorf("failed to add watch to watch chain: %w", err)
	}

	fmt.Printf("Added watch by %s\n", watchInfo.User)

	// Add the watch action hash to the repo chain
	latestBlock := g.WatchChain.Blocks[len(g.WatchChain.Blocks)-1]
	userData := master.UserData{
		ID: watchInfo.User,
	}
	_, err = g.RepoChain.AddActionHash("watch", latestBlock.Hash, userData)
	if err != nil {
		return fmt.Errorf("failed to add watch action hash to repo chain: %w", err)
	}

	return nil
}
