package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/dashboard"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/openapicheck"
	"at.draab/familyfinances/internal/storage/memory"
)

func newHandlerFixture() (http.Handler, *stubAccounts, *stubCategories, *stubTags) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	tags := newStubTags()
	svc := dashboard.NewService(memory.NewDashboardStore(), accounts, categories, tags)
	h := dashboard.NewHandler(svc, dashboard.HandlerOptions{RenderError: httpapi.WriteError})
	return h, accounts, categories, tags
}

func withUser(req *http.Request, u auth.User) *http.Request {
	return req.WithContext(auth.WithUser(req.Context(), u))
}

func conforms(t *testing.T, method, target string, rec *httptest.ResponseRecorder) {
	t.Helper()
	openapicheck.AssertResponse(t, method, target, rec.Code, rec.Header(), rec.Body.Bytes())
}

func TestHandlerListRequiresAuth(t *testing.T) {
	h, _, _, _ := newHandlerFixture()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/dashboard/cards", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerCreateAndList(t *testing.T) {
	h, accounts, _, _ := newHandlerFixture()
	accounts.add("acc1", "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards",
		strings.NewReader(`{"type":"account_stat","config":{"account_id":"acc1"}}`)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/dashboard/cards", rec)
	if !strings.Contains(rec.Body.String(), `"account_stat"`) {
		t.Fatalf("body = %s, want account_stat", rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/dashboard/cards", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/dashboard/cards", rec)
	if !strings.Contains(rec.Body.String(), `"acc1"`) {
		t.Fatalf("list body = %s, want it to include the created card", rec.Body)
	}
}

func TestHandlerListNeverIncludesAnotherUsersCards(t *testing.T) {
	h, _, _, _ := newHandlerFixture()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards",
		strings.NewReader(`{"type":"query_stat","config":{}}`)), auth.User{ID: "u1"}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/dashboard/cards", nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/dashboard/cards", rec)
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("list body = %s, want empty", rec.Body)
	}
}

func TestHandlerCreateUnknownTypeRejected(t *testing.T) {
	h, _, _, _ := newHandlerFixture()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards",
		strings.NewReader(`{"type":"something_else","config":{}}`)), auth.User{ID: "u1"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/dashboard/cards", rec)
}

func TestHandlerCreateLineChart(t *testing.T) {
	h, accounts, _, _ := newHandlerFixture()
	accounts.add("acc1", "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards",
		strings.NewReader(`{"type":"line_chart","config":{"account_id":"acc1"}}`)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/dashboard/cards", rec)
	if !strings.Contains(rec.Body.String(), `"line_chart"`) {
		t.Fatalf("body = %s, want line_chart", rec.Body)
	}
}

func TestHandlerCreateInaccessibleAccountRejected(t *testing.T) {
	h, accounts, _, _ := newHandlerFixture()
	accounts.add("acc1", "someone-else")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards",
		strings.NewReader(`{"type":"account_stat","config":{"account_id":"acc1"}}`)), auth.User{ID: "u1"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/dashboard/cards", rec)
}

func TestHandlerUpdateOnlyEverChangesConfig(t *testing.T) {
	h, accounts, _, _ := newHandlerFixture()
	accounts.add("acc1", "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards",
		strings.NewReader(`{"type":"account_stat","config":{"account_id":"acc1"}}`)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	id := extractID(t, rec.Body.String())

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/dashboard/cards/"+id,
		strings.NewReader(`{"type":"bar_chart","config":{"account_id":"acc1"}}`)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/dashboard/cards/"+id, rec)
	if !strings.Contains(rec.Body.String(), `"account_stat"`) {
		t.Fatalf("body = %s, want type to remain account_stat", rec.Body)
	}
}

func TestHandlerCrossOwnerActionsNotFound(t *testing.T) {
	h, accounts, _, _ := newHandlerFixture()
	accounts.add("acc1", "u1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards",
		strings.NewReader(`{"type":"account_stat","config":{"account_id":"acc1"}}`)), auth.User{ID: "u1"}))
	id := extractID(t, rec.Body.String())

	other := auth.User{ID: "u2"}
	for _, req := range []*http.Request{
		httptest.NewRequest("PATCH", "/api/dashboard/cards/"+id, strings.NewReader(`{"config":{"account_id":"acc1"}}`)),
		httptest.NewRequest("DELETE", "/api/dashboard/cards/"+id, nil),
		httptest.NewRequest("POST", "/api/dashboard/cards/"+id+"/move-up", nil),
		httptest.NewRequest("POST", "/api/dashboard/cards/"+id+"/move-down", nil),
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withUser(req, other))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want 404, body = %s", req.Method, req.URL.Path, rec.Code, rec.Body)
		}
	}
}

func TestHandlerMoveUpFirstCardIsNoop(t *testing.T) {
	h, _, _, _ := newHandlerFixture()
	user := auth.User{ID: "u1"}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards", strings.NewReader(`{"type":"query_stat","config":{}}`)), user))
	id := extractID(t, rec.Body.String())

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards/"+id+"/move-up", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/dashboard/cards/"+id+"/move-up", rec)
}

func TestHandlerDelete(t *testing.T) {
	h, _, _, _ := newHandlerFixture()
	user := auth.User{ID: "u1"}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/dashboard/cards", strings.NewReader(`{"type":"query_stat","config":{}}`)), user))
	id := extractID(t, rec.Body.String())

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/dashboard/cards/"+id, nil), user))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/dashboard/cards", nil), user))
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("list body = %s, want empty after delete", rec.Body)
	}
}

// extractID pulls the "id" field out of a single-object JSON response body
// — good enough for these tests without pulling in a JSON-path dependency.
func extractID(t *testing.T, body string) string {
	t.Helper()
	const marker = `"id":"`
	i := strings.Index(body, marker)
	if i == -1 {
		t.Fatalf("body = %s, no id field found", body)
	}
	rest := body[i+len(marker):]
	return rest[:strings.Index(rest, `"`)]
}
