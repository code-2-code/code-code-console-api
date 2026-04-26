package providers

import (
	"context"
	"strings"
	"testing"
	"time"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	vendordefinitionv1 "code-code.internal/go-contract/vendor_definition/v1"
)

func TestObservabilityServiceProviderIncludesVendorRuntimeGaugeMetrics(t *testing.T) {
	prometheus := &vendorRuntimeMetricPrometheusStub{}
	service, err := NewObservabilityService(ObservabilityServiceConfig{
		Providers:  vendorRuntimeMetricProviderListerStub{},
		Support:    vendorRuntimeMetricSupportStub{},
		Prometheus: prometheus,
	})
	if err != nil {
		t.Fatalf("NewObservabilityService() error = %v", err)
	}

	response, err := service.Provider(context.Background(), "provider-minimax", "1h", providerObservabilityViewFull)
	if err != nil {
		t.Fatalf("Provider() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(response.Items))
	}
	item := response.Items[0]
	if got, want := item.Owner, ownerKindVendor; got != want {
		t.Fatalf("owner = %q, want %q", got, want)
	}
	if got, want := item.VendorID, "minimax"; got != want {
		t.Fatalf("vendorId = %q, want %q", got, want)
	}
	if len(item.RuntimeMetrics) != 1 {
		t.Fatalf("runtimeMetrics = %d, want 1", len(item.RuntimeMetrics))
	}
	if got, want := item.RuntimeMetrics[0].MetricName, "gen_ai.provider.quota.remaining"; got != want {
		t.Fatalf("metricName = %q, want %q", got, want)
	}
	if len(item.RuntimeMetrics[0].Rows) != 1 || item.RuntimeMetrics[0].Rows[0].Value != 128 {
		t.Fatalf("rows = %#v, want one row with value 128", item.RuntimeMetrics[0].Rows)
	}
	if got, want := item.RuntimeMetrics[0].Rows[0].Labels["provider_id"], "provider-minimax"; got != want {
		t.Fatalf("provider_id label = %q, want %q", got, want)
	}
	if !strings.Contains(prometheus.lastQuery, `vendor_id="minimax"`) {
		t.Fatalf("lastQuery = %q, want vendor matcher", prometheus.lastQuery)
	}
	if strings.Contains(prometheus.lastQuery, "last_over_time(") {
		t.Fatalf("lastQuery = %q, want instant gauge query", prometheus.lastQuery)
	}
}

type vendorRuntimeMetricProviderListerStub struct{}

func (vendorRuntimeMetricProviderListerStub) ListProviders(context.Context) ([]*managementv1.ProviderView, error) {
	return []*managementv1.ProviderView{{
		ProviderId: "provider-minimax",
		Surfaces: []*managementv1.ProviderSurfaceBindingView{{
			SurfaceId: "provider-minimax",
			VendorId:  "minimax",
			Runtime:   testAPIProviderSurfaceRuntime(),
		}},
	}}, nil
}

type vendorRuntimeMetricSupportStub struct{}

func (vendorRuntimeMetricSupportStub) ListCLIs(context.Context) ([]*supportv1.CLI, error) {
	return nil, nil
}

func (vendorRuntimeMetricSupportStub) ListVendors(context.Context) ([]*supportv1.Vendor, error) {
	return []*supportv1.Vendor{{
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
	}}, nil
}

type vendorRuntimeMetricPrometheusStub struct {
	lastQuery string
}

func (s *vendorRuntimeMetricPrometheusStub) QueryVector(_ context.Context, query string) ([]promVectorSample, error) {
	s.lastQuery = query
	if strings.Contains(query, "gen_ai_provider_quota_remaining") {
		return []promVectorSample{{
			Metric: map[string]string{
				"provider_id": "provider-minimax",
				"vendor_id":   "minimax",
			},
			Value: 128,
		}}, nil
	}
	return nil, nil
}

func (*vendorRuntimeMetricPrometheusStub) QueryRange(_ context.Context, _ string, _ time.Time, _ time.Time, _ time.Duration) ([]promRangeSample, error) {
	return nil, nil
}
