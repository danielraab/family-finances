package recurringtransaction_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/openapicheck"
	rt "at.draab/familyfinances/internal/recurringtransaction"
	"at.draab/familyfinances/internal/storage/memory"
)

func newHandlerFixture() (http.Handler, *stubAccounts, *stubCategories) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	tags := newStubTags()
	svc := rt.NewService(memory.NewRecurringTransactionStore(), accounts, categories, tags)
	svc.SetEntryLookup(newStubEntries())
	h := rt.NewHandler(svc, rt.HandlerOptions{RenderError: httpapi.WriteError})
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
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/recurring-transactions", strings.NewReader(`{}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerCreateGetUpdateDelete(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	body := `{"account_id":"acc1","title":"Netflix","category_id":"cat1","amount":-1500,"interval_unit":"month","interval_count":1,"starts_on":"2026-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/recurring-transactions", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/recurring-transactions", rec)
	var created rt.RecurringTransaction
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.PerYearAmount != -18000 {
		t.Fatalf("per_year_amount = %d, want -18000", created.PerYearAmount)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/recurring-transactions/"+created.ID, nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/recurring-transactions/"+created.ID, rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/recurring-transactions/"+created.ID, strings.NewReader(`{"title":"Netflix Premium"}`)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/recurring-transactions/"+created.ID, rec)
	var updated rt.RecurringTransaction
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Netflix Premium" {
		t.Fatalf("title = %q, want Netflix Premium", updated.Title)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/recurring-transactions/"+created.ID, nil), user))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/recurring-transactions/"+created.ID, nil), user))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", rec.Code)
	}
	conforms(t, "GET", "/api/recurring-transactions/"+created.ID, rec)
}

func TestHandlerListAndSummary(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	body := `{"account_id":"acc1","title":"Rent","category_id":"cat1","amount":-80000,"interval_unit":"month","interval_count":1,"starts_on":"2026-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/recurring-transactions", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/recurring-transactions", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/recurring-transactions", rec)
	var items []rt.RecurringTransaction
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("list = %d items, want 1", len(items))
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/recurring-transactions/summary", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/recurring-transactions/summary", rec)
	var sum rt.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &sum); err != nil {
		t.Fatal(err)
	}
	if sum.Count != 1 || len(sum.Sums) != 1 || sum.Sums[0].Currency != "EUR" || sum.Sums[0].Amount != -960000 {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestHandlerDeleteBlockedWhileLinked(t *testing.T) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	entries := newStubEntries()
	svc := rt.NewService(memory.NewRecurringTransactionStore(), accounts, categories, newStubTags())
	svc.SetEntryLookup(entries)
	h := rt.NewHandler(svc, rt.HandlerOptions{RenderError: httpapi.WriteError})

	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	body := `{"account_id":"acc1","title":"Rent","category_id":"cat1","amount":-80000,"interval_unit":"month","interval_count":1,"starts_on":"2026-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/recurring-transactions", strings.NewReader(body)), user))
	var created rt.RecurringTransaction
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	entries.count[created.ID] = 1
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/recurring-transactions/"+created.ID, nil), user))
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete status = %d, want 409", rec.Code)
	}
	conforms(t, "DELETE", "/api/recurring-transactions/"+created.ID, rec)
}
