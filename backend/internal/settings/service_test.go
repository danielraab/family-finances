package settings_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/settings"
	"at.draab/familyfinances/internal/storage/memory"
)

func ptr(s string) *string { return &s }

func TestGetResolvesDefaultsForMissingRow(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())

	got, err := svc.Get(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	want := settings.Settings{Language: "en", Timezone: "UTC", DefaultCurrency: "EUR", DisplayedDecimalPlaces: 2}
	if got != want {
		t.Fatalf("Get = %+v, want %+v", got, want)
	}
}

func TestUpdateOnlyChangesProvidedField(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	ctx := context.Background()

	if _, err := svc.Update(ctx, "u1", settings.Update{Language: ptr("de")}); err != nil {
		t.Fatalf("first update: %v", err)
	}
	got, err := svc.Update(ctx, "u1", settings.Update{Timezone: ptr("Europe/Vienna")})
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	want := settings.Settings{Language: "de", Timezone: "Europe/Vienna", DefaultCurrency: "EUR", DisplayedDecimalPlaces: 2}
	if got != want {
		t.Fatalf("Get after two updates = %+v, want %+v", got, want)
	}
}

func TestUpdateRejectsInvalidLanguage(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	if _, err := svc.Update(context.Background(), "u1", settings.Update{Language: ptr("fr")}); !errors.Is(err, settings.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateRejectsInvalidTimezone(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	if _, err := svc.Update(context.Background(), "u1", settings.Update{Timezone: ptr("Not/AZone")}); !errors.Is(err, settings.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateRejectsMalformedCurrency(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	for _, bad := range []string{"usd", "US", "1234"} {
		if _, err := svc.Update(context.Background(), "u1", settings.Update{DefaultCurrency: ptr(bad)}); !errors.Is(err, settings.ErrInvalidValue) {
			t.Fatalf("currency %q: err = %v, want ErrInvalidValue", bad, err)
		}
	}
}

func TestUpdateRejectsOutOfRangeDisplayedDecimalPlaces(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	for _, bad := range []int{-1, 5} {
		if _, err := svc.Update(context.Background(), "u1", settings.Update{DisplayedDecimalPlaces: ptrInt(bad)}); !errors.Is(err, settings.ErrInvalidValue) {
			t.Fatalf("displayed_decimal_places %d: err = %v, want ErrInvalidValue", bad, err)
		}
	}
}

func TestUpdateDisplayedDecimalPlaces(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	got, err := svc.Update(context.Background(), "u1", settings.Update{DisplayedDecimalPlaces: ptrInt(0)})
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayedDecimalPlaces != 0 {
		t.Fatalf("DisplayedDecimalPlaces = %d, want 0", got.DisplayedDecimalPlaces)
	}
}

func ptrInt(v int) *int { return &v }

func TestInvalidFieldRejectsWholeUpdate(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	ctx := context.Background()

	if _, err := svc.Update(ctx, "u1", settings.Update{Language: ptr("de"), DefaultCurrency: ptr("nope")}); err == nil {
		t.Fatal("expected an error")
	}
	got, err := svc.Get(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Language != "en" {
		t.Fatalf("Language = %q, want unchanged default %q", got.Language, "en")
	}
}

func TestLanguageReturnsRawUnresolvedValue(t *testing.T) {
	svc := settings.NewService(memory.NewSettingsStore())
	ctx := context.Background()

	lang, err := svc.Language(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if lang != nil {
		t.Fatalf("Language for unset user = %v, want nil (not defaulted)", *lang)
	}

	if _, err := svc.Update(ctx, "u1", settings.Update{Language: ptr("de")}); err != nil {
		t.Fatal(err)
	}
	lang, err = svc.Language(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if lang == nil || *lang != "de" {
		t.Fatalf("Language = %v, want \"de\"", lang)
	}
}
