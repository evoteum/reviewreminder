// Package webhook delivers nudges to an HTTP endpoint.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Sender delivers a nudge.
type Sender interface {
	Send(ctx context.Context, message, prTitle, prURL string) error
}

// HTTP posts nudges to a webhook URL as JSON.
type HTTP struct {
	URL    string
	Client *http.Client
}

// NewHTTP builds an HTTP webhook sender.
func NewHTTP(u string) *HTTP {
	return &HTTP{URL: u, Client: &http.Client{Timeout: 30 * time.Second}}
}

type payload struct {
	Message string `json:"message"`
	PRTitle string `json:"pr_title"`
	PRURL   string `json:"pr_url"`
}

// Send posts one nudge to the webhook.
func (h *HTTP) Send(ctx context.Context, message, prTitle, prURL string) error {
	b, err := json.Marshal(payload{Message: message, PRTitle: prTitle, PRURL: prURL})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}
