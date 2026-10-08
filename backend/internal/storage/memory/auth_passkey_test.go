package memory_test

import (
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/storage/memory"
	"at.draab/familyfinances/internal/storage/storetest"
)

func TestMemoryPasskeyContract(t *testing.T) {
	storetest.PasskeyContract(t, func(*testing.T) auth.Store { return memory.NewAuthStore() })
}
