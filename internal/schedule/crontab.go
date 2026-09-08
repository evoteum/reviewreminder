package schedule

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// marker tags the crontab line reviewreminder manages, so it never disturbs the
// user's own entries.
const marker = "# reviewreminder (managed - do not edit)"

// Crontab is a Scheduler backed by the user's crontab.
type Crontab struct {
	// Command is the command line to run on schedule, e.g.
	// "/usr/local/bin/rr run >> ~/.reviewreminder/cron.log 2>&1".
	Command string
}

// Current returns the expression of the managed crontab line, if any.
func (c *Crontab) Current() (string, error) {
	lines, err := c.read()
	if err != nil {
		return "", err
	}
	for _, l := range lines {
		if strings.Contains(l, marker) {
			if f := strings.Fields(l); len(f) >= 5 {
				return strings.Join(f[:5], " "), nil
			}
		}
	}
	return "", nil
}

// Ensure installs or replaces the managed crontab line.
func (c *Crontab) Ensure(expr string) error {
	lines, err := c.read()
	if err != nil {
		return err
	}
	kept := withoutMarker(lines)
	kept = append(kept, expr+" "+c.Command+" "+marker)
	return c.write(kept)
}

// Remove deletes the managed crontab line, leaving other entries intact.
func (c *Crontab) Remove() error {
	lines, err := c.read()
	if err != nil {
		return err
	}
	return c.write(withoutMarker(lines))
}

func withoutMarker(lines []string) []string {
	kept := make([]string, 0, len(lines))
	for _, l := range lines {
		if !strings.Contains(l, marker) {
			kept = append(kept, l)
		}
	}
	return kept
}

func (c *Crontab) read() ([]string, error) {
	cmd := exec.Command("crontab", "-l")
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if strings.Contains(errb.String(), "no crontab") {
			return nil, nil // no crontab yet is not an error
		}
		return nil, fmt.Errorf("crontab -l: %v: %s", err, strings.TrimSpace(errb.String()))
	}
	var lines []string
	for _, l := range strings.Split(out.String(), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}

func (c *Crontab) write(lines []string) error {
	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("crontab -: %v: %s", err, strings.TrimSpace(errb.String()))
	}
	return nil
}
