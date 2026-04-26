package providers

type ProviderObservabilitySummaryResponse struct {
	Window      string                         `json:"window"`
	GeneratedAt string                         `json:"generatedAt"`
	Items       []ProviderCLIObservabilityCard `json:"items"`
}

type ProviderCLIObservabilityCard struct {
	Owner         string                               `json:"owner"`
	CLIID         string                               `json:"cliId,omitempty"`
	VendorID      string                               `json:"vendorId,omitempty"`
	DisplayName   string                               `json:"displayName"`
	IconURL       string                               `json:"iconUrl"`
	ProviderCount int                                  `json:"providerCount"`
	InstanceCount int                                  `json:"instanceCount"`
	Probe         *ProviderProbeOutcomeSummary         `json:"probe,omitempty"`
	Refresh       *ProviderReadinessSummary            `json:"refresh,omitempty"`
	Runtime       *ProviderRuntimeRequestStatusSummary `json:"runtime,omitempty"`
}

type ProviderProbeOutcomeSummary struct {
	Total       float64 `json:"total"`
	Executed    float64 `json:"executed"`
	Throttled   float64 `json:"throttled"`
	AuthBlocked float64 `json:"authBlocked"`
	Unsupported float64 `json:"unsupported"`
	Failed      float64 `json:"failed"`
}

type ProviderReadinessSummary struct {
	Ready float64 `json:"ready"`
	Total float64 `json:"total"`
}

type ProviderRuntimeRequestStatusSummary struct {
	Total     float64 `json:"total"`
	Status2xx float64 `json:"status2xx"`
	Status3xx float64 `json:"status3xx"`
	Status4xx float64 `json:"status4xx"`
	Status5xx float64 `json:"status5xx"`
}

type ProviderObservabilityResponse struct {
	ProviderID  string                         `json:"providerId"`
	Window      string                         `json:"window"`
	GeneratedAt string                         `json:"generatedAt"`
	Items       []ProviderCLIObservabilityItem `json:"items"`
}

type ProviderCLIObservabilityItem struct {
	Owner                     string                            `json:"owner"`
	CLIID                     string                            `json:"cliId,omitempty"`
	VendorID                  string                            `json:"vendorId,omitempty"`
	DisplayName               string                            `json:"displayName"`
	IconURL                   string                            `json:"iconUrl"`
	ProviderSurfaceBindingIDs []string                          `json:"providerSurfaceBindingIds"`
	ProbeOutcomes             []ProviderLabelValue              `json:"probeOutcomes,omitempty"`
	ProbeOutcomeSeries        []ProviderMetricSeries            `json:"probeOutcomeSeries,omitempty"`
	RefreshAttempts           []ProviderLabelValue              `json:"refreshAttempts,omitempty"`
	RefreshAttemptSeries      []ProviderMetricSeries            `json:"refreshAttemptSeries,omitempty"`
	RuntimeRequests           []ProviderLabelValue              `json:"runtimeRequests,omitempty"`
	RuntimeRequestSeries      []ProviderMetricSeries            `json:"runtimeRequestSeries,omitempty"`
	RuntimeRateLimits         []ProviderLabelValue              `json:"runtimeRateLimits,omitempty"`
	RuntimeRateLimitSeries    []ProviderMetricSeries            `json:"runtimeRateLimitSeries,omitempty"`
	RuntimeMetrics            []ProviderRuntimeMetricRows       `json:"runtimeMetrics,omitempty"`
	LastProbeRun              []ProviderSurfaceBindingTimestamp `json:"lastProbeRun,omitempty"`
	LastProbeOutcome          []ProviderSurfaceBindingValue     `json:"lastProbeOutcome,omitempty"`
	LastProbeReason           []ProviderSurfaceBindingReason    `json:"lastProbeReason,omitempty"`
	NextProbeAllowed          []ProviderSurfaceBindingTimestamp `json:"nextProbeAllowed,omitempty"`
	AuthUsable                []ProviderSurfaceBindingValue     `json:"authUsable,omitempty"`
	CredentialLastUsed        []ProviderSurfaceBindingTimestamp `json:"credentialLastUsed,omitempty"`
	RefreshReady              []ProviderSurfaceBindingReadiness `json:"refreshReady,omitempty"`
	LastRuntimeSeen           []ProviderSurfaceBindingTimestamp `json:"lastRuntimeSeen,omitempty"`
}

type ProviderLabelValue struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type ProviderSurfaceBindingTimestamp struct {
	ProviderSurfaceBindingID string `json:"providerSurfaceBindingId,omitempty"`
	Timestamp                string `json:"timestamp"`
}

type ProviderSurfaceBindingValue struct {
	ProviderSurfaceBindingID string  `json:"providerSurfaceBindingId,omitempty"`
	Value                    float64 `json:"value"`
}

type ProviderSurfaceBindingReason struct {
	ProviderSurfaceBindingID string `json:"providerSurfaceBindingId,omitempty"`
	Reason                   string `json:"reason"`
}

type ProviderSurfaceBindingReadiness struct {
	ProviderSurfaceBindingID string  `json:"providerSurfaceBindingId,omitempty"`
	Value                    float64 `json:"value"`
}

type ProviderMetricRow struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

type ProviderRuntimeMetricRows struct {
	MetricName  string              `json:"metricName"`
	DisplayName string              `json:"displayName"`
	Unit        string              `json:"unit,omitempty"`
	Category    string              `json:"category,omitempty"`
	Rows        []ProviderMetricRow `json:"rows,omitempty"`
}

type ProviderMetricSeries struct {
	Label  string                    `json:"label"`
	Points []ProviderTimeSeriesPoint `json:"points"`
}

type ProviderTimeSeriesPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}
