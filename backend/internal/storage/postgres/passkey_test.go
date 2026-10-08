package postgres

import (
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/storage/storetest"
)

func TestPGPasskeyContract(t *testing.T) {
	storetest.PasskeyContract(t, func(t *testing.T) auth.Store {
		store, _ := newAuthStore(t)
		return store
	})
}

func TestPGCleanupContract(t *testing.T) {
	storetest.CleanupContract(t, func(t *testing.T) auth.Store {
		store, _ := newAuthStore(t)
		return store
	})
}
