package connectproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code-code.internal/go-contract/platform/model/v1/modelservicev1connect"
	"code-code.internal/go-contract/platform/provider/v1/providerservicev1connect"
)

func TestModelServiceProxyForwardsListModelDefinitions(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, modelservicev1connect.ModelServiceListModelDefinitionsProcedure; got != want {
			t.Fatalf("upstream path = %q, want %q", got, want)
		}
		if got, want := r.URL.RawQuery, "pageSize=20"; got != want {
			t.Fatalf("upstream query = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	handler, err := NewHandler(Config{ModelBaseURL: upstream.URL, ProviderBaseURL: "http://platform-provider-service:8080"})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, ConsolePathPrefix+modelservicev1connect.ModelServiceListModelDefinitionsProcedure+"?pageSize=20", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestProxyForwardsListVendors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, providerservicev1connect.ProviderServiceListVendorsProcedure; got != want {
			t.Fatalf("upstream path = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	handler, err := NewHandler(Config{ModelBaseURL: "http://platform-model-service:8080", ProviderBaseURL: upstream.URL})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, ConsolePathPrefix+providerservicev1connect.ProviderServiceListVendorsProcedure, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestProxyRejectsOtherProcedures(t *testing.T) {
	handler, err := NewHandler(Config{
		ModelBaseURL:    "http://platform-model-service:8080",
		ProviderBaseURL: "http://platform-provider-service:8080",
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, ConsolePathPrefix+modelservicev1connect.ModelServiceSyncModelDefinitionsProcedure, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}
