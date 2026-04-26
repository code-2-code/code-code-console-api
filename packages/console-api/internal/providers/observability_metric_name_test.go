package providers

import "testing"

func TestCanonicalObservabilityMetricName(t *testing.T) {
	t.Run("cli oauth metric", func(t *testing.T) {
		got := canonicalObservabilityMetricName("gen_ai_provider_cli_oauth_codex_primary_window_used_percent")
		want := "gen_ai.provider.cli.oauth.codex.primary.window.used.percent"
		if got != want {
			t.Fatalf("canonicalObservabilityMetricName() = %q, want %q", got, want)
		}
	})

	t.Run("provider quota metric", func(t *testing.T) {
		got := canonicalObservabilityMetricName("gen_ai_provider_quota_remaining")
		want := "gen_ai.provider.quota.remaining"
		if got != want {
			t.Fatalf("canonicalObservabilityMetricName() = %q, want %q", got, want)
		}
	})

	t.Run("provider runtime rate limit metric", func(t *testing.T) {
		got := canonicalObservabilityMetricName("gen_ai_provider_runtime_rate_limit_remaining")
		want := "gen_ai.provider.runtime.rate_limit.remaining"
		if got != want {
			t.Fatalf("canonicalObservabilityMetricName() = %q, want %q", got, want)
		}
	})
}

func TestStorageObservabilityMetricName(t *testing.T) {
	t.Run("cli oauth metric", func(t *testing.T) {
		got := storageObservabilityMetricName("gen_ai.provider.cli.oauth.codex.primary.window.used.percent")
		want := "gen_ai_provider_cli_oauth_codex_primary_window_used_percent"
		if got != want {
			t.Fatalf("storageObservabilityMetricName() = %q, want %q", got, want)
		}
	})

	t.Run("provider quota metric", func(t *testing.T) {
		got := storageObservabilityMetricName("gen_ai.provider.quota.remaining")
		want := "gen_ai_provider_quota_remaining"
		if got != want {
			t.Fatalf("storageObservabilityMetricName() = %q, want %q", got, want)
		}
	})

	t.Run("provider runtime rate limit metric", func(t *testing.T) {
		got := storageObservabilityMetricName("gen_ai.provider.runtime.rate_limit.remaining")
		want := "gen_ai_provider_runtime_rate_limit_remaining"
		if got != want {
			t.Fatalf("storageObservabilityMetricName() = %q, want %q", got, want)
		}
	})
}
