package auth_test

import (
	"errors"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
)

func TestNormalizeDisplayName(t *testing.T) {
	cases := map[string]string{
		"  Jane Doe  ": "Jane Doe",
		"\tJane\n":     "Jane",
		"Jane  Doe":    "Jane  Doe", // interior whitespace is left alone
		"":             "",
		"   ":          "",
		"José Müller":  "José Müller",
	}
	for in, want := range cases {
		if got := auth.NormalizeDisplayName(in); got != want {
			t.Errorf("NormalizeDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateDisplayName(t *testing.T) {
	ok := []string{
		"",
		"Jane Doe",
		"O'Brien",
		"José Müller-Vega",
		"Anne-Marie de la Croix",
		"J.R.R. Tolkien",
		strings.Repeat("á", 150),
	}
	for _, s := range ok {
		if err := auth.ValidateDisplayName(s); err != nil {
			t.Errorf("ValidateDisplayName(%q) = %v, want nil", s, err)
		}
	}

	bad := []string{
		"Jane <b>Doe</b>",
		"Doe, Jane",
		"Jane Doe 3",
		"j@ne",
		"emoji 🙂",
		"line\nbreak",
		strings.Repeat("á", 151),
	}
	for _, s := range bad {
		if err := auth.ValidateDisplayName(s); !errors.Is(err, auth.ErrInvalidDisplayName) {
			t.Errorf("ValidateDisplayName(%q) = %v, want ErrInvalidDisplayName", s, err)
		}
	}
}

func TestSanitizeDisplayName(t *testing.T) {
	cases := map[string]string{
		"Jane Doe":         "Jane Doe",
		"  Jane Doe  ":     "Jane Doe",
		"Doe, Jane (Dr.)":  "Doe Jane Dr.",
		"Jane   Doe":       "Jane Doe",
		"Jane\tDoe":        "Jane Doe",
		"J. R. R. Tolkien": "J. R. R. Tolkien",
		"O'Brien":          "O'Brien",
		"José Müller-Vega": "José Müller-Vega",
		"":                 "",
		"12345":            "",
		"你好 世界":            "你好 世界",
	}
	for in, want := range cases {
		if got := auth.SanitizeDisplayName(in); got != want {
			t.Errorf("SanitizeDisplayName(%q) = %q, want %q", in, got, want)
		}
	}

	if got := auth.SanitizeDisplayName(strings.Repeat("á", 200)); len([]rune(got)) != 150 {
		t.Errorf("SanitizeDisplayName truncation: got %d runes, want 150", len([]rune(got)))
	}
}
