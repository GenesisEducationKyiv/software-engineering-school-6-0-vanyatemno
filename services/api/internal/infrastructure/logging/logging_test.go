package logging_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"se-school/internal/config"
	"se-school/internal/infrastructure/logging"

	"go.uber.org/zap"
)

func defaultConfig() *config.Log {
	return &config.Log{
		Level:       "info",
		Encoding:    "json",
		Environment: "test",
		ServiceName: "se-school",
		Version:     "test-version",
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns whatever
// was written. Init is called inside fn so that the logger binds to the pipe.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w

	fn()

	if cerr := w.Close(); cerr != nil {
		t.Fatalf("close writer: %v", cerr)
	}
	os.Stdout = orig

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}

	return string(out)
}

func TestInitEmitsECSJSON(t *testing.T) {
	var logger *zap.Logger

	out := captureStdout(t, func() {
		var err error
		logger, err = logging.Init(defaultConfig())
		if err != nil {
			t.Fatalf("Init returned error: %v", err)
		}

		logger.Info("hello world", zap.String("custom.field", "value"))
		_ = logger.Sync()
	})

	line := strings.TrimSpace(out)
	if line == "" {
		t.Fatal("no log output captured")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, line)
	}

	assertField(t, entry, "message", "hello world")
	assertField(t, entry, "log.level", "info")
	assertField(t, entry, "service.name", "se-school")
	assertField(t, entry, "service.version", "test-version")
	assertField(t, entry, "service.environment", "test")
	assertField(t, entry, "custom.field", "value")

	if _, ok := entry["@timestamp"]; !ok {
		t.Errorf("expected @timestamp field, got keys: %v", keys(entry))
	}
}

func TestInitRespectsLevel(t *testing.T) {
	cfg := defaultConfig()
	cfg.Level = "error"

	var logger *zap.Logger

	out := captureStdout(t, func() {
		var err error
		logger, err = logging.Init(cfg)
		if err != nil {
			t.Fatalf("Init returned error: %v", err)
		}

		logger.Info("should be filtered out")
		_ = logger.Sync()
	})

	if strings.TrimSpace(out) != "" {
		t.Errorf("info log should be suppressed at error level, got: %q", out)
	}
}

func TestInitInvalidLevel(t *testing.T) {
	cfg := defaultConfig()
	cfg.Level = "not-a-level"

	if _, err := logging.Init(cfg); err == nil {
		t.Fatal("expected error for invalid log level, got nil")
	}
}

func TestFromContextFallsBackToGlobal(t *testing.T) {
	if logging.FromContext(context.Background()) == nil {
		t.Fatal("FromContext must never return nil")
	}
}

func TestContextWithLoggerRoundTrips(t *testing.T) {
	logger, err := logging.Init(defaultConfig())
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	ctx := logging.ContextWithLogger(context.Background(), logger)
	if got := logging.FromContext(ctx); got != logger {
		t.Errorf("FromContext returned a different logger than was stored")
	}
}

func assertField(t *testing.T, entry map[string]any, key, want string) {
	t.Helper()

	got, ok := entry[key]
	if !ok {
		t.Errorf("missing field %q; got keys: %v", key, keys(entry))
		return
	}

	if got != want {
		t.Errorf("field %q = %v, want %q", key, got, want)
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
