package providers

import (
	"fmt"
	"strings"
)

const (
	semanticCLIOAuthMetricPrefix     = "gen_ai.provider.cli.oauth."
	semanticVendorAPIKeyMetricPrefix = "gen_ai.provider.vendor.api_key."
	storageCLIOAuthMetricPrefix      = "gen_ai_provider_cli_oauth_"
	storageVendorAPIKeyMetricPrefix  = "gen_ai_provider_vendor_api_key_"
)

var semanticProviderMetricStorageNames = map[string]string{
	"gen_ai.provider.quota.limit":                         "gen_ai_provider_quota_limit",
	"gen_ai.provider.quota.usage":                         "gen_ai_provider_quota_usage",
	"gen_ai.provider.quota.remaining":                     "gen_ai_provider_quota_remaining",
	"gen_ai.provider.quota.remaining.fraction.percent":    "gen_ai_provider_quota_remaining_fraction_percent",
	"gen_ai.provider.quota.usage.fraction.percent":        "gen_ai_provider_quota_usage_fraction_percent",
	"gen_ai.provider.quota.reset.timestamp.seconds":       "gen_ai_provider_quota_reset_timestamp_seconds",
	"gen_ai.provider.runtime.quota.limit":                 "gen_ai_provider_runtime_quota_limit",
	"gen_ai.provider.runtime.quota.remaining":             "gen_ai_provider_runtime_quota_remaining",
	"gen_ai.provider.runtime.requests.total":              "gen_ai_provider_runtime_requests_total",
	"gen_ai.provider.runtime.rate_limit.events.total":     "gen_ai_provider_runtime_rate_limit_events_total",
	"gen_ai.provider.runtime.rate_limit.limit":            "gen_ai_provider_runtime_rate_limit_limit",
	"gen_ai.provider.runtime.rate_limit.remaining":        "gen_ai_provider_runtime_rate_limit_remaining",
	"gen_ai.provider.runtime.last_seen.timestamp.seconds": "gen_ai_provider_runtime_last_seen_timestamp_seconds",
	"gen_ai.provider.runtime.retry_after.seconds":         "gen_ai_provider_runtime_retry_after_seconds",
	"gen_ai.provider.usage.requests.count":                "gen_ai_provider_usage_requests_count",
	"gen_ai.provider.usage.tokens.count":                  "gen_ai_provider_usage_tokens_count",
	"gen_ai.provider.usage.cost.usd":                      "gen_ai_provider_usage_cost_usd",
}

var storageProviderMetricSemanticNames = map[string]string{
	"gen_ai_provider_quota_limit":                         "gen_ai.provider.quota.limit",
	"gen_ai_provider_quota_usage":                         "gen_ai.provider.quota.usage",
	"gen_ai_provider_quota_remaining":                     "gen_ai.provider.quota.remaining",
	"gen_ai_provider_quota_remaining_fraction_percent":    "gen_ai.provider.quota.remaining.fraction.percent",
	"gen_ai_provider_quota_usage_fraction_percent":        "gen_ai.provider.quota.usage.fraction.percent",
	"gen_ai_provider_quota_reset_timestamp_seconds":       "gen_ai.provider.quota.reset.timestamp.seconds",
	"gen_ai_provider_runtime_quota_limit":                 "gen_ai.provider.runtime.quota.limit",
	"gen_ai_provider_runtime_quota_remaining":             "gen_ai.provider.runtime.quota.remaining",
	"gen_ai_provider_runtime_requests_total":              "gen_ai.provider.runtime.requests.total",
	"gen_ai_provider_runtime_rate_limit_events_total":     "gen_ai.provider.runtime.rate_limit.events.total",
	"gen_ai_provider_runtime_rate_limit_limit":            "gen_ai.provider.runtime.rate_limit.limit",
	"gen_ai_provider_runtime_rate_limit_remaining":        "gen_ai.provider.runtime.rate_limit.remaining",
	"gen_ai_provider_runtime_last_seen_timestamp_seconds": "gen_ai.provider.runtime.last_seen.timestamp.seconds",
	"gen_ai_provider_runtime_retry_after_seconds":         "gen_ai.provider.runtime.retry_after.seconds",
	"gen_ai_provider_usage_requests_count":                "gen_ai.provider.usage.requests.count",
	"gen_ai_provider_usage_tokens_count":                  "gen_ai.provider.usage.tokens.count",
	"gen_ai_provider_usage_cost_usd":                      "gen_ai.provider.usage.cost.usd",
}

// metricRepo hides storage-specific metric naming/read-write details from callers.
type metricRepo interface {
	StorageName(semanticOrStorageName string) string
	SemanticName(storageOrSemanticName string) string
	LatestGaugeQuery(metricName string, matcher string) string
	LatestGaugeRangeQuery(metricName string, matcher string, window string) string
}

type prometheusMetricRepo struct{}

func newMetricRepo() metricRepo {
	return prometheusMetricRepo{}
}

func (prometheusMetricRepo) StorageName(semanticOrStorageName string) string {
	normalized := strings.TrimSpace(semanticOrStorageName)
	if normalized == "" {
		return ""
	}
	if storageName, ok := semanticProviderMetricStorageNames[normalized]; ok {
		return storageName
	}
	switch {
	case strings.HasPrefix(normalized, semanticCLIOAuthMetricPrefix):
		return storageCLIOAuthMetricPrefix + strings.ReplaceAll(strings.TrimPrefix(normalized, semanticCLIOAuthMetricPrefix), ".", "_")
	case strings.HasPrefix(normalized, semanticVendorAPIKeyMetricPrefix):
		return storageVendorAPIKeyMetricPrefix + strings.ReplaceAll(strings.TrimPrefix(normalized, semanticVendorAPIKeyMetricPrefix), ".", "_")
	case strings.Contains(normalized, "."):
		return strings.ReplaceAll(normalized, ".", "_")
	default:
		return normalized
	}
}

func (prometheusMetricRepo) SemanticName(storageOrSemanticName string) string {
	normalized := strings.TrimSpace(storageOrSemanticName)
	if normalized == "" {
		return ""
	}
	if semanticName, ok := storageProviderMetricSemanticNames[normalized]; ok {
		return semanticName
	}
	switch {
	case strings.HasPrefix(normalized, storageCLIOAuthMetricPrefix):
		return semanticCLIOAuthMetricPrefix + strings.ReplaceAll(strings.TrimPrefix(normalized, storageCLIOAuthMetricPrefix), "_", ".")
	case strings.HasPrefix(normalized, storageVendorAPIKeyMetricPrefix):
		return semanticVendorAPIKeyMetricPrefix + strings.ReplaceAll(strings.TrimPrefix(normalized, storageVendorAPIKeyMetricPrefix), "_", ".")
	default:
		return normalized
	}
}

func (r prometheusMetricRepo) LatestGaugeQuery(metricName string, matcher string) string {
	return fmt.Sprintf(
		`max without (%s) (%s{%s})`,
		runtimeGaugeInfrastructureLabels,
		r.StorageName(metricName),
		matcher,
	)
}

func (r prometheusMetricRepo) LatestGaugeRangeQuery(metricName string, matcher string, window string) string {
	return fmt.Sprintf(
		`max without (%s) (last_over_time(%s{%s}[%s]))`,
		runtimeGaugeInfrastructureLabels,
		r.StorageName(metricName),
		matcher,
		strings.TrimSpace(window),
	)
}
