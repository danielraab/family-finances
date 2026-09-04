package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/settings"
)

func init() {
	// The settings domain package owns this sentinel; its HTTP meaning is
	// registered here, the one place errors become status codes.
	registerErrStatus(settings.ErrInvalidValue, http.StatusBadRequest)
}
