// Package paths resolves the locations of reviewreminder's files.
package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// Root is the reviewreminder directory, ~/.reviewreminder.
func Root() string { return filepath.Join(home(), ".reviewreminder") }

// Config is the path to config.yaml.
func Config() string { return filepath.Join(Root(), "config.yaml") }

// Messages is the path to messages.yaml.
func Messages() string { return filepath.Join(Root(), "messages.yaml") }

// History is the path to history.json.
func History() string { return filepath.Join(Root(), "history.json") }

// Token is the default path to the access-token file.
func Token() string { return filepath.Join(Root(), "token") }

// Expand resolves a leading ~ to the user's home directory.
func Expand(p string) string {
	if p == "~" {
		return home()
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home(), p[2:])
	}
	return p
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}
