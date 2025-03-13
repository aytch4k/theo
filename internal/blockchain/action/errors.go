package action

import (
	"errors"
)

// Error definitions
var (
	// Common errors
	ErrBlockNotFound = errors.New("block not found")
	ErrEmptyChain    = errors.New("chain is empty")

	// PR-specific errors
	ErrPRNotFound      = errors.New("pull request not found")
	ErrPRMergeNotFound = errors.New("pull request merge not found")
	ErrPRCloseNotFound = errors.New("pull request close not found")
)
