package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

func TestObservabilityServiceProviderIncludesRuntimeGaugeMetricsWithoutActiveQueryProfile(t *testing.T) {
	prometheus := &codexRuntimeMetricPrometheusStub{}
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  codexRuntimeMetricProviderListerStub{},
		Support:    codexRuntimeMetricSupportStub{},
		Prometheus: prometheus,
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}

	response, err := service.Provider(context.Background(), "provider-openai", "1h", providerObservabilityViewFull)
	if err != nil {
		t.Fatalf("Provider() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(response.Items))
	}
	runtimeMetrics := response.Items[0].RuntimeMetrics
	if len(runtimeMetrics) != 1 {
		t.Fatalf("runtimeMetrics = %d, want 1", len(runtimeMetrics))
	}
	if got, want := runtimeMetrics[0].MetricName, "gen_ai.provider.cli.oauth.codex.primary.window.used.percent"; got != want {
		t.Fatalf("metricName = %q, want %q", got, want)
	}
	if len(runtimeMetrics[0].Rows) != 1 || runtimeMetrics[0].Rows[0].Value != 42 {
		t.Fatalf("rows = %#v, want one row with value 42", runtimeMetrics[0].Rows)
	}
	if _, ok := runtimeMetrics[0].Rows[0].Labels["provider_surface_binding_id"]; ok {
		t.Fatal("provider_surface_binding_id label should be removed")
	}
	if got, want := runtimeMetrics[0].Rows[0].Labels["cli_id"], "codex"; got != want {
		t.Fatalf("cli_id label = %q, want %q", got, want)
	}
	if strings.Contains(prometheus.lastQuery, "last_over_time(") {
		t.Fatalf("lastQuery = %q, want instant gauge query", prometheus.lastQuery)
	}
	if !strings.Contains(prometheus.lastQuery, "max without (job,instance,pod,namespace,service,endpoint,container)") {
		t.Fatalf("lastQuery = %q, want infrastructure label dedupe", prometheus.lastQuery)
	}
}

func TestObservabilityServiceSummaryOmitsRuntimeGaugeMetricScalars(t *testing.T) {
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  codexRuntimeMetricProviderListerStub{},
		Support:    codexRuntimeMetricSupportStub{},
		Prometheus: &codexRuntimeMetricPrometheusStub{},
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}

	response, err := service.Summary(context.Background(), "1h")
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), "runtimeMetrics") {
		t.Fatalf("summary payload unexpectedly contains runtimeMetrics: %s", payload)
	}
}

func TestObservabilityServiceProviderBatchesRuntimeGaugeMetricCardQueries(t *testing.T) {
	prometheus := &batchedRuntimeMetricPrometheusStub{}
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  codexRuntimeMetricProviderListerStub{},
		Support:    batchedRuntimeMetricSupportStub{},
		Prometheus: prometheus,
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}

	response, err := service.Provider(context.Background(), "provider-openai", "15m", providerObservabilityViewCard)
	if err != nil {
		t.Fatalf("Provider() error = %v", err)
	}
	if got, want := prometheus.queryCount, 1; got != want {
		t.Fatalf("queryCount = %d, want %d", got, want)
	}
	if !strings.Contains(prometheus.lastQuery, `__name__=~`) {
		t.Fatalf("lastQuery = %q, want batch metric selector", prometheus.lastQuery)
	}
	if !strings.Contains(prometheus.lastQuery, "last_over_time(") {
		t.Fatalf("lastQuery = %q, want card window query", prometheus.lastQuery)
	}
	runtimeMetrics := response.Items[0].RuntimeMetrics
	if len(runtimeMetrics) != 2 {
		t.Fatalf("runtimeMetrics = %d, want 2", len(runtimeMetrics))
	}
	if runtimeMetrics[0].Rows[0].Labels["pod"] != "" {
		t.Fatalf("pod infrastructure label should be removed: %#v", runtimeMetrics[0].Rows[0].Labels)
	}
}

type codexRuntimeMetricProviderListerStub struct{}

func (codexRuntimeMetricProviderListerStub) ListProviders(context.Context) ([]*managementv1.ProviderView, error) {
	return []*managementv1.ProviderView{{
		ProviderId: "provider-openai",
		Surfaces: []*managementv1.ProviderSurfaceBindingView{{
			SurfaceId: "provider-openai",
			Runtime:   testCLIProviderSurfaceRuntime("codex"),
		}},
	}}, nil
}

type codexRuntimeMetricSupportStub struct{}

