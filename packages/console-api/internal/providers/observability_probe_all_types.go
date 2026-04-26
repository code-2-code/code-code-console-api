package providers

type ProviderObservabilityProbeAllResponse struct {
	TriggeredCount int                               `json:"triggeredCount"`
	WorkflowID     string                            `json:"workflowId,omitempty"`
	Message        string                            `json:"message,omitempty"`
	Results        []ProviderObservabilityProbeState `json:"results,omitempty"`
}

type ProviderObservabilityProbeRequest struct {
	ProviderIDs []string `json:"providerIds"`
}

type ProviderObservabilityProbeState struct {
	ProviderID    string `json:"providerId,omitempty"`
	Owner         string `json:"owner,omitempty"`
	CLIID         string `json:"cliId,omitempty"`
	VendorID      string `json:"vendorId,omitempty"`
	Outcome       string `json:"outcome,omitempty"`
	Message       string `json:"message,omitempty"`
	LastAttemptAt string `json:"lastAttemptAt,omitempty"`
	NextAllowedAt string `json:"nextAllowedAt,omitempty"`
}
