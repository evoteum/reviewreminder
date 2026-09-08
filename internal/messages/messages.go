// Package messages holds the ordered, escalating nudge messages.
package messages

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// Set is an ordered list of escalating nudge messages.
type Set struct {
	items []string
}

type file struct {
	Messages []string `yaml:"messages"`
}

// Parse reads a messages document.
func Parse(b []byte) (Set, error) {
	var f file
	if err := yaml.Unmarshal(b, &f); err != nil {
		return Set{}, err
	}
	return Set{items: f.Messages}, nil
}

// Load reads messages from path, falling back to defaults when the file is
// absent or defines no messages.
func Load(path string, defaults Set) (Set, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return Set{}, err
	}
	s, err := Parse(b)
	if err != nil {
		return Set{}, err
	}
	if s.Empty() {
		return defaults, nil
	}
	return s, nil
}

// For returns the message to use given how many times the pull request has
// already been nudged. Escalation stops at the final message.
func (s Set) For(alreadyNudged int) string {
	if len(s.items) == 0 {
		return ""
	}
	i := alreadyNudged
	if i >= len(s.items) {
		i = len(s.items) - 1
	}
	if i < 0 {
		i = 0
	}
	return s.items[i]
}

// Empty reports whether the set has no messages.
func (s Set) Empty() bool { return len(s.items) == 0 }
