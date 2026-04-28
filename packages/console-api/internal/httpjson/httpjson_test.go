package httpjson

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithCORSAllowsDelete(t *testing.T) {
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodOptions, "/api/connect/platform.model.v1.ModelService/ListModels", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if got, want := recorder.Header().Get("Access-Control-Allow-Methods"), "GET,POST,PUT,DELETE,OPTIONS"; got != want {
		t.Fatalf("allow methods = %q, want %q", got, want)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Connect-Protocol-Version") {
		t.Fatalf("allow headers = %q, want Connect-Protocol-Version", got)
	}
}
