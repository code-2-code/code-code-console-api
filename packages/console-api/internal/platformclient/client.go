package platformclient

import (
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	cliruntimev1 "code-code.internal/go-contract/platform/cli_runtime/v1"
	egressservicev1 "code-code.internal/go-contract/platform/egress/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"
	profileservicev1 "code-code.internal/go-contract/platform/profile/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Config struct {
	SessionConn    grpc.ClientConnInterface
	ChatConn       grpc.ClientConnInterface
	ProviderConn   grpc.ClientConnInterface
	CLIRuntimeConn grpc.ClientConnInterface
	ProfileConn    grpc.ClientConnInterface
	EgressConn     grpc.ClientConnInterface
	AuthConn       grpc.ClientConnInterface
	SupportConn    grpc.ClientConnInterface
}

// Client adapts platform gRPC upstreams to console domain services.
type Client struct {
	sessionManagement   managementv1.AgentSessionManagementServiceClient
	chat                chatv1.ChatServiceClient
	provider            providerservicev1.ProviderServiceClient
	cliRuntime          cliruntimev1.CLIRuntimeServiceClient
	profile             profileservicev1.ProfileServiceClient
	egress              egressservicev1.EgressServiceClient
	support             supportv1.SupportServiceClient
	oauthSession        oauthv1.OAuthSessionServiceClient
	oauthCallback       oauthv1.OAuthCallbackServiceClient
	agentProfiles       AgentProfiles
	mcpServers          MCPServers
	skills              Skills
	rules               Rules
	providers           Providers
	egressPolicies      EgressPolicies
	templates           Templates
	cliDefinitions      CLIDefinitions
	cliRuntimes         CLIRuntimes
	supportResources    SupportResources
	agentSessions       AgentSessions
	agentSessionActions AgentSessionActions
	agentRuns           AgentRuns
	oauthSessions       OAuthSessions
}

func New(config Config) (*Client, error) {
	c := &Client{}
	if config.SessionConn != nil {
		c.sessionManagement = managementv1.NewAgentSessionManagementServiceClient(config.SessionConn)
	}
	if config.ChatConn != nil {
		c.chat = chatv1.NewChatServiceClient(config.ChatConn)
	}
	if config.ProviderConn != nil {
		c.provider = providerservicev1.NewProviderServiceClient(config.ProviderConn)
	}
	if config.CLIRuntimeConn != nil {
		c.cliRuntime = cliruntimev1.NewCLIRuntimeServiceClient(config.CLIRuntimeConn)
	}
	if config.ProfileConn != nil {
		c.profile = profileservicev1.NewProfileServiceClient(config.ProfileConn)
	}
	if config.EgressConn != nil {
		c.egress = egressservicev1.NewEgressServiceClient(config.EgressConn)
	}
	if config.SupportConn != nil {
		c.support = supportv1.NewSupportServiceClient(config.SupportConn)
	}
	if config.AuthConn != nil {
		c.oauthSession = oauthv1.NewOAuthSessionServiceClient(config.AuthConn)
		c.oauthCallback = oauthv1.NewOAuthCallbackServiceClient(config.AuthConn)
	}
	c.agentProfiles = AgentProfiles{client: c}
	c.mcpServers = MCPServers{client: c}
	c.skills = Skills{client: c}
	c.rules = Rules{client: c}
	c.providers = Providers{client: c}
	c.egressPolicies = EgressPolicies{client: c}
	c.templates = Templates{client: c}
	c.cliDefinitions = CLIDefinitions{client: c}
	c.cliRuntimes = CLIRuntimes{client: c}
	c.supportResources = SupportResources{client: c}
	c.agentSessions = AgentSessions{client: c}
	c.agentSessionActions = AgentSessionActions{client: c}
	c.agentRuns = AgentRuns{client: c}
	c.oauthSessions = OAuthSessions{client: c}
	return c, nil
}

func (c *Client) requireSessionManagement() (managementv1.AgentSessionManagementServiceClient, error) {
	if c == nil || c.sessionManagement == nil {
		return nil, status.Error(codes.Unavailable, "agent session upstream is not configured")
	}
	return c.sessionManagement, nil
}

