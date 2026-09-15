package recurringtransaction_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

func TestHandlerPreviewRequiresCutoff(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/recurring-transactions/preview", nil), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	conforms(t, "GET", "/api/recurring-transactions/preview", rec)
}

func TestHandlerPreviewRequiresAuth(t *testing.T) {
	h, _, _ := newHandlerFixture()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/recurring-transactions/preview?to=2027-01-01T00:00:00Z", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerPreviewReturnsUpcomingOccurrences(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	starts := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	body := `{"account_id":"acc1","title":"Rent","category_id":"cat1","amount":-80000,"interval_unit":"month","interval_count":1,"starts_on":"` + starts + `"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/recurring-transactions", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}

	to := time.Now().AddDate(0, 2, 0).Format(time.RFC3339)
	target := "/api/recurring-transactions/preview?to=" + url.QueryEscape(to)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", target, nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/recurring-transactions/preview", rec)

	var page struct {
		Items []rt.PreviewItem `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 {
		t.Fatal("expected at least one previewed occurrence")
	}
	if page.Items[0].AccountCurrency != "EUR" || page.Items[0].Overdue {
		t.Fatalf("unexpected first item: %+v", page.Items[0])
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
