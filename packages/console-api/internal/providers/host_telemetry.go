package providers

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/url"
	"slices"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const providerHostTelemetryQuery = `last_over_time({job="provider-host-latency",__name__=~"probe_success|probe_duration_seconds|probe_http_status_code"}[5m])`

type HostTelemetryProviderService struct {
	delegate providerService
	prom     promQueryExecutor
}

type providerHostTelemetryTarget struct {
	key       string
	targetURL string
	scheme    string
	host      string
	port      string
}

type providerHostTelemetryPoint struct {
	target     providerHostTelemetryTarget
	success    *float64
	latency    *float64
	statusCode *float64
	sampledAt  *timestamppb.Timestamp
}

func NewHostTelemetryProviderService(delegate providerService, prom promQueryExecutor) (*HostTelemetryProviderService, error) {
	if delegate == nil {
		return nil, fmt.Errorf("consoleapi/providers: host telemetry provider delegate is nil")
	}
	if prom == nil {
		return nil, fmt.Errorf("consoleapi/providers: host telemetry prometheus query client is nil")
	}
	return &HostTelemetryProviderService{delegate: delegate, prom: prom}, nil
}

func (s *HostTelemetryProviderService) ListProviderSurfaceMetadata(ctx context.Context) ([]*providerv1.ProviderSurface, error) {
	return s.delegate.ListProviderSurfaceMetadata(ctx)
}

func (s *HostTelemetryProviderService) ListProviders(ctx context.Context) ([]*managementv1.ProviderView, error) {
	items, err := s.delegate.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	return s.attachHostTelemetry(ctx, items), nil
}

func (s *HostTelemetryProviderService) ListProviderSurfaceBindings(ctx context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	return s.delegate.ListProviderSurfaceBindings(ctx)
}

func (s *HostTelemetryProviderService) UpdateProvider(ctx context.Context, providerID string, request *managementv1.UpdateProviderRequest) (*managementv1.ProviderView, error) {
	return s.delegate.UpdateProvider(ctx, providerID, request)
}

func (s *HostTelemetryProviderService) UpdateProviderAuthentication(ctx context.Context, providerID string, request *managementv1.UpdateProviderAuthenticationRequest) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	return s.delegate.UpdateProviderAuthentication(ctx, providerID, request)
}

func (s *HostTelemetryProviderService) UpdateProviderObservabilityAuthentication(ctx context.Context, providerID string, request *managementv1.UpdateProviderObservabilityAuthenticationRequest) (*managementv1.ProviderView, error) {
	return s.delegate.UpdateProviderObservabilityAuthentication(ctx, providerID, request)
}

func (s *HostTelemetryProviderService) DeleteProvider(ctx context.Context, providerID string) error {
	return s.delegate.DeleteProvider(ctx, providerID)
}

func (s *HostTelemetryProviderService) CreateProviderSurfaceBinding(ctx context.Context, request *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	return s.delegate.CreateProviderSurfaceBinding(ctx, request)
}

func (s *HostTelemetryProviderService) UpdateProviderSurfaceBinding(ctx context.Context, surfaceID string, request *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	return s.delegate.UpdateProviderSurfaceBinding(ctx, surfaceID, request)
}

func (s *HostTelemetryProviderService) DeleteProviderSurfaceBinding(ctx context.Context, surfaceID string) error {
	return s.delegate.DeleteProviderSurfaceBinding(ctx, surfaceID)
}

func (s *HostTelemetryProviderService) Connect(ctx context.Context, request *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	return s.delegate.Connect(ctx, request)
}

func (s *HostTelemetryProviderService) GetConnectSession(ctx context.Context, sessionID string) (*managementv1.ProviderConnectSessionView, error) {
	return s.delegate.GetConnectSession(ctx, sessionID)
}

func (s *HostTelemetryProviderService) WatchStatusEvents(ctx context.Context, providerIDs []string, yield func(*managementv1.ProviderStatusEvent) error) error {
	return s.delegate.WatchStatusEvents(ctx, providerIDs, yield)
}

func (s *HostTelemetryProviderService) attachHostTelemetry(ctx context.Context, providers []*managementv1.ProviderView) []*managementv1.ProviderView {
	targets := providerHostTelemetryTargetsFromProviders(providers)
	if len(targets) == 0 {
		return providers
	}
	points := map[string]*providerHostTelemetryPoint{}
	if s != nil && s.prom != nil {
		if samples, err := s.prom.QueryVector(ctx, providerHostTelemetryQuery); err == nil {
			points = providerHostTelemetryPointsFromSamples(samples)
		}
	}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerTelemetryByKey := map[string]*managementv1.ProviderHostTelemetry{}
		for _, surface := range provider.GetSurfaces() {
			target, ok := providerHostTelemetryTargetFromSurface(surface)
			if !ok {
				continue
			}
			telemetry := providerHostTelemetryView(target, points[target.key])
			surface.HostTelemetry = cloneProviderHostTelemetry(telemetry)
			providerTelemetryByKey[target.key] = telemetry
		}
		provider.HostTelemetry = sortedProviderHostTelemetry(providerTelemetryByKey)
	}
	return providers
}

