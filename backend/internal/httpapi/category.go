package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/category"
)

func init() {
	// The category domain package owns these sentinels; their HTTP meaning
	// is registered here, the one place errors become status codes.
	registerErrStatus(category.ErrNotFound, http.StatusNotFound)
	registerErrStatus(category.ErrInvalidValue, http.StatusBadRequest)
	registerErrStatus(category.ErrInUse, http.StatusConflict)
	registerErrStatus(category.ErrCycle, http.StatusUnprocessableEntity)
}
