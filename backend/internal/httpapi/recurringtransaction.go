package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/recurringtransaction"
)

func init() {
	// The recurringtransaction domain package owns these sentinels; their
	// HTTP meaning is registered here, the one place errors become status
	// codes.
	registerErrStatus(recurringtransaction.ErrNotFound, http.StatusNotFound)
	registerErrStatus(recurringtransaction.ErrInvalidValue, http.StatusBadRequest)
	registerErrStatus(recurringtransaction.ErrAccountDisabled, http.StatusUnprocessableEntity)
	registerErrStatus(recurringtransaction.ErrForbidden, http.StatusForbidden)
	registerErrStatus(recurringtransaction.ErrInUse, http.StatusConflict)
}
