package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/storage/memory"
)

func seedUser(t *testing.T, store *memory.AuthStore, email string) {
	t.Helper()
	if _, _, err := store.CreateUserWithIdentity(context.Background(),
		auth.NewUser{Email: email},
		auth.Identity{Kind: auth.IdentityEmail, Email: email, EmailVerified: true},
	); err != nil {
		t.Fatalf("seed %s: %v", email, err)
	}
}

func runCLI(store auth.Store, args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(context.Background(), args, store, &out, &errb)
	return code, out.String(), errb.String()
}

func TestAdminGrantThenList(t *testing.T) {
	store := memory.NewAuthStore()
	seedUser(t, store, "boot@example.com") // becomes admin via bootstrap
	seedUser(t, store, "alice@example.com")

	if code, _, errOut := runCLI(store, "grant", "Alice@example.com"); code != 0 {
		t.Fatalf("grant exit = %d, stderr %q", code, errOut)
	}

	code, out, _ := runCLI(store, "list")
	if code != 0 {
		t.Fatalf("list exit = %d", code)
	}
	got := strings.Fields(out)
	want := map[string]bool{"boot@example.com": true, "alice@example.com": true}
	if len(got) != 2 || !want[got[0]] || !want[got[1]] {
		t.Fatalf("list output = %q, want the two admin emails", out)
	}
}

func TestAdminRevoke(t *testing.T) {
	store := memory.NewAuthStore()
	seedUser(t, store, "boot@example.com")

	if code, _, errOut := runCLI(store, "revoke", "boot@example.com"); code != 0 {
		t.Fatalf("revoke exit = %d, stderr %q", code, errOut)
	}
	code, out, _ := runCLI(store, "list")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("list after revoke = %q (exit %d), want empty", out, code)
	}
}

func TestAdminUnknownEmailErrors(t *testing.T) {
	store := memory.NewAuthStore()
	seedUser(t, store, "boot@example.com")

	code, _, errOut := runCLI(store, "grant", "nobody@example.com")
	if code == 0 {
		t.Fatal("grant on unknown email exited 0, want non-zero")
	}
	if !strings.Contains(errOut, "no user with email") {
		t.Fatalf("stderr = %q, want a clear unknown-email message", errOut)
	}
	// Nothing was created.
	if _, err := store.UserByEmail(context.Background(), "nobody@example.com"); err == nil {
		t.Fatal("grant created a user for an unknown email")
	}
}

func TestAdminUsage(t *testing.T) {
	store := memory.NewAuthStore()
	for _, args := range [][]string{{}, {"bogus"}, {"grant"}, {"grant", "a", "b"}} {
		if code, _, _ := runCLI(store, args...); code != 2 {
			t.Errorf("args %v exit = %d, want 2 (usage)", args, code)
		}
	}
}
