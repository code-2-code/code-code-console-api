package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	vendordefinitionv1 "code-code.internal/go-contract/vendor_definition/v1"
)

func TestRegisterObservabilityHandlersSummary(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/summary?window=15m", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"cliId":"codex"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersRejectInvalidWindow(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/providers/provider-openai?window=2h", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProbeAll(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusStub{},
		Prober:     observabilityProbeStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/providers/observability:probe-all", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"triggeredCount":1`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"message":"probe completed"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProbeAllRejectsNilProber(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/providers/observability:probe-all", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"provider_observability_probe_all_failed"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"message":"internal server error"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProbeAllIncludesVendorTarget(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers: observabilityProviderListerStub{
			items: []*managementv1.ProviderView{{
				ProviderId: "provider-cli",
				Surfaces: []*managementv1.ProviderSurfaceBindingView{{
					SurfaceId: "provider-openai",
					Runtime:   testCLIProviderSurfaceRuntime("codex"),
				}},
			}, {
				ProviderId: "provider-vendor",
				Surfaces: []*managementv1.ProviderSurfaceBindingView{{
					SurfaceId: "provider-minimax",
					VendorId:  "minimax",
					Runtime:   testAPIProviderSurfaceRuntime(),
				}},
			}},
		},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusStub{},
		Prober: observabilityProbeStub{
			responsesByProvider: map[string]*managementv1.ProbeProviderObservabilityResponse{
				"provider-cli": {
					ProviderId: "provider-cli",
					CliId:      "codex",
					Outcome:    managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED,
					Message:    "provider probe completed",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/providers/observability:probe-all", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"triggeredCount":2`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersSummaryIncludesVendor(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers: observabilityProviderListerStub{
			items: []*managementv1.ProviderView{{
				ProviderId: "provider-minimax",
				Surfaces: []*managementv1.ProviderSurfaceBindingView{{
					SurfaceId: "provider-minimax",
					VendorId:  "minimax",
					Runtime:   testAPIProviderSurfaceRuntime(),
				}},
			}},
		},
		Support: observabilitySupportStub{
			vendors: []*supportv1.Vendor{{
				Vendor: &vendordefinitionv1.Vendor{
					VendorId:    "minimax",
					DisplayName: "MiniMax",
				},
				ProviderBindings: []*supportv1.VendorProviderBinding{{
					Observability: &observabilityv1.ObservabilityCapability{
						Profiles: []*observabilityv1.ObservabilityProfile{{
							Metrics: []*observabilityv1.ObservabilityMetric{{
								Name: "gen_ai.provider.quota.remaining",
							}},
							Collection: &observabilityv1.ObservabilityProfile_ActiveQuery{
								ActiveQuery: &observabilityv1.ActiveQueryCollection{},
							},
						}},
					},
				}},
			}},
		},
		Prometheus: observabilityPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/summary?window=15m", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"owner":"vendor"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"vendorId":"minimax"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersSummaryReturnsPartialDataOnPrometheusError(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusErrorStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/summary?window=15m", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"owner":"cli"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"provider_observability_summary_failed"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProviderReturnsPartialDataOnPrometheusError(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusErrorStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/providers/provider-openai?window=15m", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"providerSurfaceBindingIds":["provider-openai"]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"provider_observability_failed"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProviderIncludesLastProbeOutcome(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/providers/provider-openai?window=15m", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"lastProbeOutcome":[{"value":1}]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"authUsable":[{"value":1}]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"credentialLastUsed":[{"timestamp":"2024-04-18T15:00:00Z"}]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProviderStatusViewOmitsRuntimeMetrics(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/providers/provider-openai?window=15m&view=status", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"lastProbeOutcome":[{"value":1}]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"runtimeMetrics"`) {
		t.Fatalf("response unexpectedly contains runtimeMetrics: %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProviderStatusViewOmitsStaleProbeReasonAfterSuccess(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityProbeReasonPrometheusStub{outcome: 1},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/providers/provider-openai?window=1h&view=status", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"lastProbeOutcome":[{"value":1}]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"lastProbeReason"`) {
		t.Fatalf("response unexpectedly contains stale lastProbeReason: %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProviderStatusViewKeepsProbeReasonForFailure(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  observabilityProviderListerStub{},
		Support:    observabilitySupportStub{},
		Prometheus: observabilityProbeReasonPrometheusStub{outcome: 5},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/providers/provider-openai?window=1h&view=status", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"lastProbeOutcome":[{"value":5}]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"lastProbeReason":[{"reason":"PROBE_FAILED"}]`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterObservabilityHandlersProviderCardViewOmitsProbeSeries(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers: observabilityProviderListerStub{
			items: []*managementv1.ProviderView{{
				ProviderId: "provider-minimax",
				Surfaces: []*managementv1.ProviderSurfaceBindingView{{
					SurfaceId: "provider-minimax",
					VendorId:  "minimax",
					Runtime:   testAPIProviderSurfaceRuntime(),
				}},
			}},
		},
		Support: observabilitySupportStub{
			vendors: []*supportv1.Vendor{{
				Vendor: &vendordefinitionv1.Vendor{
					VendorId:    "minimax",
					DisplayName: "MiniMax",
				},
				ProviderBindings: []*supportv1.VendorProviderBinding{{
					Observability: &observabilityv1.ObservabilityCapability{
						Profiles: []*observabilityv1.ObservabilityProfile{{
							Metrics: []*observabilityv1.ObservabilityMetric{{
								Name:     "gen_ai.provider.quota.remaining",
								Kind:     observabilityv1.ObservabilityMetricKind_OBSERVABILITY_METRIC_KIND_GAUGE,
								Category: observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_QUOTA,
							}},
							Collection: &observabilityv1.ObservabilityProfile_ActiveQuery{
								ActiveQuery: &observabilityv1.ActiveQueryCollection{},
							},
						}},
					},
				}},
			}},
		},
		Prometheus: observabilityPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}
	mux := http.NewServeMux()
	RegisterObservabilityHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/observability/providers/provider-minimax?window=15m&view=card", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"runtimeMetrics"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"probeOutcomeSeries"`) {
		t.Fatalf("response unexpectedly contains probeOutcomeSeries: %s", recorder.Body.String())
	}
}

type observabilityProviderListerStub struct {
	items []*managementv1.ProviderView
}

func (s observabilityProviderListerStub) ListProviders(context.Context) ([]*managementv1.ProviderView, error) {
	if s.items != nil {
		return s.items, nil
	}
	return []*managementv1.ProviderView{
		{
			ProviderId: "provider-openai",
			Surfaces: []*managementv1.ProviderSurfaceBindingView{
				{
					SurfaceId: "provider-openai",
					Runtime:   testCLIProviderSurfaceRuntime("codex"),
				},
			},
		},
	}, nil
}

type observabilitySupportStub struct {
	clis    []*supportv1.CLI
	vendors []*supportv1.Vendor
}

func (s observabilitySupportStub) ListCLIs(context.Context) ([]*supportv1.CLI, error) {
	if s.clis != nil {
		return s.clis, nil
	}
	return []*supportv1.CLI{
		{
			CliId:       "codex",
			DisplayName: "Codex CLI",
			IconUrl:     "https://example.com/icon.svg",
			Oauth: &supportv1.OAuthSupport{
				Observability: &observabilityv1.ObservabilityCapability{
					Profiles: []*observabilityv1.ObservabilityProfile{
						{
							Metrics: []*observabilityv1.ObservabilityMetric{
								{Name: refreshReadyMetric},
								{Name: runtimeRequestsMetric},
							},
							Collection: &observabilityv1.ObservabilityProfile_ActiveQuery{
								ActiveQuery: &observabilityv1.ActiveQueryCollection{},
							},
						},
					},
				},
			},
		},
	}, nil
}

func (s observabilitySupportStub) ListVendors(context.Context) ([]*supportv1.Vendor, error) {
	if s.vendors != nil {
		return s.vendors, nil
	}
	return nil, nil
}

type observabilityPrometheusStub struct{}

type observabilityProbeReasonPrometheusStub struct {
	outcome float64
}

type observabilityProbeStub struct {
	errorsByProvider    map[string]error
	responsesByProvider map[string]*managementv1.ProbeProviderObservabilityResponse
}

type observabilityProbeFunc func(context.Context, []string) (*managementv1.ProbeProviderObservabilityResponse, error)

func (f observabilityProbeFunc) ProbeProvidersObservability(
	ctx context.Context,
	providerIDs []string,
) (*managementv1.ProbeProviderObservabilityResponse, error) {
	return f(ctx, providerIDs)
}

func (s observabilityProbeStub) ProbeProvidersObservability(
	_ context.Context,
	providerIDs []string,
) (*managementv1.ProbeProviderObservabilityResponse, error) {
	providerID := firstProviderID(providerIDs)
	if s.errorsByProvider != nil {
		if err := s.errorsByProvider[providerID]; err != nil {
			return nil, err
		}
	}
	if s.responsesByProvider != nil {
		if response, ok := s.responsesByProvider[providerID]; ok {
			return response, nil
		}
	}
	return &managementv1.ProbeProviderObservabilityResponse{
		ProviderId:    providerID,
		ProviderIds:   providerIDs,
		CliId:         "codex",
		Outcome:       managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED,
		Message:       "probe completed",
		NextAllowedAt: "",
	}, nil
}

func (observabilityPrometheusStub) QueryVector(_ context.Context, query string) ([]promVectorSample, error) {
	switch {
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_runs_total"):
		return []promVectorSample{
			{Metric: map[string]string{"outcome": "executed"}, Value: 2},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_last_outcome"):
		return []promVectorSample{
			{Metric: map[string]string{}, Value: 1},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_auth_usable"):
		return []promVectorSample{
			{Metric: map[string]string{}, Value: 1},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_credential_last_used_timestamp_seconds"):
		return []promVectorSample{
			{Metric: map[string]string{}, Value: 1713452400},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_last_run_timestamp_seconds"):
		return []promVectorSample{
			{Metric: map[string]string{}, Value: 1713452400},
		}, nil
	case strings.Contains(query, "gen_ai_provider_vendor_api_key_active_operation_runs_total"):
		return []promVectorSample{
			{Metric: map[string]string{"outcome": "executed"}, Value: 1},
		}, nil
	case strings.Contains(query, "gen_ai_provider_vendor_api_key_active_operation_last_outcome"):
		return []promVectorSample{
			{Metric: map[string]string{}, Value: 1},
		}, nil
	case strings.Contains(query, "gen_ai_provider_quota_remaining"):
		return []promVectorSample{
			{
				Metric: map[string]string{
					"provider_surface_binding_id": "provider-minimax",
					"model_id":                    "MiniMax-M1",
				},
				Value: 42,
			},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_refresh_ready"):
		return []promVectorSample{
			{Metric: map[string]string{}, Value: 1},
		}, nil
	case strings.Contains(query, "gen_ai_provider_runtime_requests_total"):
		return []promVectorSample{
			{Metric: map[string]string{"status_class": "2xx"}, Value: 10},
		}, nil
	default:
		return nil, nil
	}
}

func (s observabilityProbeReasonPrometheusStub) QueryVector(_ context.Context, query string) ([]promVectorSample, error) {
	const providerID = "provider-openai"
	switch {
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_runs_total"):
		return []promVectorSample{
			{Metric: map[string]string{"outcome": "executed", "provider_id": providerID}, Value: 1},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_last_run_timestamp_seconds"):
		return []promVectorSample{
			{Metric: map[string]string{"provider_id": providerID}, Value: 1713452400},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_last_outcome"):
		return []promVectorSample{
			{Metric: map[string]string{"provider_id": providerID}, Value: s.outcome},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_auth_usable"):
		value := float64(1)
		if s.outcome == 3 {
			value = 0
		}
		return []promVectorSample{
			{Metric: map[string]string{"provider_id": providerID}, Value: value},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_credential_last_used_timestamp_seconds"):
		return []promVectorSample{
			{Metric: map[string]string{"provider_id": providerID}, Value: 1713452400},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_last_reason"):
		return []promVectorSample{
			{Metric: map[string]string{"provider_id": providerID, "reason": "PROBE_FAILED"}, Value: 1},
		}, nil
	case strings.Contains(query, "gen_ai_provider_cli_oauth_active_operation_next_allowed"):
		return []promVectorSample{
			{Metric: map[string]string{"provider_id": providerID}, Value: 1713452700},
		}, nil
	default:
		return nil, nil
	}
}

func (observabilityProbeReasonPrometheusStub) QueryRange(_ context.Context, _ string, _ time.Time, _ time.Time, _ time.Duration) ([]promRangeSample, error) {
	return nil, nil
}

func (observabilityPrometheusStub) QueryRange(_ context.Context, _ string, _ time.Time, _ time.Time, _ time.Duration) ([]promRangeSample, error) {
	return nil, nil
}

type observabilityPrometheusErrorStub struct{}

func (observabilityPrometheusErrorStub) QueryVector(_ context.Context, _ string) ([]promVectorSample, error) {
	return nil, fmt.Errorf("prometheus unavailable")
}

func (observabilityPrometheusErrorStub) QueryRange(_ context.Context, _ string, _ time.Time, _ time.Time, _ time.Duration) ([]promRangeSample, error) {
	return nil, fmt.Errorf("prometheus unavailable")
}
