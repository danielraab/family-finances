package postgres

import (
	"context"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/settings"
)

func newSettingsStore(t *testing.T) (*SettingsStore, *AuthStore) {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewSettingsStore(pool), NewAuthStore(pool)
}

func ptr(s string) *string { return &s }

func TestPGSettingsGetMissingRowIsZeroValue(t *testing.T) {
	store, authStore := newSettingsStore(t)
	ctx := context.Background()

	u, _, err := authStore.CreateUserWithIdentity(ctx, auth.NewUser{Email: "u@example.com"}, emailIdentity("u@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	row, err := store.Get(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.Language != nil || row.Timezone != nil || row.DefaultCurrency != nil {
		t.Fatalf("row = %+v, want all nil", row)
	}
}

func TestPGSettingsUpsertMergesPartialUpdates(t *testing.T) {
	store, authStore := newSettingsStore(t)
	ctx := context.Background()

	u, _, err := authStore.CreateUserWithIdentity(ctx, auth.NewUser{Email: "u@example.com"}, emailIdentity("u@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Upsert(ctx, u.ID, settings.Update{Language: ptr("de")}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	row, err := store.Upsert(ctx, u.ID, settings.Update{Timezone: ptr("Europe/Vienna")})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if row.Language == nil || *row.Language != "de" {
		t.Fatalf("Language = %v, want \"de\" (untouched by second upsert)", row.Language)
	}
	if row.Timezone == nil || *row.Timezone != "Europe/Vienna" {
		t.Fatalf("Timezone = %v, want \"Europe/Vienna\"", row.Timezone)
	}
	if row.DefaultCurrency != nil {
		t.Fatalf("DefaultCurrency = %v, want nil (never set)", row.DefaultCurrency)
	}
}

func TestPGSettingsLanguageCheckConstraint(t *testing.T) {
	store, authStore := newSettingsStore(t)
	ctx := context.Background()

	u, _, err := authStore.CreateUserWithIdentity(ctx, auth.NewUser{Email: "u@example.com"}, emailIdentity("u@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Upsert(ctx, u.ID, settings.Update{Language: ptr("fr")}); err == nil {
		t.Fatal("expected the DB CHECK constraint to reject an unsupported language")
	}
}

func TestPGSettingsCascadesOnUserDelete(t *testing.T) {
	store, authStore := newSettingsStore(t)
	ctx := context.Background()

	u, _, err := authStore.CreateUserWithIdentity(ctx, auth.NewUser{Email: "u@example.com"}, emailIdentity("u@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upsert(ctx, u.ID, settings.Update{Language: ptr("de")}); err != nil {
		t.Fatal(err)
	}

	if _, err := authStore.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, u.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	row, err := store.Get(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.Language != nil {
		t.Fatalf("row survived user deletion: %+v, want ON DELETE CASCADE to have removed it", row)
	}
}
