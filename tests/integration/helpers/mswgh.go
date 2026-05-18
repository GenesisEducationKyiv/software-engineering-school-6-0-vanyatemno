package helpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// MSWServer is a tiny MSW-style request-handler registry over httptest.
//
// Usage:
//
//	gh := NewMSWServer(t)
//	gh.Get("/repos/:owner/:repo/releases/latest", func(req Request) Response {
//	    return JSON(200, map[string]any{"tag_name": "v1.0.0"})
//	})
//
// Handlers match by HTTP method + path; ":name" path segments match any
// single non-empty segment. Each request increments a call counter exposed
// via Calls() so tests can assert on cache-miss behavior.
type MSWServer struct {
	t         *testing.T
	server    *httptest.Server
	mu        sync.Mutex
	handlers  []registration
	calls     []Call
	unmatched func(Request) Response
}

type registration struct {
	method   string
	segments []string
	handler  func(Request) Response
}

type Request struct {
	Method string
	Path   string
	Params map[string]string
	Query  map[string][]string
	Header http.Header
}

type Response struct {
	Status  int
	Body    []byte
	Headers map[string]string
}

type Call struct {
	Method string
	Path   string
}

func NewMSWServer(t *testing.T) *MSWServer {
	t.Helper()
	m := &MSWServer{t: t}
	m.unmatched = defaultUnmatched(t)
	m.server = httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(m.server.Close)
	return m
}

// defaultUnmatched records the unexpected request and replies 500 so the
// caller sees a hard failure rather than silently observing a "200 with
// empty body" response. When this fires, expect TWO failures in the test
// output: the t.Errorf below (the cause) and whatever assertion the
// caller had downstream. The first message is the cause.
func defaultUnmatched(t *testing.T) func(Request) Response {
	return func(req Request) Response {
		t.Errorf("msw: unmatched request %s %s (cause; downstream assertions may also fail)", req.Method, req.Path)
		return JSON(http.StatusInternalServerError, map[string]string{"error": "unmatched"})
	}
}

func (m *MSWServer) URL() string { return m.server.URL + "/" }

func (m *MSWServer) Get(pattern string, h func(Request) Response) {
	m.register(http.MethodGet, pattern, h)
}

// FailOnAnyRequest replaces the default unmatched handler with one that
// fails the test for ANY request, matched or not. Use when the test
// requires zero HTTP traffic (e.g., cache-hit assertions).
func (m *MSWServer) FailOnAnyRequest(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = nil
	m.unmatched = func(req Request) Response {
		m.t.Errorf("msw: unexpected request %s %s (%s)", req.Method, req.Path, reason)
		return JSON(http.StatusInternalServerError, map[string]string{"error": reason})
	}
}

func (m *MSWServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = nil
	m.calls = nil
	m.unmatched = defaultUnmatched(m.t)
}

func (m *MSWServer) Calls() []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Call, len(m.calls))
	copy(out, m.calls)
	return out
}

func (m *MSWServer) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *MSWServer) register(method, pattern string, h func(Request) Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, registration{
		method:   method,
		segments: splitPath(pattern),
		handler:  h,
	})
}

func (m *MSWServer) serve(w http.ResponseWriter, r *http.Request) {
	req := Request{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Header: r.Header.Clone(),
	}

	m.mu.Lock()
	m.calls = append(m.calls, Call{Method: r.Method, Path: r.URL.Path})
	handlers := append([]registration(nil), m.handlers...)
	unmatched := m.unmatched
	m.mu.Unlock()

	for _, reg := range handlers {
		params, ok := matchPath(reg.segments, splitPath(r.URL.Path))
		if !ok || reg.method != r.Method {
			continue
		}
		req.Params = params
		writeResp(w, reg.handler(req))
		return
	}
	writeResp(w, unmatched(req))
}

func writeResp(w http.ResponseWriter, resp Response) {
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if len(resp.Body) > 0 {
		_, _ = w.Write(resp.Body)
	}
}

func JSON(status int, payload any) Response {
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{Status: http.StatusInternalServerError, Body: []byte(`{"error":"marshal"}`)}
	}
	return Response{Status: status, Body: body, Headers: map[string]string{"Content-Type": "application/json"}}
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func matchPath(pattern, actual []string) (map[string]string, bool) {
	if len(pattern) != len(actual) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range pattern {
		if strings.HasPrefix(seg, ":") {
			if actual[i] == "" {
				return nil, false
			}
			params[strings.TrimPrefix(seg, ":")] = actual[i]
			continue
		}
		if seg != actual[i] {
			return nil, false
		}
	}
	return params, true
}
