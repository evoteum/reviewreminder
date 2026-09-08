// Package nudge holds the core review-reminder logic: decide which pull requests
// are due a nudge, send them, and keep the history up to date.
package nudge

import (
	"context"
	"fmt"
	"time"

	"github.com/evoteum/reviewreminder/internal/forge"
	"github.com/evoteum/reviewreminder/internal/history"
	"github.com/evoteum/reviewreminder/internal/messages"
	"github.com/evoteum/reviewreminder/internal/webhook"
)

// Options configures a nudge run.
type Options struct {
	Forge    forge.Client
	Messages messages.Set
	Webhook  webhook.Sender
	History  history.History
	Interval time.Duration
	Now      func() time.Time // defaults to time.Now
}

// Result reports the outcome of a run.
type Result struct {
	History history.History
	Sent    int
}

// Run fetches pull requests awaiting review, nudges those that are due, and
// returns the updated history pruned of pull requests no longer awaiting review.
// A newly seen pull request is nudged immediately; a previously nudged one is
// nudged again only once Interval has elapsed since its last nudge.
func Run(ctx context.Context, o Options) (Result, error) {
	now := o.Now
	if now == nil {
		now = time.Now
	}

	prs, err := o.Forge.AwaitingReview(ctx)
	if err != nil {
		return Result{History: o.History}, err
	}

	updated := o.History.Clone() // preserves records if we fail partway
	next := history.History{}    // only currently-awaiting PRs; prunes the rest
	sent := 0

	for _, pr := range prs {
		entry, seen := updated[pr.URL]
		entry.PRTitle = pr.Title

		due := !seen || now().Unix()-entry.LastNudgedAt >= int64(o.Interval.Seconds())
		if due {
			if err := o.Webhook.Send(ctx, o.Messages.For(entry.NudgeCount), pr.Title, pr.URL); err != nil {
				updated[pr.URL] = entry
				return Result{History: updated, Sent: sent}, fmt.Errorf("nudging %s: %w", pr.URL, err)
			}
			entry.NudgeCount++
			entry.LastNudgedAt = now().Unix()
			sent++
		}

		updated[pr.URL] = entry
		next[pr.URL] = entry
	}

	return Result{History: next, Sent: sent}, nil
}
