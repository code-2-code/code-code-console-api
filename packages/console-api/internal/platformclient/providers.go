package platformclient

import (
	"context"
	"fmt"
	"io"
	"strings"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ProviderAuthenticationSummary struct {
	Provider      *CredentialSubjectSummary `json:"provider,omitempty"`
	Observability *CredentialSubjectSummary `json:"observability,omitempty"`
}

type CredentialSubjectSummary struct {
	CredentialID string                          `json:"credentialId,omitempty"`
	Fields       []CredentialSubjectSummaryField `json:"fields,omitempty"`
}

type CredentialSubjectSummaryField struct {
	FieldID string `json:"fieldId,omitempty"`
	Label   string `json:"label,omitempty"`
	Value   string `json:"value,omitempty"`
}

func (p *Providers) ListProviderSurfaceMetadata(ctx context.Context) ([]*supportv1.Surface, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.ListProviderSurfaces(ctx, &providerservicev1.ListProviderSurfacesRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (p *Providers) ListProviders(ctx context.Context) ([]*managementv1.ProviderView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.ListProviders(ctx, &providerservicev1.ListProvidersRequest{})
	if err != nil {
		return nil, err
	}
	out := &managementv1.ListProvidersResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetItems(), nil
}

func (p *Providers) UpdateProvider(ctx context.Context, providerID string, request *managementv1.UpdateProviderRequest) (*managementv1.ProviderView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateProvider(ctx, &providerservicev1.UpdateProviderRequest{
		ProviderId: providerID,
		Provider: &providerservicev1.UpsertProviderRequest{
			DisplayName: request.GetProvider().GetDisplayName(),
		},
	})
	if err != nil {
		return nil, err
	}
	out := &managementv1.UpdateProviderResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetProvider(), nil
}

func (p *Providers) UpdateProviderAuthentication(ctx context.Context, providerID string, request *managementv1.UpdateProviderAuthenticationRequest) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	client, err := p.client.requireProviderOrchestration()
	if err != nil {
		return nil, err
	}
	request.ProviderId = providerID
	response, err := client.UpdateProviderAuthentication(ctx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (p *Providers) UpdateProviderObservabilityAuthentication(
	ctx context.Context,
	providerID string,
	request *managementv1.UpdateProviderObservabilityAuthenticationRequest,
) (*managementv1.ProviderView, error) {
	client, err := p.client.requireProviderOrchestration()
	if err != nil {
		return nil, err
	}
	request.ProviderId = providerID
	response, err := client.UpdateProviderObservabilityAuthentication(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.GetProvider(), nil
}

func (p *Providers) GetProviderAuthenticationSummary(ctx context.Context, providerID string) (*ProviderAuthenticationSummary, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, fmt.Errorf("provider_id is required")
	}
	auth, err := p.client.requireAuth()
	if err != nil {
		return nil, err
	}
	provider, err := p.providerByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	providerSummary, err := credentialSubjectSummary(ctx, auth, provider.GetProviderCredentialId())
	if err != nil {
		return nil, err
	}
	observabilitySummary, err := credentialSubjectSummary(ctx, auth, providerObservabilityCredentialID(provider.GetProviderId()))
	if err != nil {
		return nil, err
	}
	return &ProviderAuthenticationSummary{
		Provider:      providerSummary,
		Observability: observabilitySummary,
	}, nil
}

func (p *Providers) DeleteProvider(ctx context.Context, providerID string) error {
	client, err := p.client.requireProvider()
	if err != nil {
		return err
	}
	_, err = client.DeleteProvider(ctx, &providerservicev1.DeleteProviderRequest{ProviderId: providerID})
	return err
}

func (p *Providers) ProbeProviderModelCatalog(ctx context.Context, providerID string) (*managementv1.ProbeProviderModelCatalogResponse, error) {
	client, err := p.client.requireProviderOrchestration()
	if err != nil {
		return nil, err
	}
	return client.ProbeProviderModelCatalog(ctx, &managementv1.ProbeProviderModelCatalogRequest{ProviderId: providerID})
}

func (p *Providers) Connect(ctx context.Context, request *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	client, err := p.client.requireProviderOrchestration()
	if err != nil {
		return nil, err
	}
	response, err := client.ConnectProvider(ctx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (p *Providers) GetConnectSession(ctx context.Context, sessionID string) (*managementv1.ProviderConnectSessionView, error) {
	client, err := p.client.requireProviderOrchestration()
	if err != nil {
		return nil, err
	}
	response, err := client.GetProviderConnectSession(ctx, &managementv1.GetProviderConnectSessionRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (p *Providers) providerByID(ctx context.Context, providerID string) (*managementv1.ProviderView, error) {
	items, err := p.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.GetProviderId()) == providerID {
			return item, nil
		}
	}
	return nil, fmt.Errorf("provider %q not found", providerID)
}

func credentialSubjectSummary(ctx context.Context, auth authv1.AuthServiceClient, credentialID string) (*CredentialSubjectSummary, error) {
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return nil, nil
	}
	response, err := auth.GetCredentialSubjectSummary(ctx, &authv1.GetCredentialSubjectSummaryRequest{
		CredentialId: credentialID,
	})
	if err != nil {
		return nil, err
	}
	return &CredentialSubjectSummary{
		CredentialID: credentialID,
		Fields:       credentialSubjectSummaryFields(response.GetFields()),
	}, nil
}

func credentialSubjectSummaryFields(fields []*managementv1.CredentialSubjectSummaryFieldView) []CredentialSubjectSummaryField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]CredentialSubjectSummaryField, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.GetValue()) == "" {
			continue
		}
		out = append(out, CredentialSubjectSummaryField{
			FieldID: strings.TrimSpace(field.GetFieldId()),
			Label:   strings.TrimSpace(field.GetLabel()),
			Value:   strings.TrimSpace(field.GetValue()),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func providerObservabilityCredentialID(providerID string) string {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ""
	}
	return providerID + "-observability"
}

func (p *Providers) ProbeProvidersObservability(ctx context.Context, providerIDs []string) (*managementv1.ProbeProviderObservabilityResponse, error) {
	client, err := p.client.requireProviderOrchestration()
	if err != nil {
		return nil, err
	}
	response, err := client.ProbeProviderObservability(ctx, &managementv1.ProbeProviderObservabilityRequest{ProviderIds: providerIDs})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (p *Providers) WatchStatusEvents(
	ctx context.Context,
	providerIDs []string,
	yield func(*managementv1.ProviderStatusEvent) error,
) error {
	if yield == nil {
		return fmt.Errorf("console-api/platformclient: provider status event yield is nil")
	}
	client, err := p.client.requireProvider()
	if err != nil {
		return err
	}
	stream, err := client.WatchProviderStatusEvents(ctx, &providerservicev1.WatchProviderStatusEventsRequest{
		ProviderIds: providerIDs,
	})
	if err != nil {
		return err
	}
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		event := &managementv1.ProviderStatusEvent{}
		if err := transcodeProviderMessage(response.GetEvent(), event); err != nil {
			return err
		}
		if err := yield(event); err != nil {
			return err
		}
	}
}

func transcodeProviderMessage(src proto.Message, dst proto.Message) error {
	if src == nil || dst == nil {
		return nil
	}
	body, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(src)
	if err != nil {
		return fmt.Errorf("console-api/platformclient: marshal provider message: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, dst); err != nil {
		return fmt.Errorf("console-api/platformclient: unmarshal provider message: %w", err)
	}
	return nil
}
