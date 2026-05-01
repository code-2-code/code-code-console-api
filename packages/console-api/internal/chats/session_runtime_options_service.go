package chats

import (
	"context"
	"fmt"

	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	cliruntimev1 "code-code.internal/go-contract/platform/cli_runtime/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

type providerLister interface {
	ListProviders(context.Context) ([]*managementv1.ProviderView, error)
}

type cliSupportLister interface {
	ListCLIs(context.Context) ([]*supportv1.CLI, error)
}

type cliRuntimeImageLister interface {
	LatestAvailableImages(context.Context) ([]*cliruntimev1.CLIRuntimeImage, error)
}

type sessionRuntimeOptionsService interface {
	View(context.Context) (*sessionRuntimeOptionsView, error)
	ValidateInlineSpec(context.Context, *agentsessionv1.AgentSessionSpec) error
}

type sessionRuntimeOptionsCatalogService struct {
	providers        providerLister
	cliSupport       cliSupportLister
	cliRuntimeImages cliRuntimeImageLister
}

func NewSessionRuntimeOptionsService(
	providers providerLister,
	cliSupport cliSupportLister,
	cliRuntimeImages cliRuntimeImageLister,
) sessionRuntimeOptionsService {
	if providers == nil || cliSupport == nil || cliRuntimeImages == nil {
		return nil
	}
	return &sessionRuntimeOptionsCatalogService{
		providers:        providers,
		cliSupport:       cliSupport,
		cliRuntimeImages: cliRuntimeImages,
	}
}

func (s *sessionRuntimeOptionsCatalogService) View(ctx context.Context) (*sessionRuntimeOptionsView, error) {
	catalog, err := s.loadCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return catalog.view, nil
}

func (s *sessionRuntimeOptionsCatalogService) ValidateInlineSpec(ctx context.Context, spec *agentsessionv1.AgentSessionSpec) error {
	catalog, err := s.loadCatalog(ctx)
	if err != nil {
		return err
	}
	return validateInlineSpecAgainstCatalog(spec, catalog)
}

func (s *sessionRuntimeOptionsCatalogService) loadCatalog(ctx context.Context) (*runtimeCatalog, error) {
	clis, err := s.cliSupport.ListCLIs(ctx)
	if err != nil {
		return nil, fmt.Errorf("load CLIs from support: %w", err)
	}
	providerSurfaces, err := s.providers.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("load providers: %w", err)
	}
	availableImages, err := s.cliRuntimeImages.LatestAvailableImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("load CLI runtime images: %w", err)
	}
	return buildRuntimeCatalog(clis, availableImages, providerSurfaces), nil
}
