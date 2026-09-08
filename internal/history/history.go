// Package history persists which pull requests have been nudged, and when.
package history

import (
	"encoding/json"
	"errors"
	"os"
)

// Entry records how a single pull request has been nudged.
type Entry struct {
	PRTitle      string `json:"pr_title"`
	NudgeCount   int    `json:"nudge_count"`
	LastNudgedAt int64  `json:"last_nudged_at"` // Unix seconds, UTC
}

// History maps a pull request URL to its nudge record.
type History map[string]Entry

// Load reads history from path. A missing file yields an empty history.
func Load(path string) (History, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return History{}, nil
	}
	if err != nil {
		return nil, err
	}
	h := History{}
	if len(b) == 0 {
		return h, nil
	}
	if err := json.Unmarshal(b, &h); err != nil {
		return nil, err
	}
	return h, nil
}

// Save writes history to path with owner-only permissions.
func Save(path string, h History) error {
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// Clone returns a shallow copy of the history.
func (h History) Clone() History {
	c := make(History, len(h))
	for k, v := range h {
		c[k] = v
	}
	return c
}
