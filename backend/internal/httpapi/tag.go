package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/tag"
)

func init() {
	// The tag domain package owns these sentinels; their HTTP meaning is
	// registered here, the one place errors become status codes.
	registerErrStatus(tag.ErrNotFound, http.StatusNotFound)
	registerErrStatus(tag.ErrInvalidValue, http.StatusBadRequest)
	registerErrStatus(tag.ErrDuplicateName, http.StatusConflict)
}
