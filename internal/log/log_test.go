package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestInfoProducesLogfmtWithTimestamp(t *testing.T) {
	buf := new(bytes.Buffer)
	original := Logger()
	ReplaceLogger(slog.New(newHandler(buf)))
	t.Cleanup(func() {
		ReplaceLogger(original)
	})

	Info(context.Background(), "hello", "user", "test")

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatalf("expected log output, got empty string")
	}
	if !strings.Contains(line, "ts=") {
		t.Fatalf("expected timestamp field in log line, got %q", line)
	}
	if !strings.Contains(line, "level=info") {
		t.Fatalf("expected level field in log line, got %q", line)
	}
	if !strings.Contains(line, "msg=hello") {
		t.Fatalf("expected message field in log line, got %q", line)
	}
	if !strings.Contains(line, "user=test") {
		t.Fatalf("expected structured field in log line, got %q", line)
	}
}
