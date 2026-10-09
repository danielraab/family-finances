package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/config"
)

func TestRecoverPanicYields500(t *testing.T) {
	var buf bytes.Buffer
	restore := swapDefaultLogger(&buf)
	defer restore()

	h := withAllMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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
	h := withAllMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
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

	h := withAllMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func withAllMiddleware(h http.Handler) http.Handler {
	return withMiddleware(h, config.RequestLogAll)
}

func TestRequestLogModes(t *testing.T) {
	statuses := []int{http.StatusOK, http.StatusFound, http.StatusNotFound, http.StatusTooManyRequests, http.StatusInternalServerError}
	for _, tc := range []struct {
		mode   config.RequestLogMode
		logged map[int]bool
	}{
		{config.RequestLogAll, map[int]bool{200: true, 302: true, 404: true, 429: true, 500: true}},
		{config.RequestLogErrors, map[int]bool{404: true, 429: true, 500: true}},
		{config.RequestLogOff, map[int]bool{}},
	} {
		for _, status := range statuses {
			t.Run(fmt.Sprintf("%s/%d", tc.mode, status), func(t *testing.T) {
				var buf bytes.Buffer
				restore := swapDefaultLogger(&buf)
				defer restore()

				h := withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// A handler's own log line must survive every mode.
					Logger(r.Context()).Info("handler line")
					w.WriteHeader(status)
				}), tc.mode)
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

				out := buf.String()
				if got := strings.Contains(out, `"msg":"request"`); got != tc.logged[status] {
					t.Errorf("request line logged = %v, want %v; output %s", got, tc.logged[status], out)
				}
				if !strings.Contains(out, "handler line") {
					t.Errorf("handler log line missing in mode %s", tc.mode)
				}
			})
		}
	}
}
