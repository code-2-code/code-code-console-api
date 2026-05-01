package referencedata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

func TestRegisterCLIDefinitionHandlersListCLIDefinitions(t *testing.T) {
	t.Parallel()

	stub := &cliSupportManagementStub{}
	mux := http.NewServeMux()
	RegisterCLIDefinitionHandlers(mux, stub)

	request := httptest.NewRequest(http.MethodGet, "/api/cli-definitions", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !stub.listCalled {
		t.Fatal("ListCLIs() was not called")
	}

	var payload struct {
		Items []struct {
			CLIID      string `json:"cliId"`
			WebsiteURL string `json:"websiteUrl"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].CLIID != "codex" {
		t.Fatalf("items = %#v, want codex", payload.Items)
	}
	if payload.Items[0].WebsiteURL != "https://openai.com/codex/" {
		t.Fatalf("websiteUrl = %q, want OpenAI Codex website", payload.Items[0].WebsiteURL)
	}
}

type cliSupportManagementStub struct {
	listCalled bool
}

func (s *cliSupportManagementStub) ListCLIs(context.Context) ([]*supportv1.CLI, error) {
	s.listCalled = true
	return []*supportv1.CLI{{
		CliId:       "codex",
		DisplayName: "Codex CLI",
		WebsiteUrl:  "https://openai.com/codex/",
	}}, nil
}
