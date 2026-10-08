package aaguid

import (
	"encoding/hex"
	"strings"
	"testing"
)

func mustAAGUID(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
	if err != nil || len(b) != 16 {
		t.Fatalf("bad test AAGUID %q", s)
	}
	return b
}

func TestSnapshotIsUsable(t *testing.T) {
	n, err := Count()
	if err != nil {
		t.Fatal(err)
	}
	// Upstream retires the list by emptying it; don't let that snapshot in.
	if n < 20 {
		t.Fatalf("embedded list has %d providers; was a retired ({}) or truncated snapshot copied in?", n)
	}
}

func TestLookupKnownProvider(t *testing.T) {
	p, ok := Lookup(mustAAGUID(t, "ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4"))
	if !ok || p.Name != "Google Password Manager" {
		t.Fatalf("Lookup = %+v, %v; want Google Password Manager", p, ok)
	}
	for _, icon := range []string{p.IconLight, p.IconDark} {
		if !strings.HasPrefix(icon, "data:image/svg+xml;base64,") {
			t.Errorf("icon %.40q is not an SVG data URI", icon)
		}
	}
}

func TestLookupMisses(t *testing.T) {
	for name, id := range map[string][]byte{
		"nil":     nil,
		"zero":    make([]byte, 16),
		"short":   {1, 2, 3},
		"unknown": mustAAGUID(t, "00112233-4455-6677-8899-aabbccddeeff"),
	} {
		if p, ok := Lookup(id); ok {
			t.Errorf("%s: Lookup = %+v, want no provider", name, p)
		}
	}
}

func TestFormatMatchesListKeys(t *testing.T) {
	id := "fbfc3007-154e-4ecc-8c0b-6e020557d7bd"
	if got := format(mustAAGUID(t, id)); got != id {
		t.Fatalf("format = %s, want %s", got, id)
	}
}

func TestOnlySVGDataURIsAreKept(t *testing.T) {
	for in, want := range map[string]string{
		"data:image/svg+xml;base64,PHN2Zz4=": "data:image/svg+xml;base64,PHN2Zz4=",
		"https://tracker.example/icon.svg":   "",
		"data:image/png;base64,iVBORw0KGgo=": "",
		"javascript:alert(1)":                "",
	} {
		if got := svgDataURI(in); got != want {
			t.Errorf("svgDataURI(%q) = %q, want %q", in, got, want)
		}
	}
}
