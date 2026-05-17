package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

// MailpitClient talks to Mailpit's HTTP API (default :8025).
type MailpitClient struct {
	baseURL string
	http    *http.Client
}

type mailpitListResponse struct {
	Messages []mailpitMessage `json:"messages"`
}

type mailpitMessage struct {
	ID    string         `json:"ID"`
	To    []mailpitEAddr `json:"To"`
	From  mailpitEAddr   `json:"From"`
	Subject string       `json:"Subject"`
}

type mailpitEAddr struct {
	Address string `json:"Address"`
	Name    string `json:"Name"`
}

type mailpitMessageDetail struct {
	HTML string `json:"HTML"`
	Text string `json:"Text"`
}

// DeleteAll wipes all messages from mailpit.
func (m *MailpitClient) DeleteAll(t *testing.T) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, m.baseURL+"/api/v1/messages", nil)
	resp, err := m.http.Do(req)
	if err != nil {
		t.Fatalf("mailpit delete-all: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// WaitForMessageTo polls mailpit until a message addressed to `email` is
// found or the deadline expires. Returns the parsed HTML body.
func (m *MailpitClient) WaitForMessageTo(t *testing.T, email string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		id, ok := m.findMessageID(t, email)
		if ok {
			return m.fetchHTML(t, id)
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("mailpit: no message to %s within %s", email, timeout)
	return ""
}

func (m *MailpitClient) findMessageID(t *testing.T, email string) (string, bool) {
	t.Helper()
	u := m.baseURL + "/api/v1/search?query=" + url.QueryEscape("to:"+email)
	resp, err := m.http.Get(u)
	if err != nil {
		t.Fatalf("mailpit search: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mailpit search status %d", resp.StatusCode)
	}
	var list mailpitListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("mailpit decode: %v", err)
	}
	if len(list.Messages) == 0 {
		return "", false
	}
	return list.Messages[0].ID, true
}

func (m *MailpitClient) fetchHTML(t *testing.T, id string) string {
	t.Helper()
	resp, err := m.http.Get(m.baseURL + "/api/v1/message/" + id)
	if err != nil {
		t.Fatalf("mailpit get message: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mailpit get message status %d", resp.StatusCode)
	}
	var detail mailpitMessageDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatalf("mailpit decode message: %v", err)
	}
	if detail.HTML != "" {
		return detail.HTML
	}
	return detail.Text
}

// ExtractConfirmToken pulls the confirmation token out of the email HTML.
// The product wires the link as `<frontend>/confirm/<code>`.
func ExtractConfirmToken(html string) (string, error) {
	re := regexp.MustCompile(`/confirm/([A-Za-z0-9._\-]+)`)
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return "", fmt.Errorf("no confirm token in email")
	}
	return strings.TrimSpace(m[1]), nil
}
