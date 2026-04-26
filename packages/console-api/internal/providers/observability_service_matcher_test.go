package providers

import (
	"strings"
	"testing"
)

func TestProviderRegexSingleProviderIncludesFilter(t *testing.T) {
	subject := &cliSubject{
		providerIDs: map[string]struct{}{"provider-openai": {}},
	}
	if got, want := providerRegex(subject), "provider-openai"; got != want {
		t.Fatalf("providerRegex() = %q, want %q", got, want)
	}
}

func TestProviderRegexMultipleProvidersIncludesRegex(t *testing.T) {
	subject := &cliSubject{
		providerIDs: map[string]struct{}{
			"provider-alpha": {},
			"provider-beta":  {},
		},
	}
	got := providerRegex(subject)
	want := "provider-alpha|provider-beta"
	if got != want {
		t.Fatalf("providerRegex() = %q, want %q", got, want)
	}
}

func TestPromActiveDiscoveryMatcherIncludesProviderFilterWhenSingleProvider(t *testing.T) {
	matcher := promActiveDiscoveryMatcher(&cliSubject{
		matcherLabel: "cli_id",
		ownerID:      "codex",
		providerIDs: map[string]struct{}{
			"provider-openai": {},
		},
	})
	if got, want := matcher, `cli_id="codex"`; !strings.Contains(matcher, want) {
		t.Fatalf("promActiveDiscoveryMatcher() = %q, want contains %q", got, want)
	}
	if !strings.Contains(matcher, `provider_id=~"provider-openai"`) {
		t.Fatalf("promActiveDiscoveryMatcher() = %q, want single provider filter", matcher)
	}
}

func TestPromActiveDiscoveryMatcherIncludesProviderFilterForMultipleProviders(t *testing.T) {
	matcher := promActiveDiscoveryMatcher(&cliSubject{
		matcherLabel: "cli_id",
		ownerID:      "codex",
		providerIDs: map[string]struct{}{
			"provider-alpha": {},
			"provider-beta":  {},
		},
	})
	if !strings.Contains(matcher, `cli_id="codex"`) {
		t.Fatalf("promActiveDiscoveryMatcher() = %q, want contains %q", matcher, `cli_id="codex"`)
	}
	if !strings.Contains(matcher, `provider_id=~"provider-alpha|provider-beta"`) {
		t.Fatalf("promActiveDiscoveryMatcher() = %q, want provider filter", matcher)
	}
}

func TestActiveProbeMetricNamesMatchAuthServiceOperationMetrics(t *testing.T) {
	for _, metric := range []string{
		cliProbeRunsMetric,
		cliProbeLastRunMetric,
		cliProbeLastOutcomeMetric,
		cliProbeLastReasonMetric,
		cliProbeNextAllowedMetric,
		vendorProbeRunsMetric,
		vendorProbeLastRunMetric,
		vendorProbeLastOutcomeMetric,
		vendorProbeLastReasonMetric,
		vendorProbeNextMetric,
	} {
		if !strings.Contains(metric, ".active.operation.") {
			t.Fatalf("metric %q should use active.operation namespace", metric)
		}
	}
}
