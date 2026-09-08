package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/evoteum/reviewreminder/internal/config"
	"github.com/evoteum/reviewreminder/internal/forge"
	"github.com/evoteum/reviewreminder/internal/history"
	"github.com/evoteum/reviewreminder/internal/messages"
	"github.com/evoteum/reviewreminder/internal/nudge"
	"github.com/evoteum/reviewreminder/internal/scaffold"
	"github.com/evoteum/reviewreminder/internal/schedule"
)

const genericTitle = "an awaiting pull request"

type mrSpec struct {
	title, url, author, state string
	draft, approved           bool
	iid, project              int
}

type capturedSend struct{ message, prTitle, prURL string }

type fakeWebhook struct{ w *world }

func (f *fakeWebhook) Send(_ context.Context, message, prTitle, prURL string) error {
	f.w.sent = append(f.w.sent, capturedSend{message, prTitle, prURL})
	return nil
}

type fakeScheduler struct{ current string }

func (f *fakeScheduler) Current() (string, error) { return f.current, nil }
func (f *fakeScheduler) Ensure(e string) error    { f.current = e; return nil }
func (f *fakeScheduler) Remove() error            { f.current = ""; return nil }

type world struct {
	author     string
	mrs        []*mrSpec
	urlByTitle map[string]string
	lastTitle  string
	nextIID    int

	now      time.Time
	interval time.Duration
	msgs     messages.Set
	hist     history.History

	server  *httptest.Server
	sent    []capturedSend
	result  nudge.Result
	runErr  error
	ranTool bool

	sched           *fakeScheduler
	schedConfigured bool
	schedEnabled    bool
	schedExpr       string
	schedErr        error

	root        string
	cfgSentinel []byte
	initErr     error
}

