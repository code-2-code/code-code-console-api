package mcpservers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	mcpv1 "code-code.internal/go-contract/platform/mcp/v1"
)

func TestRegisterHandlersListMCPServers(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, testService{})

	request := httptest.NewRequest(http.MethodGet, "/api/mcps", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var payload struct {
		Items []struct {
			McpID string `json:"mcpId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].McpID != "filesystem" {
		t.Fatalf("payload = %#v", payload.Items)
	}
}

func TestRegisterHandlersCreateAndDeleteMCPServer(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, testService{})

	create := httptest.NewRequest(http.MethodPost, "/api/mcps", strings.NewReader(`{"mcpId":"fetch","name":"Fetch","streamableHttp":{"endpointUrl":"https://mcp.example.com"}}`))
	create.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createRecorder, create)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body=%s", createRecorder.Code, createRecorder.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/mcps/fetch", nil)
	deleteRecorder := httptest.NewRecorder()
	mux.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200, body=%s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

type testService struct{}

func (testService) List(context.Context) ([]*managementv1.MCPServerListItem, error) {
	return []*managementv1.MCPServerListItem{{McpId: "filesystem", Name: "Filesystem"}}, nil
}

func (testService) Get(context.Context, string) (*mcpv1.MCPServer, error) {
	return &mcpv1.MCPServer{McpId: "filesystem", Name: "Filesystem"}, nil
}

func (testService) Create(_ context.Context, request *managementv1.UpsertMCPServerRequest) (*mcpv1.MCPServer, error) {
	return &mcpv1.MCPServer{McpId: request.GetMcpId(), Name: request.GetName()}, nil
}

func (testService) Update(_ context.Context, mcpID string, request *managementv1.UpsertMCPServerRequest) (*mcpv1.MCPServer, error) {
	return &mcpv1.MCPServer{McpId: mcpID, Name: request.GetName()}, nil
}

func (testService) Delete(context.Context, string) error { return nil }
