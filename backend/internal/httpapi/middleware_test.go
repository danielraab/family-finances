package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverPanicYields500(t *testing.T) {
	var buf bytes.Buffer
	restore := swapDefaultLogger(&buf)
	defer restore()

	h := withMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(nonEmptyLines(buf.String())[0]), &entry); err != nil {
		t.Fatalf("panic log line not JSON: %v", err)
	}
	if entry["msg"] != "panic recovered" {
		t.Fatalf("msg = %v, want %q", entry["msg"], "panic recovered")
	}
	if entry["request_id"] == nil || entry["request_id"] == "" {
		t.Fatalf("panic log missing request_id: %s", buf.String())
	}
}

func TestRequestContextSetsHeaderAndLoggerMatch(t *testing.T) {
	var buf bytes.Buffer
	restore := swapDefaultLogger(&buf)
	defer restore()

	var seenID string
	h := withMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seenID = RequestID(r.Context())
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	header := rec.Header().Get("X-Request-Id")
	if header == "" {
		t.Fatal("X-Request-Id header not set")
	}
	if seenID != header {
		t.Fatalf("RequestID(ctx) = %q, X-Request-Id = %q, want equal", seenID, header)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(nonEmptyLines(buf.String())[0]), &entry); err != nil {
		t.Fatalf("log line not JSON: %v", err)
	}
	if entry["request_id"] != header {
		t.Fatalf("log request_id = %v, want %q", entry["request_id"], header)
	}
}

func TestLogRequestsLogsOnce(t *testing.T) {
	var buf bytes.Buffer
	restore := swapDefaultLogger(&buf)
	defer restore()

	h := withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	lines := nonEmptyLines(buf.String())
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %q", len(lines), buf.String())
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line not JSON: %v", err)
	}
	if entry["msg"] != "request" {
		t.Fatalf("msg = %v, want %q", entry["msg"], "request")
	}
	if entry["status"] != float64(http.StatusTeapot) {
		t.Fatalf("status = %v, want %d", entry["status"], http.StatusTeapot)
	}
	if entry["request_id"] == nil || entry["request_id"] == "" {
		t.Fatalf("missing request_id in %q", lines[0])
	}
}

func swapDefaultLogger(buf *bytes.Buffer) func() {
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	return func() { slog.SetDefault(prev) }
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
