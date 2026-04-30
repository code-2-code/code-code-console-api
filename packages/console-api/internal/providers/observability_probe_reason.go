package providers

import (
	"strings"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const providerObservabilityProbeReasonEnumPrefix = "PROVIDER_OBSERVABILITY_PROBE_REASON_"

func providerObservabilityProbeReasonKnownLabel(label string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(label))
	if normalized == "" {
		return false
	}
	value := providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UNSPECIFIED.
		Descriptor().
		Values().
		ByName(protoreflect.Name(providerObservabilityProbeReasonEnumPrefix + normalized))
	return value != nil && value.Number() != 0
}
