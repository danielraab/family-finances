// Package openapicheck is test-support: it validates recorded HTTP responses
// against the hand-written contract in openapi/openapi.yaml (via the synced
// backend/openapi.yaml copy).
//
// It is imported only by _test.go files. A guard test
// (TestServerBinaryExcludesOpenAPICheck) asserts the compiled server's
// dependency graph contains neither this package nor kin-openapi, so the
// validator never ships in the binary.
package openapicheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

var (
	loadOnce sync.Once
	router   routers.Router
	loadErr  error
)

// specPath resolves backend/openapi.yaml from this source file's location, so
// it works regardless of which package's tests are running.
func specPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "openapi.yaml")
}

func load() {
	data, err := os.ReadFile(specPath())
	if err != nil {
		loadErr = err
		return
	}
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		loadErr = err
		return
	}
	if err := doc.Validate(loader.Context); err != nil {
		loadErr = err
		return
	}
	router, loadErr = gorillamux.NewRouter(doc)
}

// AssertResponse fails t unless a response with the given status, headers, and
// body conforms to the operation documented for method+target in
// openapi/openapi.yaml. target is a path (optionally with a query string), e.g.
// "/api/auth/me". An undocumented status code is a failure.
func AssertResponse(t testing.TB, method, target string, status int, header http.Header, body []byte) {
	t.Helper()
	loadOnce.Do(load)
	if loadErr != nil {
		t.Fatalf("openapicheck: loading openapi.yaml: %v", loadErr)
	}

	req := httptest.NewRequest(method, target, nil)
	route, pathParams, err := router.FindRoute(req)
	if err != nil {
		t.Fatalf("openapicheck: no operation in openapi.yaml for %s %s: %v", method, target, err)
	}

	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status:  status,
		Header:  header,
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	in.SetBodyBytes(body)

	if err := openapi3filter.ValidateResponse(context.Background(), in); err != nil {
		t.Fatalf("openapicheck: %s %s response does not conform to openapi.yaml (status %d): %v",
			method, target, status, err)
	}
}