func (w *world) reset() {
	m, _ := messages.Parse(exampleMessages)
	*w = world{
		urlByTitle: map[string]string{},
		now:        time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
		interval:   24 * time.Hour,
		hist:       history.History{},
		sched:      &fakeScheduler{},
		msgs:       m,
		nextIID:    1,
	}
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

func (w *world) deriveURL(title string) string {
	return "https://git.example.com/mr/" + slug(title)
}

func (w *world) urlFor(title string) string {
	if u, ok := w.urlByTitle[title]; ok {
		return u
	}
	return w.deriveURL(title)
}

func (w *world) addMR(title, author, state string, draft, approved bool, url string) {
	if url == "" {
		url = w.deriveURL(title)
	}
	w.urlByTitle[title] = url
	w.mrs = append(w.mrs, &mrSpec{
		title: title, url: url, author: author, state: state,
		draft: draft, approved: approved, iid: w.nextIID, project: 1,
	})
	w.nextIID++
	w.lastTitle = title
}

func (w *world) setHistory(title string, count int, at time.Time) {
	url := w.urlFor(title)
	w.urlByTitle[title] = url
	w.hist[url] = history.Entry{PRTitle: title, NudgeCount: count, LastNudgedAt: at.Unix()}
}

// --- Given: pull request states -------------------------------------------

func (w *world) iAmAuthor(name string) error { w.author = name; return nil }

func (w *world) openNonDraftUnapproved(title string) error {
	w.addMR(title, w.author, "opened", false, false, "")
	return nil
}

func (w *world) openApproved(title string) error {
	w.addMR(title, w.author, "opened", false, true, "")
	return nil
}

func (w *world) openDraft(title string) error {
	w.addMR(title, w.author, "opened", true, false, "")
	return nil
}

func (w *world) putBackIntoDraft() error {
	for _, m := range w.mrs {
		if m.title == w.lastTitle {
			m.draft = true
		}
	}
	return nil
}

func (w *world) someoneElseOpen(author, title string) error {
	w.addMR(title, author, "opened", false, false, "")
	return nil
}

func (w *world) mergedPR(title string) error {
	w.addMR(title, w.author, "merged", false, false, "")
	return nil
}

func (w *world) closedPR(title string) error {
	w.addMR(title, w.author, "closed", false, false, "")
	return nil
}

func (w *world) followingAwaiting(t *godog.Table) error {
	for _, row := range t.Rows[1:] {
		w.addMR(strings.TrimSpace(row.Cells[0].Value), w.author, "opened", false, false, "")
	}
	return nil
}

func (w *world) noPRsAwaiting() error { return nil }

func (w *world) awaiting(title string) error {
	w.addMR(title, w.author, "opened", false, false, "")
	return nil
}

func (w *world) awaitingAtURL(title, url string) error {
	w.addMR(title, w.author, "opened", false, false, url)
	return nil
}

func (w *world) notNudgedBefore() error {
	delete(w.hist, w.urlFor(w.lastTitle))
	return nil
}

func (w *world) nudgedOnPreviousRun() error {
	w.setHistory(w.lastTitle, 1, w.now.Add(-1000*time.Hour))
	return nil
}

func (w *world) alreadyNudgedNTimes(n int) error {
	w.setHistory(w.lastTitle, n, w.now.Add(-1000*time.Hour))
	return nil
}

func (w *world) wasNudgedBefore(title string) error {
	w.setHistory(title, 1, w.now.Add(-1000*time.Hour))
	w.lastTitle = title
	return nil
}

func (w *world) itIsNowMerged() error {
	for _, m := range w.mrs {
		if m.title == w.lastTitle {
			m.state = "merged"
			return nil
		}
	}
	w.addMR(w.lastTitle, w.author, "merged", false, false, "")
	return nil
}

func (w *world) hadPRNudgedNTimes(title string, n int) error {
	w.setHistory(title, n, w.now.Add(-1000*time.Hour))
	w.lastTitle = title
	return nil
}

func (w *world) putBackDraftThenReady() error {
	// The draft round-trip already caused the history to be forgotten; the pull
	// request is now ready for review again with no record.
	delete(w.hist, w.urlFor(w.lastTitle))
	w.addMR(w.lastTitle, w.author, "opened", false, false, "")
	return nil
}

// --- Given: throttling -----------------------------------------------------

func (w *world) nudgeIntervalIs(s string) error {
	d, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	w.interval = d
	return nil
}

func (w *world) awaitingNeverNudged() error {
	w.addMR(genericTitle, w.author, "opened", false, false, "")
	return nil
}

func (w *world) awaitingLastNudgedHoursAgo(h int) error {
	w.addMR(genericTitle, w.author, "opened", false, false, "")
	w.setHistory(genericTitle, 1, w.now.Add(-time.Duration(h)*time.Hour))
	return nil
}

// --- Given: scheduling -----------------------------------------------------

func (w *world) schedulingDisabled() error {
	w.schedConfigured, w.schedEnabled, w.schedExpr = true, false, ""
	return nil
}

func (w *world) schedulingEnabledWith(expr string) error {
	w.schedConfigured, w.schedEnabled, w.schedExpr = true, true, expr
	return nil
}

func (w *world) schedulingEnabledNoExpr() error {
	w.schedConfigured, w.schedEnabled, w.schedExpr = true, true, ""
	return nil
}

func (w *world) scheduleExprChangedTo(expr string) error {
	w.schedConfigured, w.schedEnabled, w.schedExpr = true, true, expr
	return nil
}

func (w *world) notCurrentlyScheduled() error {
	w.sched.current = ""
	return nil
}

// scheduledToRunOn is a precondition before the tool runs and an assertion after.
func (w *world) scheduledToRunOn(expr string) error {
	if !w.ranTool {
		w.sched.current = expr
		return nil
	}
	if w.sched.current != expr {
		return fmt.Errorf("expected schedule %q, got %q", expr, w.sched.current)
	}
	return nil
}

// --- Given: init -----------------------------------------------------------

func (w *world) noConfigExists() error {
	dir, err := os.MkdirTemp("", "rr-bdd-")
	if err != nil {
		return err
	}
	w.root = filepath.Join(dir, ".reviewreminder")
	return nil
}

func (w *world) configAlreadyExists() error {
	dir, err := os.MkdirTemp("", "rr-bdd-")
	if err != nil {
		return err
	}
	w.root = filepath.Join(dir, ".reviewreminder")
	if err := os.MkdirAll(w.root, 0o700); err != nil {
		return err
	}
	w.cfgSentinel = []byte("sentinel: do-not-touch\n")
	return os.WriteFile(filepath.Join(w.root, "config.yaml"), w.cfgSentinel, 0o600)
}

// --- When ------------------------------------------------------------------

func (w *world) theNudgeToolRuns() error {
	if w.schedConfigured {
		w.schedErr = schedule.Reconcile(w.sched, w.schedEnabled, w.schedExpr)
	}
	if w.author != "" {
		w.startServer()
		client := forge.NewGitLab(w.server.URL, w.author, "t")
		res, err := nudge.Run(context.Background(), nudge.Options{
			Forge:    client,
			Messages: w.msgs,
			Webhook:  &fakeWebhook{w: w},
			History:  w.hist,
			Interval: w.interval,
			Now:      func() time.Time { return w.now },
		})
		w.result, w.runErr = res, err
	}
	w.ranTool = true
	return nil
}

func (w *world) initialiseConfiguration() error {
	w.initErr = scaffold.Init(w.root, exampleConfig, exampleMessages)
	return w.initErr
}

// --- Then: nudges ----------------------------------------------------------

func (w *world) find(title string) (capturedSend, bool) {
	for _, s := range w.sent {
		if s.prTitle == title {
			return s, true
		}
	}
	return capturedSend{}, false
}

func (w *world) nudgeSentFor(title string) error {
	if w.runErr != nil {
		return w.runErr
	}
	if _, ok := w.find(title); !ok {
		return fmt.Errorf("expected a nudge for %q, got %d sends", title, len(w.sent))
	}
	return nil
}

func (w *world) noNudgeSentFor(title string) error {
	if w.runErr != nil {
		return w.runErr
	}
	if _, ok := w.find(title); ok {
		return fmt.Errorf("expected no nudge for %q", title)
	}
	return nil
}

func (w *world) noNudgeSent() error {
	if w.runErr != nil {
		return w.runErr
	}
	if len(w.sent) != 0 {
		return fmt.Errorf("expected no nudges, got %d", len(w.sent))
	}
	return nil
}

func (w *world) nudgeSentForThat() error {
	if w.runErr != nil {
		return w.runErr
	}
	if len(w.sent) == 0 {
		return errors.New("expected a nudge, got none")
	}
	return nil
}

func (w *world) noNudgeSentForThat() error {
	if w.runErr != nil {
		return w.runErr
	}
	if len(w.sent) != 0 {
		return fmt.Errorf("expected no nudge, got %d", len(w.sent))
	}
	return nil
}

func (w *world) asksForReview(title string) error {
	s, ok := w.find(title)
	if !ok {
		return fmt.Errorf("no nudge sent for %q", title)
	}
	if want := w.msgs.For(0); s.message != want {
		return fmt.Errorf("expected first message %q, got %q", want, s.message)
	}
	return nil
}

func (w *world) notesStillOutstanding(title string) error {
	s, ok := w.find(title)
	if !ok {
		return fmt.Errorf("no nudge sent for %q", title)
	}
	if want := w.msgs.For(1); s.message != want {
		return fmt.Errorf("expected follow-up message %q, got %q", want, s.message)
	}
	return nil
}

func (w *world) finalReminder(title string) error {
	s, ok := w.find(title)
	if !ok {
		return fmt.Errorf("no nudge sent for %q", title)
	}
	if want := w.msgs.For(1 << 30); s.message != want {
		return fmt.Errorf("expected final message %q, got %q", want, s.message)
	}
	return nil
}

func (w *world) includesTitleAndLink(title, url string) error {
	for _, s := range w.sent {
		if s.prTitle == title && s.prURL == url {
			return nil
		}
	}
	return fmt.Errorf("no nudge with title %q and link %q", title, url)
}

func (w *world) rememberedAsNudged(title string) error {
	e, ok := w.result.History[w.urlFor(title)]
	if !ok || e.NudgeCount < 1 {
		return fmt.Errorf("expected %q remembered as nudged", title)
	}
	return nil
}

func (w *world) noLongerRemembered(title string) error {
	if _, ok := w.result.History[w.urlFor(title)]; ok {
		return fmt.Errorf("expected %q to be forgotten", title)
	}
	return nil
}

// --- Then: scheduling ------------------------------------------------------

func (w *world) nudgesNotScheduled() error {
	if w.sched.current != "" {
		return fmt.Errorf("expected no schedule, got %q", w.sched.current)
	}
	return nil
}

func (w *world) toldInvalidCron(expr string) error {
	if w.schedErr == nil || errors.Is(w.schedErr, schedule.ErrExpressionRequired) {
		return fmt.Errorf("expected an invalid-cron error for %q, got %v", expr, w.schedErr)
	}
	return nil
}

func (w *world) toldExprRequired() error {
	if !errors.Is(w.schedErr, schedule.ErrExpressionRequired) {
		return fmt.Errorf("expected ErrExpressionRequired, got %v", w.schedErr)
	}
	return nil
}

// --- Then: init ------------------------------------------------------------

func (w *world) fileCreated(name string) error {
	if _, err := os.Stat(filepath.Join(w.root, name)); err != nil {
		return fmt.Errorf("expected %s to exist: %w", name, err)
	}
	return nil
}

func (w *world) configFileCreated() error   { return w.fileCreated("config.yaml") }
func (w *world) messagesFileCreated() error { return w.fileCreated("messages.yaml") }

func (w *world) tokenFileCreated() error {
	info, err := os.Stat(filepath.Join(w.root, "token"))
	if err != nil {
		return fmt.Errorf("expected token file: %w", err)
	}
	if info.Size() != 0 {
		return fmt.Errorf("expected empty token file, got %d bytes", info.Size())
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		return fmt.Errorf("expected token file mode 0600, got %o", perm)
	}
	return nil
}

func (w *world) schedulingIsDisabled() error {
	cfg, err := config.Load(filepath.Join(w.root, "config.yaml"))
	if err != nil {
		return err
	}
	if cfg.Schedule.Enabled {
		return errors.New("expected scheduling disabled in scaffolded config")
	}
	return nil
}

func (w *world) existingConfigUnchanged() error {
	got, err := os.ReadFile(filepath.Join(w.root, "config.yaml"))
	if err != nil {
		return err
	}
	if string(got) != string(w.cfgSentinel) {
		return errors.New("existing config.yaml was modified")
	}
	return nil
}

// --- server ----------------------------------------------------------------

func (w *world) startServer() {
	byIID := map[int]*mrSpec{}
	for _, m := range w.mrs {
		byIID[m.iid] = m
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/merge_requests", func(rw http.ResponseWriter, r *http.Request) {
		author := r.URL.Query().Get("author_username")
		state := r.URL.Query().Get("state")
		out := []map[string]any{}
		for _, m := range w.mrs {
			if author != "" && m.author != author {
				continue
			}
			if state != "" && m.state != state {
				continue
			}
			out = append(out, map[string]any{
				"iid": m.iid, "project_id": m.project,
				"title": m.title, "web_url": m.url, "draft": m.draft,
			})
		}
		writeJSON(rw, out)
	})
	mux.HandleFunc("/api/v4/projects/", func(rw http.ResponseWriter, r *http.Request) {
		iid := 0
		if i := strings.Index(r.URL.Path, "/merge_requests/"); i >= 0 {
			rest := strings.TrimSuffix(r.URL.Path[i+len("/merge_requests/"):], "/approvals")
			iid, _ = strconv.Atoi(rest)
		}
		approvedBy := []map[string]any{}
		if m, ok := byIID[iid]; ok && m.approved {
			approvedBy = append(approvedBy, map[string]any{"user": map[string]any{"username": "someone.else"}})
		}
		writeJSON(rw, map[string]any{"approved_by": approvedBy})
	})
	w.server = httptest.NewServer(mux)
}

