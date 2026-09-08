// Package forge reads pull requests awaiting review from a git forge.
package forge

import "context"

// PullRequest is a pull/merge request awaiting review.
type PullRequest struct {
	URL   string
	Title string
}

// Client reads pull requests from a git forge.
type Client interface {
	// AwaitingReview returns the current user's open, non-draft, unapproved
	// pull requests.
	AwaitingReview(ctx context.Context) ([]PullRequest, error)
}
