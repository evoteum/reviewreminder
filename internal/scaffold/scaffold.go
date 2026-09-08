// Package scaffold creates reviewreminder's starter files.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

// Init creates root (0700) and writes starter config, messages and an empty
// token file (0600), without overwriting any that already exist.
func Init(root string, configYAML, messagesYAML []byte) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	files := []struct {
		name    string
		content []byte
	}{
		{"config.yaml", configYAML},
		{"messages.yaml", messagesYAML},
		{"token", []byte{}},
	}
	for _, f := range files {
		if err := writeIfAbsent(filepath.Join(root, f.name), f.content); err != nil {
			return err
		}
	}
	return nil
}

func writeIfAbsent(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("kept existing %s\n", filepath.Base(path))
		return nil
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	fmt.Printf("created %s\n", filepath.Base(path))
	return nil
}