func writeJSON(rw http.ResponseWriter, v any) {
	rw.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(rw).Encode(v)
}

// --- wiring ----------------------------------------------------------------

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &world{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		w.reset()
		return ctx, nil
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if w.server != nil {
			w.server.Close()
			w.server = nil
		}
		return ctx, nil
	})

	// Given: pull request states
	sc.Step(`^I am the pull request author "([^"]*)"$`, w.iAmAuthor)
	sc.Step(`^I have an open, non-draft, unapproved pull request titled "([^"]*)"$`, w.openNonDraftUnapproved)
	sc.Step(`^I have an open pull request titled "([^"]*)" that has already been approved$`, w.openApproved)
	sc.Step(`^I have an open draft pull request titled "([^"]*)"$`, w.openDraft)
	sc.Step(`^it is put back into draft$`, w.putBackIntoDraft)
	sc.Step(`^"([^"]*)" has an open pull request titled "([^"]*)"$`, w.someoneElseOpen)
	sc.Step(`^I have a merged pull request titled "([^"]*)"$`, w.mergedPR)
	sc.Step(`^I have a closed pull request titled "([^"]*)"$`, w.closedPR)
	sc.Step(`^I have the following pull requests awaiting review:$`, w.followingAwaiting)
	sc.Step(`^I have no pull requests awaiting review$`, w.noPRsAwaiting)
	sc.Step(`^I have a pull request titled "([^"]*)" awaiting review$`, w.awaiting)
	sc.Step(`^I have a pull request titled "([^"]*)" at "([^"]*)" awaiting review$`, w.awaitingAtURL)
	sc.Step(`^it has not been nudged before$`, w.notNudgedBefore)
	sc.Step(`^it was nudged on a previous run$`, w.nudgedOnPreviousRun)
	sc.Step(`^it has already been nudged (\d+) times$`, w.alreadyNudgedNTimes)
	sc.Step(`^"([^"]*)" was nudged before$`, w.wasNudgedBefore)
	sc.Step(`^it is now merged$`, w.itIsNowMerged)
	sc.Step(`^I had a pull request titled "([^"]*)" that was nudged (\d+) times$`, w.hadPRNudgedNTimes)
	sc.Step(`^it was put back into draft and then marked ready for review again$`, w.putBackDraftThenReady)

	// Given: throttling
	sc.Step(`^the nudge interval is "([^"]*)"$`, w.nudgeIntervalIs)
	sc.Step(`^I have a pull request awaiting review that has never been nudged$`, w.awaitingNeverNudged)
	sc.Step(`^I have a pull request awaiting review that was last nudged (\d+) hours? ago$`, w.awaitingLastNudgedHoursAgo)

	// Given: scheduling
	sc.Step(`^scheduling is disabled in the configuration$`, w.schedulingDisabled)
	sc.Step(`^scheduling is enabled with the expression "([^"]*)"$`, w.schedulingEnabledWith)
	sc.Step(`^scheduling is enabled with no expression$`, w.schedulingEnabledNoExpr)
	sc.Step(`^the schedule expression is changed to "([^"]*)"$`, w.scheduleExprChangedTo)
	sc.Step(`^nudges are not currently scheduled$`, w.notCurrentlyScheduled)
	sc.Step(`^nudges are scheduled to run on "([^"]*)"$`, w.scheduledToRunOn)

	// Given: init
	sc.Step(`^no reviewreminder configuration exists$`, w.noConfigExists)
	sc.Step(`^a reviewreminder configuration already exists$`, w.configAlreadyExists)

	// When
	sc.Step(`^the nudge tool runs$`, w.theNudgeToolRuns)
	sc.Step(`^I initialise the configuration$`, w.initialiseConfiguration)

	// Then: nudges
	sc.Step(`^a nudge is sent for "([^"]*)"$`, w.nudgeSentFor)
	sc.Step(`^no nudge is sent for "([^"]*)"$`, w.noNudgeSentFor)
	sc.Step(`^no nudge is sent$`, w.noNudgeSent)
	sc.Step(`^a nudge is sent for that pull request$`, w.nudgeSentForThat)
	sc.Step(`^no nudge is sent for that pull request$`, w.noNudgeSentForThat)
	sc.Step(`^the nudge for "([^"]*)" asks for a review$`, w.asksForReview)
	sc.Step(`^the nudge for "([^"]*)" notes that review is still outstanding$`, w.notesStillOutstanding)
	sc.Step(`^the nudge for "([^"]*)" is the final reminder message$`, w.finalReminder)
	sc.Step(`^the nudge for "([^"]*)" includes its title and a link to "([^"]*)"$`, w.includesTitleAndLink)
	sc.Step(`^"([^"]*)" is remembered as nudged$`, w.rememberedAsNudged)
	sc.Step(`^"([^"]*)" is no longer remembered$`, w.noLongerRemembered)

	// Then: scheduling
	sc.Step(`^nudges are not scheduled$`, w.nudgesNotScheduled)
	sc.Step(`^I am told that "([^"]*)" is not a valid cron expression$`, w.toldInvalidCron)
	sc.Step(`^I am told that a schedule expression is required$`, w.toldExprRequired)

	// Then: init
	sc.Step(`^a configuration file is created for me to edit$`, w.configFileCreated)
	sc.Step(`^a messages file is created for me to edit$`, w.messagesFileCreated)
	sc.Step(`^an empty, private token file is created for me to add my access token$`, w.tokenFileCreated)
	sc.Step(`^scheduling is disabled$`, w.schedulingIsDisabled)
	sc.Step(`^my existing configuration is left unchanged$`, w.existingConfigUnchanged)
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
			Strict:   true,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("godog: failing or undefined steps")
	}
}