func (codexRuntimeMetricSupportStub) ListCLIs(context.Context) ([]*supportv1.CLI, error) {
	return []*supportv1.CLI{{
		CliId:       "codex",
		DisplayName: "Codex CLI",
		Oauth: &supportv1.OAuthSupport{
			Observability: &observabilityv1.ObservabilityCapability{
				Profiles: []*observabilityv1.ObservabilityProfile{
					{
						ProfileId: "oauth_management_state",
						Metrics: []*observabilityv1.ObservabilityMetric{{
							Name:     refreshReadyMetric,
							Kind:     observabilityv1.ObservabilityMetricKind_OBSERVABILITY_METRIC_KIND_GAUGE,
							Category: observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_USAGE,
						}},
						Collection: &observabilityv1.ObservabilityProfile_ActiveQuery{
							ActiveQuery: &observabilityv1.ActiveQueryCollection{},
						},
					},
					{
						ProfileId: "oauth_runtime_headers",
						Metrics: []*observabilityv1.ObservabilityMetric{{
							Name:     "gen_ai.provider.cli.oauth.codex.primary.window.used.percent",
							Kind:     observabilityv1.ObservabilityMetricKind_OBSERVABILITY_METRIC_KIND_GAUGE,
							Category: observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_QUOTA,
						}},
					},
				},
			},
		},
	}}, nil
}

func (codexRuntimeMetricSupportStub) ListVendors(context.Context) ([]*supportv1.Vendor, error) {
	return nil, nil
}

type batchedRuntimeMetricSupportStub struct{}

func (batchedRuntimeMetricSupportStub) ListCLIs(context.Context) ([]*supportv1.CLI, error) {
	return []*supportv1.CLI{{
		CliId:       "codex",
		DisplayName: "Codex CLI",
		Oauth: &supportv1.OAuthSupport{
			Observability: &observabilityv1.ObservabilityCapability{
				Profiles: []*observabilityv1.ObservabilityProfile{{
					ProfileId: "oauth_runtime_headers",
					Metrics: []*observabilityv1.ObservabilityMetric{
						{
							Name:     "gen_ai.provider.cli.oauth.codex.primary.remaining.percent",
							Kind:     observabilityv1.ObservabilityMetricKind_OBSERVABILITY_METRIC_KIND_GAUGE,
							Category: observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_QUOTA,
						},
						{
							Name:     "gen_ai.provider.cli.oauth.codex.primary.window.used.percent",
							Kind:     observabilityv1.ObservabilityMetricKind_OBSERVABILITY_METRIC_KIND_GAUGE,
							Category: observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_QUOTA,
						},
					},
				}},
			},
		},
	}}, nil
}

func (batchedRuntimeMetricSupportStub) ListVendors(context.Context) ([]*supportv1.Vendor, error) {
	return nil, nil
}

type codexRuntimeMetricPrometheusStub struct {
	lastQuery string
}

func (s *codexRuntimeMetricPrometheusStub) QueryVector(_ context.Context, query string) ([]promVectorSample, error) {
	s.lastQuery = query
	switch {
	case containsMetricQuery(query, "gen_ai_provider_cli_oauth_codex_primary_window_used_percent"):
		return []promVectorSample{{
			Metric: map[string]string{"provider_surface_binding_id": "provider-openai", "cli_id": "codex"},
			Value:  42,
		}}, nil
	case containsMetricQuery(query, refreshReadyMetric):
		return []promVectorSample{{Metric: map[string]string{}, Value: 1}}, nil
	default:
		return nil, nil
	}
}

func (*codexRuntimeMetricPrometheusStub) QueryRange(_ context.Context, _ string, _ time.Time, _ time.Time, _ time.Duration) ([]promRangeSample, error) {
	return nil, nil
}

type batchedRuntimeMetricPrometheusStub struct {
	queryCount int
	lastQuery  string
}

func (s *batchedRuntimeMetricPrometheusStub) QueryVector(_ context.Context, query string) ([]promVectorSample, error) {
	s.queryCount++
	s.lastQuery = query
	return []promVectorSample{
		{
			Metric: map[string]string{
				"__name__":  "gen_ai_provider_cli_oauth_codex_primary_remaining_percent",
				"cli_id":    "codex",
				"pod":       "console-api-abc",
				"namespace": "code-code",
			},
			Value: 88,
		},
		{
			Metric: map[string]string{
				"__name__": "gen_ai_provider_cli_oauth_codex_primary_window_used_percent",
				"cli_id":   "codex",
				"pod":      "console-api-def",
			},
			Value: 42,
		},
	}, nil
}

func (*batchedRuntimeMetricPrometheusStub) QueryRange(_ context.Context, _ string, _ time.Time, _ time.Time, _ time.Duration) ([]promRangeSample, error) {
	return nil, nil
}

func containsMetricQuery(query string, metricName string) bool {
	return strings.Contains(query, storageObservabilityMetricName(metricName))
}
