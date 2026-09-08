// Command rr nudges for reviews on your open pull requests.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/evoteum/reviewreminder/internal/config"
	"github.com/evoteum/reviewreminder/internal/forge"
	"github.com/evoteum/reviewreminder/internal/history"
	"github.com/evoteum/reviewreminder/internal/messages"
	"github.com/evoteum/reviewreminder/internal/nudge"
	"github.com/evoteum/reviewreminder/internal/paths"
	"github.com/evoteum/reviewreminder/internal/scaffold"
	"github.com/evoteum/reviewreminder/internal/schedule"
	"github.com/evoteum/reviewreminder/internal/webhook"
)

//go:embed examples/config.yaml
var exampleConfig []byte

//go:embed examples/messages.yaml
var exampleMessages []byte

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "init":
		err = runInit()
	case "", "run":
		err = runNudge(context.Background())
	case "version", "-v", "--version":
		fmt.Println("rr " + version)
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "rr: "+err.Error())
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`reviewreminder (rr) — nudge for reviews on your open pull requests

Usage:
  rr init      Create ~/.reviewreminder with starter config, messages and token files
  rr           Check your open pull requests and nudge those still awaiting review
  rr version   Print the version`)
}

func runInit() error {
	if err := scaffold.Init(paths.Root(), exampleConfig, exampleMessages); err != nil {
		return err
	}
	fmt.Printf("\nEdit %s and put your access token in %s, then run: rr\n",
		paths.Config(), paths.Token())
	return nil
}

// reconcileSchedule keeps the crontab in step with config; it is best-effort and
// never blocks nudging.
func reconcileSchedule(cfg *config.Config) {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "rr: schedule: "+err.Error())
		return
	}
	cmd := fmt.Sprintf("%s run >> %s 2>&1", exe, filepath.Join(paths.Root(), "cron.log"))
	sched := &schedule.Crontab{Command: cmd}
	if err := schedule.Reconcile(sched, cfg.Schedule.Enabled, cfg.Schedule.Expression); err != nil {
		fmt.Fprintln(os.Stderr, "rr: schedule: "+err.Error())
	}
}

func runNudge(ctx context.Context) error {
	cfg, err := config.Load(paths.Config())
	if err != nil {
		return fmt.Errorf("%w (run `rr init` first?)", err)
	}

	// Keep the automatic-run schedule in step with the configuration.
	reconcileSchedule(cfg)

	if cfg.WebhookURL == "" {
		return fmt.Errorf("webhook_url is not set in %s", paths.Config())
	}
	token, err := cfg.Token()
	if err != nil {
		return err
	}
	interval, err := cfg.Interval()
	if err != nil {
		return err
	}

	var fc forge.Client
	switch cfg.Forge.Type {
	case "gitlab":
		fc = forge.NewGitLab(cfg.Forge.URL, cfg.Forge.Username, token)
	default:
		return fmt.Errorf("unsupported forge type %q (only \"gitlab\" is implemented)", cfg.Forge.Type)
	}

	defaults, err := messages.Parse(exampleMessages)
	if err != nil {
		return err
	}
	msgs, err := messages.Load(paths.Messages(), defaults)
	if err != nil {
		return err
	}

	hist, err := history.Load(paths.History())
	if err != nil {
		return err
	}

	res, runErr := nudge.Run(ctx, nudge.Options{
		Forge:    fc,
		Messages: msgs,
		Webhook:  webhook.NewHTTP(cfg.WebhookURL),
		History:  hist,
		Interval: interval,
	})
	// Persist whatever history we have, even on partial failure.
	if err := history.Save(paths.History(), res.History); err != nil && runErr == nil {
		return err
	}
	if runErr != nil {
		return runErr
	}

	fmt.Printf("sent %d nudge(s)\n", res.Sent)
	return nil
}
