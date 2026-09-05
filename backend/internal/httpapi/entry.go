package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/entry"
)

func init() {
	// The entry domain package owns these sentinels; their HTTP meaning is
	// registered here, the one place errors become status codes.
	registerErrStatus(entry.ErrNotFound, http.StatusNotFound)
	registerErrStatus(entry.ErrInvalidValue, http.StatusBadRequest)
	registerErrStatus(entry.ErrAccountDisabled, http.StatusUnprocessableEntity)
}
