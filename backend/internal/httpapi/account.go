package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/account"
)

func init() {
	// The account domain package owns these sentinels; their HTTP meaning is
	// registered here, the one place errors become status codes.
	registerErrStatus(account.ErrNotFound, http.StatusNotFound)
	registerErrStatus(account.ErrInvalidValue, http.StatusBadRequest)
	registerErrStatus(account.ErrTypeInUse, http.StatusConflict)
	registerErrStatus(account.ErrTypeDisabled, http.StatusUnprocessableEntity)
	registerErrStatus(account.ErrForbidden, http.StatusForbidden)
}
