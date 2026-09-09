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

func TestHandlerUpdateRejectsKind(t *testing.T) {
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
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/entries/"+created.ID, strings.NewReader(`{"kind":"balance_adjustment"}`)), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (unknown field kind rejected)", rec.Code)
	}
}

func TestHandlerUpdateMovesEntryToAnotherAccount(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "EUR")
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
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/entries/"+created.ID, rec)
	var updated entry.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.AccountID != "acc2" {
		t.Fatalf("AccountID = %q, want acc2", updated.AccountID)
	}
}

func TestHandlerUpdateMoveToOtherOwnersAccountRejected(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u2", "EUR")
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
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	conforms(t, "PATCH", "/api/entries/"+created.ID, rec)
}

func TestHandlerUpdateMoveToDisabledAccountRejected(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "EUR")
	accounts.disabled["acc2"] = true
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
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	conforms(t, "PATCH", "/api/entries/"+created.ID, rec)
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

func TestHandlerSummary(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "USD")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	for _, body := range []string{
		`{"account_id":"acc1","kind":"transaction","amount":-100,"booking_timestamp":"2024-01-01T00:00:00Z","title":"x","category_id":"cat1"}`,
		`{"account_id":"acc2","kind":"transaction","amount":-20,"booking_timestamp":"2024-01-02T00:00:00Z","title":"y","category_id":"cat1"}`,
		`{"account_id":"acc1","kind":"balance_adjustment","balance":99999,"booking_timestamp":"2024-01-01T00:00:00Z","title":"adj"}`,
	} {
		h.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest("POST", "/api/entries", strings.NewReader(body)), user))
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries/summary?category_id=cat1", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/entries/summary?category_id=cat1", rec)
	var got entry.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 2 {
		t.Fatalf("Count = %d, want 2 (balance adjustment excluded)", got.Count)
	}
	byCurrency := map[string]int64{}
	for _, s := range got.Sums {
		byCurrency[s.Currency] = s.Amount
	}
	if len(byCurrency) != 2 || byCurrency["EUR"] != -100 || byCurrency["USD"] != -20 {
		t.Fatalf("Sums = %+v, want EUR -100 and USD -20", got.Sums)
	}
}

func TestHandlerSummaryRequiresAuth(t *testing.T) {
	h, _, _ := newHandlerFixture()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/entries/summary", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerSummaryWithNoMatches(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries/summary?category_id=cat1", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/entries/summary?category_id=cat1", rec)
	var got entry.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 || len(got.Sums) != 0 {
		t.Fatalf("got = %+v, want empty", got)
	}
}

func TestHandlerListRejectsInvalidCategoryMode(t *testing.T) {
	h, accounts, _ := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries?category_id=cat1&category_mode=bogus", nil), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlerListCategoryExactExcludesDescendants(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("parent")
	categories.add("child")
	categories.children["parent"] = []string{"child"}
	user := auth.User{ID: "u1"}

	h.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest("POST", "/api/entries",
		strings.NewReader(`{"account_id":"acc1","kind":"transaction","amount":1,"booking_timestamp":"2024-01-01T00:00:00Z","title":"parent","category_id":"parent"}`)), user))
	h.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest("POST", "/api/entries",
		strings.NewReader(`{"account_id":"acc1","kind":"transaction","amount":2,"booking_timestamp":"2024-01-02T00:00:00Z","title":"child","category_id":"child"}`)), user))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries?category_id=parent&category_mode=exact", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/entries?category_id=parent&category_mode=exact", rec)
	var page struct {
		Items []entry.Entry `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "parent" {
		t.Fatalf("items = %+v, want just the parent-category entry", page.Items)
	}
}

func TestHandlerBalance(t *testing.T) {
	h, accounts, _ := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	user := auth.User{ID: "u1"}

	body := `{"account_id":"acc1","kind":"balance_adjustment","balance":5000,"booking_timestamp":"2024-01-01T00:00:00Z","title":"Opening"}`
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

func TestHandlerFlowSummaryRequiresAuth(t *testing.T) {
	h, _, _ := newHandlerFixture()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/entries/flow-summary?unit=month&year=2024", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerFlowSummaryMonthly(t *testing.T) {
	h, accounts, categories := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	user := auth.User{ID: "u1"}

	for _, body := range []string{
		`{"account_id":"acc1","kind":"transaction","amount":1000,"booking_timestamp":"2024-01-15T00:00:00Z","title":"x","category_id":"cat1"}`,
		`{"account_id":"acc1","kind":"transaction","amount":-200,"booking_timestamp":"2024-01-16T00:00:00Z","title":"y","category_id":"cat1"}`,
	} {
		h.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest("POST", "/api/entries", strings.NewReader(body)), user))
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries/flow-summary?account_id=acc1&unit=month&year=2024", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/entries/flow-summary?account_id=acc1&unit=month&year=2024", rec)
	var got struct {
		Buckets []entry.FlowBucket `json:"buckets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Buckets) != 12 {
		t.Fatalf("len(Buckets) = %d, want 12", len(got.Buckets))
	}
	jan := got.Buckets[0]
	if jan.Period != "2024-01-01" || len(jan.Income) != 1 || jan.Income[0].Amount != 1000 ||
		len(jan.Outcome) != 1 || jan.Outcome[0].Amount != 200 {
		t.Fatalf("January bucket = %+v", jan)
	}
	for _, b := range got.Buckets[1:] {
		if len(b.Income) != 0 || len(b.Outcome) != 0 {
			t.Fatalf("bucket %+v, want empty (no entries)", b)
		}
	}
}

func TestHandlerFlowSummaryDayUnitRequiresMonth(t *testing.T) {
	h, accounts, _ := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries/flow-summary?unit=day&year=2024", nil), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
}

func TestHandlerFlowSummaryInvalidYearRejected(t *testing.T) {
	h, accounts, _ := newHandlerFixture()
	accounts.add("acc1", "u1", "EUR")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/entries/flow-summary?unit=month&year=notanumber", nil), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
}
