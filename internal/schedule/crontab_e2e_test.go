package schedule

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCrontabRealE2E exercises the real `crontab` command. It is guarded by
// RR_CRON_E2E=1 because it mutates the invoking user's crontab (restoring it
// afterwards) and is therefore unsuitable for ordinary `go test` runs.
//
//	RR_CRON_E2E=1 go test ./internal/schedule -run TestCrontabRealE2E -v
func TestCrontabRealE2E(t *testing.T) {
	if os.Getenv("RR_CRON_E2E") == "" {
		t.Skip("set RR_CRON_E2E=1 to run the real crontab test")
	}

	original, hadCrontab := currentCrontab(t)
	t.Cleanup(func() { restoreCrontab(t, original, hadCrontab) })

	// Seed an unrelated user entry we must never disturb.
	const userLine = "0 0 * * * /bin/true # my own job"
	setCrontab(t, userLine+"\n")

	c := &Crontab{Command: "/tmp/rr-e2e run"}

	// Install.
	if err := c.Ensure("*/5 * * * *"); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if got, _ := c.Current(); got != "*/5 * * * *" {
		t.Fatalf("after install, Current = %q, want %q", got, "*/5 * * * *")
	}
	assertContains(t, "user entry preserved", userLine)
	assertMarkerCount(t, 1)

	// Update replaces, does not duplicate.
	if err := c.Ensure("0 9 * * *"); err != nil {
		t.Fatalf("Ensure update: %v", err)
	}
	if got, _ := c.Current(); got != "0 9 * * *" {
		t.Fatalf("after update, Current = %q, want %q", got, "0 9 * * *")
	}
	assertMarkerCount(t, 1)
	assertContains(t, "user entry preserved after update", userLine)

	// Remove leaves the user entry intact.
	if err := c.Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got, _ := c.Current(); got != "" {
		t.Fatalf("after remove, Current = %q, want empty", got)
	}
	assertMarkerCount(t, 0)
	assertContains(t, "user entry survives removal", userLine)
}

func currentCrontab(t *testing.T) (string, bool) {
	t.Helper()
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func setCrontab(t *testing.T, content string) {
	t.Helper()
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		t.Fatalf("seeding crontab: %v", err)
	}
}

func restoreCrontab(t *testing.T, original string, had bool) {
	t.Helper()
	if had {
		setCrontab(t, original)
		return
	}
	_ = exec.Command("crontab", "-r").Run() // best effort back to "no crontab"
}

func assertMarkerCount(t *testing.T, want int) {
	t.Helper()
	out, _ := exec.Command("crontab", "-l").Output()
	if got := strings.Count(string(out), marker); got != want {
		t.Fatalf("managed marker count = %d, want %d\ncrontab:\n%s", got, want, out)
	}
}

func assertContains(t *testing.T, what, substr string) {
	t.Helper()
	out, _ := exec.Command("crontab", "-l").Output()
	if !strings.Contains(string(out), substr) {
		t.Fatalf("%s: crontab missing %q\ncrontab:\n%s", what, substr, out)
	}
}