func (c *Client) requireChat() (chatv1.ChatServiceClient, error) {
	if c == nil || c.chat == nil {
		return nil, status.Error(codes.Unavailable, "chat upstream is not configured")
	}
	return c.chat, nil
}

func (c *Client) requireProvider() (providerservicev1.ProviderServiceClient, error) {
	if c == nil || c.provider == nil {
		return nil, status.Error(codes.Unavailable, "provider upstream is not configured")
	}
	return c.provider, nil
}

func (c *Client) requireCLIRuntime() (cliruntimev1.CLIRuntimeServiceClient, error) {
	if c == nil || c.cliRuntime == nil {
		return nil, status.Error(codes.Unavailable, "cli runtime upstream is not configured")
	}
	return c.cliRuntime, nil
}

func (c *Client) requireProfile() (profileservicev1.ProfileServiceClient, error) {
	if c == nil || c.profile == nil {
		return nil, status.Error(codes.Unavailable, "profile upstream is not configured")
	}
	return c.profile, nil
}

func (c *Client) requireEgress() (egressservicev1.EgressServiceClient, error) {
	if c == nil || c.egress == nil {
		return nil, status.Error(codes.Unavailable, "egress upstream is not configured")
	}
	return c.egress, nil
}

func (c *Client) requireSupport() (supportv1.SupportServiceClient, error) {
	if c == nil || c.support == nil {
		return nil, status.Error(codes.Unavailable, "support upstream is not configured")
	}
	return c.support, nil
}

func (c *Client) requireOAuthSession() (oauthv1.OAuthSessionServiceClient, error) {
	if c == nil || c.oauthSession == nil {
		return nil, status.Error(codes.Unavailable, "oauth session upstream is not configured")
	}
	return c.oauthSession, nil
}

func (c *Client) requireOAuthCallback() (oauthv1.OAuthCallbackServiceClient, error) {
	if c == nil || c.oauthCallback == nil {
		return nil, status.Error(codes.Unavailable, "oauth callback upstream is not configured")
	}
	return c.oauthCallback, nil
}

func (c *Client) AgentProfiles() *AgentProfiles             { return &c.agentProfiles }
func (c *Client) MCPServers() *MCPServers                   { return &c.mcpServers }
func (c *Client) Skills() *Skills                           { return &c.skills }
func (c *Client) Rules() *Rules                             { return &c.rules }
func (c *Client) Providers() *Providers                     { return &c.providers }
func (c *Client) EgressPolicies() *EgressPolicies           { return &c.egressPolicies }
func (c *Client) Templates() *Templates                     { return &c.templates }
func (c *Client) CLIDefinitions() *CLIDefinitions           { return &c.cliDefinitions }
func (c *Client) CLIRuntimes() *CLIRuntimes                 { return &c.cliRuntimes }
func (c *Client) SupportResources() *SupportResources       { return &c.supportResources }
func (c *Client) AgentSessions() *AgentSessions             { return &c.agentSessions }
func (c *Client) AgentSessionActions() *AgentSessionActions { return &c.agentSessionActions }
func (c *Client) AgentRuns() *AgentRuns                     { return &c.agentRuns }
func (c *Client) OAuthSessions() *OAuthSessions             { return &c.oauthSessions }

func (c *Client) AgentSessionManagementClient() (managementv1.AgentSessionManagementServiceClient, error) {
	return c.requireSessionManagement()
}

func (c *Client) ChatServiceClient() (chatv1.ChatServiceClient, error) {
	return c.requireChat()
}

type AgentProfiles struct{ client *Client }
type MCPServers struct{ client *Client }
type Skills struct{ client *Client }
type Rules struct{ client *Client }
type Providers struct{ client *Client }
type EgressPolicies struct{ client *Client }
type Templates struct{ client *Client }
type CLIDefinitions struct{ client *Client }
type CLIRuntimes struct{ client *Client }
type SupportResources struct{ client *Client }
type AgentSessions struct{ client *Client }
type AgentSessionActions struct{ client *Client }
type AgentRuns struct{ client *Client }
type OAuthSessions struct{ client *Client }