func providerHostTelemetryTargetsFromProviders(providers []*managementv1.ProviderView) map[string]providerHostTelemetryTarget {
	targets := map[string]providerHostTelemetryTarget{}
	for _, provider := range providers {
		for _, surface := range provider.GetSurfaces() {
			target, ok := providerHostTelemetryTargetFromSurface(surface)
			if ok {
				targets[target.key] = target
			}
		}
	}
	return targets
}

func providerHostTelemetryPointsFromSamples(samples []promVectorSample) map[string]*providerHostTelemetryPoint {
	points := map[string]*providerHostTelemetryPoint{}
	for _, sample := range samples {
		target, ok := providerHostTelemetryTargetFromMetric(sample.Metric)
		if !ok {
			continue
		}
		point := points[target.key]
		if point == nil {
			point = &providerHostTelemetryPoint{target: target}
			points[target.key] = point
		}
		if point.sampledAt == nil || sample.Timestamp.After(point.sampledAt.AsTime()) {
			point.sampledAt = timestamppb.New(sample.Timestamp)
		}
		value := sample.Value
		switch strings.TrimSpace(sample.Metric["__name__"]) {
		case "probe_success":
			point.success = &value
		case "probe_duration_seconds":
			point.latency = &value
		case "probe_http_status_code":
			point.statusCode = &value
		}
	}
	return points
}

func providerHostTelemetryTargetFromMetric(metric map[string]string) (providerHostTelemetryTarget, bool) {
	if len(metric) == 0 {
		return providerHostTelemetryTarget{}, false
	}
	scheme := strings.ToLower(strings.TrimSpace(metric["scheme"]))
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(metric["host"])), ".")
	port := strings.TrimSpace(metric["port"])
	if scheme == "" || host == "" || port == "" {
		return normalizeProviderHostTelemetryTarget(metric["instance"])
	}
	return providerHostTelemetryTargetFromParts(scheme, host, port)
}

func providerHostTelemetryTargetFromSurface(surface *managementv1.ProviderSurfaceBindingView) (providerHostTelemetryTarget, bool) {
	if surface == nil || surface.GetRuntime().GetApi() == nil {
		return providerHostTelemetryTarget{}, false
	}
	return normalizeProviderHostTelemetryTarget(surface.GetRuntime().GetApi().GetBaseUrl())
}

func normalizeProviderHostTelemetryTarget(raw string) (providerHostTelemetryTarget, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return providerHostTelemetryTarget{}, false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return providerHostTelemetryTarget{}, false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(parsed.Hostname())), ".")
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		port = providerHostTelemetryDefaultPort(scheme)
	}
	return providerHostTelemetryTargetFromParts(scheme, host, port)
}

func providerHostTelemetryTargetFromParts(scheme string, host string, port string) (providerHostTelemetryTarget, bool) {
	if scheme != "http" && scheme != "https" {
		return providerHostTelemetryTarget{}, false
	}
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	port = strings.TrimSpace(port)
	if host == "" || port == "" {
		return providerHostTelemetryTarget{}, false
	}
	targetURL := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
		Path:   "/",
	}
	return providerHostTelemetryTarget{
		key:       strings.Join([]string{scheme, host, port}, "\x00"),
		targetURL: targetURL.String(),
		scheme:    scheme,
		host:      host,
		port:      port,
	}, true
}

func providerHostTelemetryDefaultPort(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func providerHostTelemetryView(target providerHostTelemetryTarget, point *providerHostTelemetryPoint) *managementv1.ProviderHostTelemetry {
	view := &managementv1.ProviderHostTelemetry{
		TargetUrl:    target.targetURL,
		Host:         target.host,
		Scheme:       target.scheme,
		Port:         target.port,
		Availability: managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_UNKNOWN,
		Reason:       "no_recent_sample",
	}
	if point == nil {
		return view
	}
	if point.latency != nil {
		view.LatencySeconds = *point.latency
	}
	if point.statusCode != nil {
		view.HttpStatusCode = int32(math.Round(*point.statusCode))
	}
	if point.sampledAt != nil {
		view.SampledAt = point.sampledAt
	}
	if point.success == nil {
		return view
	}
	if *point.success > 0 {
		view.Availability = managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_REACHABLE
		view.Reason = ""
		return view
	}
	view.Availability = managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_UNREACHABLE
	view.Reason = "probe_failed"
	return view
}

func sortedProviderHostTelemetry(items map[string]*managementv1.ProviderHostTelemetry) []*managementv1.ProviderHostTelemetry {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([]*managementv1.ProviderHostTelemetry, 0, len(keys))
	for _, key := range keys {
		out = append(out, cloneProviderHostTelemetry(items[key]))
	}
	return out
}

func cloneProviderHostTelemetry(item *managementv1.ProviderHostTelemetry) *managementv1.ProviderHostTelemetry {
	if item == nil {
		return nil
	}
	return proto.Clone(item).(*managementv1.ProviderHostTelemetry)
}

var _ providerService = (*HostTelemetryProviderService)(nil)
