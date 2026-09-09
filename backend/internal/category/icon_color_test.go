package category_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/storage/memory"
)

func createCategory(t *testing.T, h http.Handler, user auth.User, body string) category.Category {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/categories status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories", rec)
	var c category.Category
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHandlerCreateCategoryWithIconAndColor(t *testing.T) {
	h, _ := newHandler()
	c := createCategory(t, h, auth.User{ID: "u1"}, `{"name":"Groceries","icon":"shopping-cart","color":"orange"}`)
	if c.Icon != "shopping-cart" || c.Color != "orange" {
		t.Fatalf("icon/color = %q/%q, want shopping-cart/orange", c.Icon, c.Color)
	}
}

func TestHandlerCreateCategoryWithoutIconOrColor(t *testing.T) {
	h, _ := newHandler()
	c := createCategory(t, h, auth.User{ID: "u1"}, `{"name":"Groceries"}`)
	if c.Icon != "" || c.Color != "" {
		t.Fatalf("icon/color = %q/%q, want both empty", c.Icon, c.Color)
	}
}

func TestHandlerCreateCategoryMalformedIconRejected(t *testing.T) {
	h, _ := newHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories", strings.NewReader(
		`{"name":"Groceries","icon":"Shopping Cart"}`,
	)), auth.User{ID: "u1"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories", rec)
}

func TestHandlerUpdateCategoryClearsColorKeepsIcon(t *testing.T) {
	h, _ := newHandler()
	user := auth.User{ID: "u1"}
	c := createCategory(t, h, user, `{"name":"Groceries","icon":"shopping-cart","color":"orange"}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/categories/"+c.ID, strings.NewReader(
		`{"color":""}`,
	)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/categories/"+c.ID, rec)
	var got category.Category
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Color != "" {
		t.Fatalf("color = %q after clear, want empty", got.Color)
	}
	if got.Icon != "shopping-cart" {
		t.Fatalf("icon = %q, want shopping-cart (unchanged)", got.Icon)
	}
}

func TestHandlerRenameCategoryPreservesIconAndColor(t *testing.T) {
	h, _ := newHandler()
	user := auth.User{ID: "u1"}
	c := createCategory(t, h, user, `{"name":"Groceries","icon":"shopping-cart","color":"orange"}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/categories/"+c.ID, strings.NewReader(
		`{"name":"Food"}`,
	)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var got category.Category
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Icon != "shopping-cart" || got.Color != "orange" {
		t.Fatalf("icon/color = %q/%q after rename, want shopping-cart/orange", got.Icon, got.Color)
	}
}

func TestSeededCategoriesCarryIconAndColor(t *testing.T) {
	svc := category.NewService(memory.NewCategoryStore())
	if err := svc.SeedDefaults(context.Background(), "u1"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	got, err := svc.List(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no seeded categories")
	}
	for _, c := range got {
		if c.Icon == "" || c.Color == "" {
			t.Fatalf("seeded category %q has icon=%q color=%q, want both non-empty", c.Name, c.Icon, c.Color)
		}
	}
}
