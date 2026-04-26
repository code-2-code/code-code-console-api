package platformclient

import (
	"context"
	"fmt"
	"io"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (p *Providers) ListProviderSurfaceMetadata(ctx context.Context) ([]*providerv1.ProviderSurface, error) {
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

func (p *Providers) ListProviderSurfaceBindings(ctx context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.ListProviderSurfaceBindings(ctx, &providerservicev1.ListProviderSurfaceBindingsRequest{})
	if err != nil {
		return nil, err
	}
	out := &managementv1.ListProviderSurfaceBindingsResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetItems(), nil
}

func (p *Providers) CreateProviderSurfaceBinding(ctx context.Context, request *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	surface := &providerservicev1.UpsertProviderSurfaceBindingRequest{}
	if err := transcodeProviderMessage(request, surface); err != nil {
		return nil, err
	}
	response, err := client.CreateProviderSurfaceBinding(ctx, &providerservicev1.CreateProviderSurfaceBindingRequest{Surface: surface})
	if err != nil {
		return nil, err
	}
	out := &managementv1.CreateProviderSurfaceBindingResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetSurface(), nil
}

func (p *Providers) UpdateProviderSurfaceBinding(ctx context.Context, surfaceID string, request *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	surface := &providerservicev1.UpsertProviderSurfaceBindingRequest{}
	if err := transcodeProviderMessage(request, surface); err != nil {
		return nil, err
	}
	response, err := client.UpdateProviderSurfaceBinding(ctx, &providerservicev1.UpdateProviderSurfaceBindingRequest{SurfaceId: surfaceID, Surface: surface})
	if err != nil {
		return nil, err
	}
	out := &managementv1.UpdateProviderSurfaceBindingResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetSurface(), nil
}

func (p *Providers) DeleteProviderSurfaceBinding(ctx context.Context, surfaceID string) error {
	client, err := p.client.requireProvider()
	if err != nil {
		return err
	}
	_, err = client.DeleteProviderSurfaceBinding(ctx, &providerservicev1.DeleteProviderSurfaceBindingRequest{SurfaceId: surfaceID})
	return err
}

func (p *Providers) UpdateProvider(ctx context.Context, providerID string, request *managementv1.UpdateProviderRequest) (*managementv1.ProviderView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateProvider(ctx, &providerservicev1.UpdateProviderRequest{
		ProviderId:  providerID,
		DisplayName: request.GetDisplayName(),
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
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	request.ProviderId = providerID
	providerRequest := &providerservicev1.UpdateProviderAuthenticationRequest{}
	if err := transcodeProviderMessage(request, providerRequest); err != nil {
		return nil, err
	}
	response, err := client.UpdateProviderAuthentication(ctx, providerRequest)
	if err != nil {
		return nil, err
	}
	out := &managementv1.UpdateProviderAuthenticationResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *Providers) UpdateProviderObservabilityAuthentication(
	ctx context.Context,
	providerID string,
	request *managementv1.UpdateProviderObservabilityAuthenticationRequest,
) (*managementv1.ProviderView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	request.ProviderId = providerID
	providerRequest := &providerservicev1.UpdateProviderObservabilityAuthenticationRequest{}
	if err := transcodeProviderMessage(request, providerRequest); err != nil {
		return nil, err
	}
	response, err := client.UpdateProviderObservabilityAuthentication(ctx, providerRequest)
	if err != nil {
		return nil, err
	}
	out := &managementv1.UpdateProviderObservabilityAuthenticationResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetProvider(), nil
}

func (p *Providers) DeleteProvider(ctx context.Context, providerID string) error {
	client, err := p.client.requireProvider()
	if err != nil {
		return err
	}
	_, err = client.DeleteProvider(ctx, &providerservicev1.DeleteProviderRequest{ProviderId: providerID})
	return err
}

func (p *Providers) Connect(ctx context.Context, request *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	providerRequest := &providerservicev1.ConnectProviderRequest{}
	if err := transcodeProviderMessage(request, providerRequest); err != nil {
		return nil, err
	}
	response, err := client.ConnectProvider(ctx, providerRequest)
	if err != nil {
		return nil, err
	}
	out := &managementv1.ConnectProviderResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *Providers) GetConnectSession(ctx context.Context, sessionID string) (*managementv1.ProviderConnectSessionView, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.GetProviderConnectSession(ctx, &providerservicev1.GetProviderConnectSessionRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	out := &managementv1.GetProviderConnectSessionResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetSession(), nil
}

func (p *Providers) ProbeProvidersObservability(ctx context.Context, providerIDs []string) (*managementv1.ProbeProviderObservabilityResponse, error) {
	client, err := p.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.ProbeProviderObservability(ctx, &providerservicev1.ProbeProviderObservabilityRequest{ProviderIds: providerIDs})
	if err != nil {
		return nil, err
	}
	out := &managementv1.ProbeProviderObservabilityResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out, nil
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
