package chats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	cliruntimev1 "code-code.internal/go-contract/platform/cli_runtime/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRegisterHandlersSessionRuntimeOptions(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(
		mux,
		newFakeSessions(),
		newFakeSessions(),
		&fakeActions{},
		&fakeRuns{},
		nil,
		newTestSessionRuntimeOptionsService(),
	)

	request := httptest.NewRequest(http.MethodGet, "/api/chats/session-runtime-options", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("session runtime options status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload chatv1.GetSessionRuntimeOptionsResponse
	if err := protojson.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("protojson.Unmarshal() error = %v", err)
	}
	if len(payload.GetItems()) != 1 {
		t.Fatalf("runtime option count = %d, want 1", len(payload.GetItems()))
	}
	item := payload.GetItems()[0]
	if item.GetProviderId() != "codex" {
		t.Fatalf("providerId = %q, want codex", item.GetProviderId())
	}
	if len(item.GetExecutionClasses()) != 2 || item.GetExecutionClasses()[0] != "cli-standard" {
		t.Fatalf("execution classes = %#v", item.GetExecutionClasses())
	}
	if len(item.GetSurfaces()) != 1 || item.GetSurfaces()[0].GetRuntimeRef().GetSurfaceId() != "openai-default" {
		t.Fatalf("surfaces = %#v", item.GetSurfaces())
	}
	if len(item.GetSurfaces()[0].GetModels()) != 2 || item.GetSurfaces()[0].GetModels()[0] != "gpt-5" {
		t.Fatalf("models = %#v", item.GetSurfaces()[0].GetModels())
	}
}

func TestRegisterHandlersRejectsInvalidInlineRuntimeSelection(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	RegisterHandlers(
		mux,
		sessions,
		sessions,
		&fakeActions{},
		&fakeRuns{},
		nil,
		newTestSessionRuntimeOptionsService(),
	)

	request := httptest.NewRequest(http.MethodPut, "/api/chats/chat-invalid", strings.NewReader(`{
		"sessionSetup":{
			"mode":"inline",
			"inline":{
				"providerId":"codex",
				"executionClass":"cli-standard",
				"runtimeConfig":{
					"providerRuntimeRef":{"surfaceId":"anthropic-default"},
					"primaryModelSelector":{"providerModelId":"claude-3-7-sonnet"}
				},
				"resourceConfig":{}
			}
		}
	}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid inline status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "not selectable") == false {
		t.Fatalf("expected selectable validation error, body=%s", recorder.Body.String())
	}
	if _, ok := sessions.stateByID["chat-invalid"]; ok {
		t.Fatalf("invalid inline request should not create session")
	}
}

func newTestSessionRuntimeOptionsService() sessionRuntimeOptionsService {
	return NewSessionRuntimeOptionsService(
		sessionRuntimeProviderSurfaceBindingStub{
			items: []*managementv1.ProviderSurfaceBindingView{
				runtimeOptionInstance(
					"openai-default",
					"OpenAI Default",
					"",
					apiprotocolv1.Protocol_PROTOCOL_OPENAI_RESPONSES,
					[]string{"gpt-5", "gpt-5-mini"},
				),
				runtimeOptionInstance(
					"anthropic-default",
					"Anthropic Default",
					"",
					apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC,
					[]string{"claude-3-7-sonnet"},
				),
			},
		},
		sessionRuntimeCLIDefinitionStub{
			items: []*managementv1.CLIDefinitionView{{
				CliId:       "codex",
				DisplayName: "Codex",
				ContainerImages: []*managementv1.CLIContainerImageView{
					{ExecutionClass: "cli-standard"},
					{ExecutionClass: "cli-long-context"},
					{ExecutionClass: "cli-sandboxed"},
				},
			}},
		},
		sessionRuntimeCLIStub{
			items: []*supportv1.CLI{{
				CliId:       "codex",
				DisplayName: "Codex",
				ApiKeyProtocols: []*supportv1.APIKeyProtocolSupport{{
					Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_RESPONSES,
				}},
			}},
		},
		sessionRuntimeImageStub{
			items: []*cliruntimev1.CLIRuntimeImage{
				{CliId: "codex", ExecutionClass: "cli-standard", Image: "code-code/codex:cli-latest"},
				{CliId: "codex", ExecutionClass: "cli-long-context", Image: "code-code/codex-long:cli-latest"},
			},
		},
	)
}

func runtimeOptionInstance(
	instanceID string,
	label string,
	cliID string,
	protocol apiprotocolv1.Protocol,
	models []string,
) *managementv1.ProviderSurfaceBindingView {
	entries := make([]*providerv1.ProviderModelCatalogEntry, 0, len(models))
	for _, modelID := range models {
		entries = append(entries, &providerv1.ProviderModelCatalogEntry{ProviderModelId: modelID})
	}
	runtime := &providerv1.ProviderSurfaceRuntime{
		DisplayName: label,
		Catalog:     &providerv1.ProviderModelCatalog{Models: entries},
	}
	if strings.TrimSpace(cliID) != "" {
		runtime.Access = &providerv1.ProviderSurfaceRuntime_Cli{
			Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: strings.TrimSpace(cliID)},
		}
	} else {
		runtime.Access = &providerv1.ProviderSurfaceRuntime_Api{
			Api: &providerv1.ProviderAPISurfaceRuntime{Protocol: protocol},
		}
	}
	return &managementv1.ProviderSurfaceBindingView{
		SurfaceId:   instanceID,
		DisplayName: label,
		VendorId:    "vendor-" + instanceID,
		ProviderId:  "provider-" + instanceID,
		Runtime:     runtime,
	}
}

type sessionRuntimeProviderSurfaceBindingStub struct {
	items []*managementv1.ProviderSurfaceBindingView
}

func (s sessionRuntimeProviderSurfaceBindingStub) ListProviderSurfaceBindings(context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	return s.items, nil
}

type sessionRuntimeCLIDefinitionStub struct {
	items []*managementv1.CLIDefinitionView
}

func (s sessionRuntimeCLIDefinitionStub) List(context.Context) ([]*managementv1.CLIDefinitionView, error) {
	return s.items, nil
}

type sessionRuntimeCLIStub struct {
	items []*supportv1.CLI
}

func (s sessionRuntimeCLIStub) ListCLIs(context.Context) ([]*supportv1.CLI, error) {
	return s.items, nil
}

type sessionRuntimeImageStub struct {
	items []*cliruntimev1.CLIRuntimeImage
}

func (s sessionRuntimeImageStub) LatestAvailableImages(context.Context) ([]*cliruntimev1.CLIRuntimeImage, error) {
	return s.items, nil
}
