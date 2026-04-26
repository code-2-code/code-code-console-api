package providers

import "strings"

type providerObservabilityView string

const (
	providerObservabilityViewFull   providerObservabilityView = "full"
	providerObservabilityViewStatus providerObservabilityView = "status"
	providerObservabilityViewCard   providerObservabilityView = "card"
)

func parseProviderObservabilityView(raw string) (providerObservabilityView, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "full":
		return providerObservabilityViewFull, nil
	case "status":
		return providerObservabilityViewStatus, nil
	case "card":
		return providerObservabilityViewCard, nil
	default:
		return "", &providerObservabilityViewError{raw: raw}
	}
}

func (view providerObservabilityView) includesFullDetail() bool {
	return view == providerObservabilityViewFull
}

func (view providerObservabilityView) includesStatus() bool {
	return view == providerObservabilityViewFull || view == providerObservabilityViewStatus
}

func (view providerObservabilityView) includesCard() bool {
	return view == providerObservabilityViewFull || view == providerObservabilityViewCard
}

func (view providerObservabilityView) includesObservedAt() bool {
	return view.includesStatus() || view.includesCard()
}

type providerObservabilityViewError struct {
	raw string
}

func (e *providerObservabilityViewError) Error() string {
	return "view must be one of full, status, card"
}
