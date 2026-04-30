package server

import (
	"fmt"
	"net/http"
	"time"

	"code-code.internal/console-api/internal/chats"
	"code-code.internal/console-api/internal/connectproxy"
	"code-code.internal/console-api/internal/crudhandler"
	"code-code.internal/console-api/internal/egresspolicies"
	"code-code.internal/console-api/internal/httpjson"
	"code-code.internal/console-api/internal/oauthsessions"
	"code-code.internal/console-api/internal/platformclient"
	"code-code.internal/console-api/internal/providers"
	"code-code.internal/console-api/internal/referencedata"
	"code-code.internal/console-api/internal/templates"
	agentprofilev1 "code-code.internal/go-contract/platform/agent_profile/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	mcpv1 "code-code.internal/go-contract/platform/mcp/v1"
	rulev1 "code-code.internal/go-contract/platform/rule/v1"
	skillv1 "code-code.internal/go-contract/platform/skill/v1"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/protobuf/proto"
)

// Config groups dependencies required to assemble the console API.
type Config struct {
	Platform               *platformclient.Client
	PrometheusBaseURL      string
	ModelConnectBaseURL    string
	ProviderConnectBaseURL string
}

// Server bundles the HTTP handler.
type Server struct {
	Handler http.Handler
}

// New creates one console API server.
func New(config Config) (*Server, error) {
	if config.Platform == nil {
		return nil, fmt.Errorf("consoleapi/server: platform client is nil")
	}
	connectHandler, err := connectproxy.NewHandler(connectproxy.Config{
		ModelBaseURL:    config.ModelConnectBaseURL,
		ProviderBaseURL: config.ProviderConnectBaseURL,
	})
	if err != nil {
		return nil, err
	}
	promQueryClient, err := providers.NewPrometheusQueryClient(
		config.PrometheusBaseURL,
		&http.Client{
			Timeout:   8 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	)
	if err != nil {
		return nil, err
	}
	observabilityService, err := providers.NewObservabilityService(providers.ObservabilityServiceConfig{
		Providers:  config.Platform.Providers(),
		Support:    config.Platform.SupportResources(),
		Prometheus: promQueryClient,
		Prober:     config.Platform.Providers(),
	})
	if err != nil {
		return nil, err
	}
	providerService, err := providers.NewHostTelemetryProviderService(config.Platform.Providers(), promQueryClient)
	if err != nil {
		return nil, err
	}
	sessionClient, err := config.Platform.AgentSessionManagementClient()
	if err != nil {
		return nil, err
	}
	chatClient, err := config.Platform.ChatServiceClient()
	if err != nil {
		return nil, err
	}
	chatFacade := chats.NewGRPCChatClient(chatClient)

	mux := http.NewServeMux()
	mux.Handle(connectproxy.ConsolePathPrefix+"/", connectHandler)
	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	chats.RegisterHandlers(
		mux,
		chatFacade,
		config.Platform.AgentSessions(),
		config.Platform.AgentSessionActions(),
		config.Platform.AgentRuns(),
		chats.NewGRPCRunOutputClient(sessionClient),
		chatFacade,
	)
	registerCRUDHandlers(mux, config.Platform)
	providers.RegisterHandlers(mux, providerService)
	providers.RegisterObservabilityHandlers(mux, observabilityService)
	egresspolicies.RegisterHandlers(mux, config.Platform.EgressPolicies())
	oauthsessions.RegisterHandlers(mux, config.Platform.OAuthSessions())
	templates.RegisterHandlers(mux, config.Platform.Templates())
	referencedata.RegisterCLIDefinitionHandlers(mux, config.Platform.CLIDefinitions())
	referencedata.RegisterSupportResourceHandlers(mux, config.Platform.SupportResources())
	return &Server{Handler: httpjson.WithCORS(mux)}, nil
}

func registerCRUDHandlers(mux *http.ServeMux, platform *platformclient.Client) {
	crudhandler.Register(mux, platform.AgentProfiles(), crudhandler.Config[*managementv1.AgentProfileListItem, *agentprofilev1.AgentProfile, *managementv1.UpsertAgentProfileRequest]{
		ResourceName: "agent-profiles",
		WrapList:     func(items []*managementv1.AgentProfileListItem) proto.Message { return &managementv1.ListAgentProfilesResponse{Items: items} },
		NewRequest:   func() *managementv1.UpsertAgentProfileRequest { return &managementv1.UpsertAgentProfileRequest{} },
	})
	crudhandler.Register(mux, platform.MCPServers(), crudhandler.Config[*managementv1.MCPServerListItem, *mcpv1.MCPServer, *managementv1.UpsertMCPServerRequest]{
		ResourceName: "mcps",
		WrapList:     func(items []*managementv1.MCPServerListItem) proto.Message { return &managementv1.ListMCPServersResponse{Items: items} },
		NewRequest:   func() *managementv1.UpsertMCPServerRequest { return &managementv1.UpsertMCPServerRequest{} },
	})
	crudhandler.Register(mux, platform.Skills(), crudhandler.Config[*managementv1.SkillListItem, *skillv1.Skill, *managementv1.UpsertSkillRequest]{
		ResourceName: "skills",
		WrapList:     func(items []*managementv1.SkillListItem) proto.Message { return &managementv1.ListSkillsResponse{Items: items} },
		NewRequest:   func() *managementv1.UpsertSkillRequest { return &managementv1.UpsertSkillRequest{} },
	})
	crudhandler.Register(mux, platform.Rules(), crudhandler.Config[*managementv1.RuleListItem, *rulev1.Rule, *managementv1.UpsertRuleRequest]{
		ResourceName: "rules",
		WrapList:     func(items []*managementv1.RuleListItem) proto.Message { return &managementv1.ListRulesResponse{Items: items} },
		NewRequest:   func() *managementv1.UpsertRuleRequest { return &managementv1.UpsertRuleRequest{} },
	})
}
