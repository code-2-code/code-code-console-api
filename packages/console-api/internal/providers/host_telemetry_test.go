package providers

import (
	"context"
	"testing"
	"time"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestHostTelemetryProviderServiceListProvidersQueriesOnceAndMatchesByHost(t *testing.T) {
	prom := &hostTelemetryPrometheusStub{
		samples: []promVectorSample{
			hostTelemetrySample("probe_success", "https", "api.openai.com", "443", 1, time.Unix(100, 0)),
			hostTelemetrySample("probe_duration_seconds", "https", "api.openai.com", "443", 0.123, time.Unix(100, 0)),
			hostTelemetrySample("probe_http_status_code", "https", "api.openai.com", "443", 401, time.Unix(100, 0)),
		},
	}
	service, err := NewHostTelemetryProviderService(hostTelemetryProviderStub{
		items: []*managementv1.ProviderView{{
			ProviderId: "provider-openai",
			Surfaces: []*managementv1.ProviderSurfaceBindingView{{
				SurfaceId: "openai-api",
				Runtime:   hostTelemetryAPIRuntime("https://api.openai.com/v1"),
			}},
		}},
	}, prom)
	if err != nil {
		t.Fatalf("NewHostTelemetryProviderService() error = %v", err)
	}

	items, err := service.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if got, want := prom.queryCount, 1; got != want {
		t.Fatalf("queryCount = %d, want %d", got, want)
	}
	if got, want := prom.lastQuery, providerHostTelemetryQuery; got != want {
		t.Fatalf("lastQuery = %q, want %q", got, want)
	}
	telemetry := items[0].GetSurfaces()[0].GetHostTelemetry()
	if telemetry.GetAvailability() != managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_REACHABLE {
		t.Fatalf("availability = %v, want reachable", telemetry.GetAvailability())
	}
	if got, want := telemetry.GetHost(), "api.openai.com"; got != want {
		t.Fatalf("host = %q, want %q", got, want)
	}
	if got, want := telemetry.GetLatencySeconds(), 0.123; got != want {
		t.Fatalf("latency = %v, want %v", got, want)
	}
	if got, want := telemetry.GetHttpStatusCode(), int32(401); got != want {
		t.Fatalf("http status = %d, want %d", got, want)
	}
	if got, want := len(items[0].GetHostTelemetry()), 1; got != want {
		t.Fatalf("provider host telemetry count = %d, want %d", got, want)
	}
}

func TestHostTelemetryProviderServiceListProvidersReturnsUnknownWithoutSample(t *testing.T) {
	service, err := NewHostTelemetryProviderService(hostTelemetryProviderStub{
		items: []*managementv1.ProviderView{{
			ProviderId: "provider-custom",
			Surfaces: []*managementv1.ProviderSurfaceBindingView{{
				SurfaceId: "custom-api",
				Runtime:   hostTelemetryAPIRuntime("https://custom.example.test/v1"),
			}},
		}},
	}, &hostTelemetryPrometheusStub{})
	if err != nil {
		t.Fatalf("NewHostTelemetryProviderService() error = %v", err)
	}

	items, err := service.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	telemetry := items[0].GetSurfaces()[0].GetHostTelemetry()
	if telemetry.GetAvailability() != managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_UNKNOWN {
		t.Fatalf("availability = %v, want unknown", telemetry.GetAvailability())
	}
	if got, want := telemetry.GetReason(), "no_recent_sample"; got != want {
		t.Fatalf("reason = %q, want %q", got, want)
	}
}

type hostTelemetryProviderStub struct {
	items []*managementv1.ProviderView
}

func (s hostTelemetryProviderStub) ListProviderSurfaceMetadata(context.Context) ([]*providerv1.ProviderSurface, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) ListProviders(context.Context) ([]*managementv1.ProviderView, error) {
	return s.items, nil
}

func (s hostTelemetryProviderStub) ListProviderSurfaceBindings(context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) UpdateProvider(context.Context, string, *managementv1.UpdateProviderRequest) (*managementv1.ProviderView, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) UpdateProviderAuthentication(context.Context, string, *managementv1.UpdateProviderAuthenticationRequest) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) UpdateProviderObservabilityAuthentication(context.Context, string, *managementv1.UpdateProviderObservabilityAuthenticationRequest) (*managementv1.ProviderView, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) DeleteProvider(context.Context, string) error { return nil }

func (s hostTelemetryProviderStub) CreateProviderSurfaceBinding(context.Context, *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) UpdateProviderSurfaceBinding(context.Context, string, *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) DeleteProviderSurfaceBinding(context.Context, string) error {
	return nil
}

func (s hostTelemetryProviderStub) Connect(context.Context, *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) GetConnectSession(context.Context, string) (*managementv1.ProviderConnectSessionView, error) {
	return nil, nil
}

func (s hostTelemetryProviderStub) WatchStatusEvents(context.Context, []string, func(*managementv1.ProviderStatusEvent) error) error {
	return nil
}

type hostTelemetryPrometheusStub struct {
	queryCount int
	lastQuery  string
	samples    []promVectorSample
}

func (s *hostTelemetryPrometheusStub) QueryVector(_ context.Context, query string) ([]promVectorSample, error) {
	s.queryCount++
	s.lastQuery = query
	return s.samples, nil
}

func (s *hostTelemetryPrometheusStub) QueryRange(context.Context, string, time.Time, time.Time, time.Duration) ([]promRangeSample, error) {
	return nil, nil
}

func hostTelemetryAPIRuntime(baseURL string) *providerv1.ProviderSurfaceRuntime {
	return &providerv1.ProviderSurfaceRuntime{
		DisplayName: "api",
		Access: &providerv1.ProviderSurfaceRuntime_Api{
			Api: &providerv1.ProviderAPISurfaceRuntime{BaseUrl: baseURL},
		},
	}
}

func hostTelemetrySample(metricName string, scheme string, host string, port string, value float64, timestamp time.Time) promVectorSample {
	return promVectorSample{
		Metric: map[string]string{
			"__name__": metricName,
			"job":      "provider-host-latency",
			"scheme":   scheme,
			"host":     host,
			"port":     port,
		},
		Value:     value,
		Timestamp: timestamp,
	}
}
