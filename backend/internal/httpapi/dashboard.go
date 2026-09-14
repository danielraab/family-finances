package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/dashboard"
)

func init() {
	// The dashboard domain package owns these sentinels; their HTTP
	// meaning is registered here, the one place errors become status
	// codes.
	registerErrStatus(dashboard.ErrNotFound, http.StatusNotFound)
	registerErrStatus(dashboard.ErrInvalidValue, http.StatusBadRequest)
}
