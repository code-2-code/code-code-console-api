package server

import (
	"fmt"
	"net/http"
	"time"

	"code-code.internal/console-api/internal/agentprofiles"
	"code-code.internal/console-api/internal/chats"
	"code-code.internal/console-api/internal/connectproxy"
	"code-code.internal/console-api/internal/egresspolicies"
	"code-code.internal/console-api/internal/httpjson"
	"code-code.internal/console-api/internal/mcpservers"
	"code-code.internal/console-api/internal/oauthsessions"
	"code-code.internal/console-api/internal/platformclient"
	"code-code.internal/console-api/internal/providers"
	"code-code.internal/console-api/internal/referencedata"
	"code-code.internal/console-api/internal/rules"
	"code-code.internal/console-api/internal/skills"
	"code-code.internal/console-api/internal/templates"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	agentprofiles.RegisterHandlers(mux, config.Platform.AgentProfiles())
	mcpservers.RegisterHandlers(mux, config.Platform.MCPServers())
	skills.RegisterHandlers(mux, config.Platform.Skills())
	rules.RegisterHandlers(mux, config.Platform.Rules())
	providers.RegisterHandlers(mux, providerService)
	providers.RegisterObservabilityHandlers(mux, observabilityService)
	egresspolicies.RegisterHandlers(mux, config.Platform.EgressPolicies())
	oauthsessions.RegisterHandlers(mux, config.Platform.OAuthSessions())
	templates.RegisterHandlers(mux, config.Platform.Templates())
	referencedata.RegisterCLIDefinitionHandlers(mux, config.Platform.CLIDefinitions())
	referencedata.RegisterSupportResourceHandlers(mux, config.Platform.SupportResources())
	return &Server{Handler: httpjson.WithCORS(mux)}, nil
}
