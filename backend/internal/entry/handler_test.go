package entry_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/openapicheck"
	"at.draab/familyfinances/internal/storage/memory"
)

func newHandlerFixture() (http.Handler, *stubAccounts, *stubCategories) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	tags := newStubTags()
	svc := entry.NewService(memory.NewEntryStore(), accounts, categories, tags)
	h := entry.NewHandler(svc, entry.HandlerOptions{RenderError: httpapi.WriteError})
	return h, accounts, categories
}

func withUser(req *http.Request, u auth.User) *http.Request {
	return req.WithContext(auth.WithUser(req.Context(), u))
}

func conforms(t *testing.T, method, target string, rec *httptest.ResponseRecorder) {
	t.Helper()
	openapicheck.AssertResponse(t, method, target, rec.Code, rec.Header(), rec.Body.Bytes())
}

func TestHandlerCreateRequiresAuth(t *testing.T) {
	h, _, _ := newHandlerFixture()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/entries", strings.NewReader(`{}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerCreateGetUpdateDelete(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	body := `{"account_id":"acc1","kind":"transaction","amount":1234,"booking_timestamp":"2024-01-01T00:00:00Z","title":"Coffee","category_id":"cat1"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/entries", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/entries", rec)
	var created entry.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries/"+created.ID, nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/entries/"+created.ID, rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/entries/"+created.ID, strings.NewReader(`{"title":"Coffee and pastry"}`)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/entries/"+created.ID, rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/entries/"+created.ID, nil), user))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}
	conforms(t, "DELETE", "/api/entries/"+created.ID, rec)
}

func TestHandlerUpdateRejectsAccountIDAndKind(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	body := `{"account_id":"acc1","kind":"transaction","amount":1,"booking_timestamp":"2024-01-01T00:00:00Z","title":"X","category_id":"cat1"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/entries", strings.NewReader(body)), user))
	var created entry.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/entries/"+created.ID, strings.NewReader(`{"account_id":"acc2"}`)), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (unknown field account_id rejected)", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/entries/"+created.ID, strings.NewReader(`{"kind":"balance_adjustment"}`)), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (unknown field kind rejected)", rec.Code)
	}
}

func TestHandlerListAndPaginate(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	for _, ts := range []string{"2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z", "2024-01-03T00:00:00Z"} {
		body := `{"account_id":"acc1","kind":"transaction","amount":1,"booking_timestamp":"` + ts + `","title":"X","category_id":"cat1"}`
		h.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest("POST", "/api/entries", strings.NewReader(body)), user))
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries?limit=2", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/entries?limit=2", rec)
	var page struct {
		Items      []entry.Entry `json:"items"`
		NextCursor *string       `json:"next_cursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("page = %+v", page)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries?limit=2&after="+*page.NextCursor, nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("second page status = %d", rec.Code)
	}
	var page2 struct {
		Items      []entry.Entry `json:"items"`
		NextCursor *string       `json:"next_cursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page2); err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 1 || page2.NextCursor != nil {
		t.Fatalf("page2 = %+v", page2)
	}
}

func TestHandlerBalance(t *testing.T) {
	h, accounts, _ := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	user := auth.User{ID: "u1"}

	body := `{"account_id":"acc1","kind":"balance_adjustment","amount":5000,"booking_timestamp":"2024-01-01T00:00:00Z","title":"Opening"}`
	h.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest("POST", "/api/entries", strings.NewReader(body)), user))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/acc1/balance", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("balance status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/accounts/acc1/balance", rec)
	var got struct {
		Balance int64 `json:"balance"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Balance != 5000 {
		t.Fatalf("balance = %d, want 5000", got.Balance)
	}
}
