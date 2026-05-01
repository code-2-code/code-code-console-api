package providers

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

const (
	ownerKindCLI    = "cli"
	ownerKindVendor = "vendor"

	surfaceProbeRunsMetric          = "gen_ai.provider.surface.probe.active.operation.runs.total"
	surfaceProbeLastRunMetric       = "gen_ai.provider.surface.probe.active.operation.last.run.timestamp.seconds"
	surfaceProbeLastOutcomeMetric   = "gen_ai.provider.surface.probe.active.operation.last.outcome"
	surfaceProbeLastReasonMetric    = "gen_ai.provider.surface.probe.active.operation.last.reason"
	surfaceProbeNextAllowedMetric   = "gen_ai.provider.surface.probe.active.operation.next.allowed.timestamp.seconds"
	surfaceAuthUsableMetric         = "gen_ai.provider.surface.probe.active.operation.auth.usable"
	surfaceCredentialLastUsedMetric = "gen_ai.provider.surface.probe.credential.last.used.timestamp.seconds"
	refreshReadyMetric              = "gen_ai.provider.cli.oauth.refresh.ready"
	refreshAttemptsMetric           = "gen_ai.provider.cli.oauth.refresh.attempts.total"
	runtimeRequestsMetric           = "gen_ai.provider.runtime.requests.total"
	runtimeRateLimitMetric          = "gen_ai.provider.runtime.rate_limit.events.total"
	runtimeLastSeenMetric           = "gen_ai.provider.runtime.last_seen.timestamp.seconds"
)

type providerLister interface {
	ListProviders(context.Context) ([]*managementv1.ProviderView, error)
}

type supportResourceLister interface {
	ListCLIs(context.Context) ([]*supportv1.CLI, error)
	ListVendors(context.Context) ([]*supportv1.Vendor, error)
}

type ObservabilityService struct {
	providers  providerLister
	support    supportResourceLister
	prom       promQueryExecutor
	prober     providerObservabilityProber
	metricRepo metricRepo
}

type ObservabilityServiceConfig struct {
	Providers  providerLister
	Support    supportResourceLister
	Prometheus promQueryExecutor
	Prober     providerObservabilityProber
	MetricRepo metricRepo
}

type cliSubject struct {
	owner                    string
	ownerID                  string
	cliID                    string
	vendorID                 string
	matcherLabel             string
	probeRunsMetric          string
	probeLastRunMetric       string
	probeLastReasonMetric    string
	probeNextAllowMetric     string
	authUsableMetric         string
	credentialLastUsedMetric string
	displayName              string
	iconURL                  string
	providerIDs              map[string]struct{}
	surfaceIDs               map[string]struct{}
	activeProbeIDs           map[string]struct{}
	metricNames              map[string]struct{}
	metricDescriptors        map[string]runtimeMetricDescriptor
	supportsQuotaQuery       bool
}

type providerSurfaceOwner struct {
	kind string
	id   string
}

type runtimeMetricDescriptor struct {
	name        string
	displayName string
	unit        string
	kind        observabilityv1.ObservabilityMetricKind
	category    observabilityv1.ObservabilityMetricCategory
	quotaQuery  bool
}

func NewObservabilityService(config ObservabilityServiceConfig) (*ObservabilityService, error) {
	switch {
	case config.Providers == nil:
		return nil, fmt.Errorf("consoleapi/providers: observability provider lister is nil")
	case config.Support == nil:
		return nil, fmt.Errorf("consoleapi/providers: observability support resource lister is nil")
	case config.Prometheus == nil:
		return nil, fmt.Errorf("consoleapi/providers: observability prometheus query client is nil")
	}
	resolvedMetricRepo := config.MetricRepo
	if resolvedMetricRepo == nil {
		resolvedMetricRepo = newMetricRepo()
	}
	return &ObservabilityService{
		providers:  config.Providers,
		support:    config.Support,
		prom:       config.Prometheus,
		prober:     config.Prober,
		metricRepo: resolvedMetricRepo,
	}, nil
}

func (s *ObservabilityService) Summary(ctx context.Context, window string) (*ProviderObservabilitySummaryResponse, error) {
	subjects, err := s.buildSubjects(ctx, "")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	items := make([]ProviderCLIObservabilityCard, 0, len(subjects))
	for _, subject := range subjects {
		card, buildErr := s.buildSummaryCard(ctx, subject, window)
		if buildErr != nil {
			items = append(items, buildSummaryCardBase(subject))
			continue
		}
		items = append(items, card)
	}
	return &ProviderObservabilitySummaryResponse{
		Window:      window,
		GeneratedAt: now.Format(time.RFC3339),
		Items:       items,
	}, nil
}

func (s *ObservabilityService) Provider(
	ctx context.Context,
	providerID string,
	window string,
	view providerObservabilityView,
) (*ProviderObservabilityResponse, error) {
	subjects, err := s.buildSubjects(ctx, strings.TrimSpace(providerID))
	if err != nil {
		return nil, err
	}
	items := make([]ProviderCLIObservabilityItem, 0, len(subjects))
	for _, subject := range subjects {
		item, buildErr := s.buildProviderItem(ctx, subject, window, view)
		if buildErr != nil {
			items = append(items, buildProviderItemBase(subject))
			continue
		}
		items = append(items, item)
	}
	return &ProviderObservabilityResponse{
		ProviderID:  providerID,
		Window:      window,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Items:       items,
	}, nil
}

func (s *ObservabilityService) buildSubjects(ctx context.Context, providerID string) ([]*cliSubject, error) {
	providers, err := s.providers.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	cliItems, err := s.support.ListCLIs(ctx)
	if err != nil {
		return nil, err
	}
	vendorItems, err := s.support.ListVendors(ctx)
	if err != nil {
		return nil, err
	}
	cliByID := make(map[string]*supportv1.CLI, len(cliItems))
	for _, cli := range cliItems {
		if cli == nil {
			continue
		}
		cliID := strings.TrimSpace(cli.GetCliId())
		if cliID != "" {
			cliByID[cliID] = cli
		}
	}
	vendorByID := make(map[string]*supportv1.Vendor, len(vendorItems))
	surfaceOwnerByID := make(map[string]string)
	for _, vendor := range vendorItems {
		if vendor == nil || vendor.GetVendor() == nil {
			continue
		}
		vendorID := strings.TrimSpace(vendor.GetVendor().GetVendorId())
		if vendorID != "" {
			vendorByID[vendorID] = vendor
		}
		for _, surface := range vendor.GetSurfaces() {
			surfaceID := strings.TrimSpace(surface.GetSurfaceId())
			if surfaceID != "" && vendorID != "" {
				surfaceOwnerByID[surfaceID] = vendorID
			}
		}
	}
	subjectsByID := map[string]*cliSubject{}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		currentProviderID := strings.TrimSpace(provider.GetProviderId())
		if providerID != "" && currentProviderID != providerID {
			continue
		}
		instanceID := strings.TrimSpace(provider.GetSurfaceId())
		if instanceID == "" {
			continue
		}
		owner := providerSurfaceOwnerFromProvider(provider, surfaceOwnerByID)
		if owner.id == "" {
			continue
		}
		switch owner.kind {
		case ownerKindCLI:
			key := owner.kind + ":" + owner.id
			subject, ok := subjectsByID[key]
			if !ok {
				subject = buildCLISubject(owner.id, cliByID[owner.id])
				subjectsByID[key] = subject
			}
			addSubjectProvider(subject, provider)
		case ownerKindVendor:
			key := owner.kind + ":" + owner.id
			subject, ok := subjectsByID[key]
			if !ok {
				subject = buildVendorSubject(owner.id, vendorByID[owner.id])
				subjectsByID[key] = subject
			}
			addSubjectProvider(subject, provider)
		}
	}
	subjects := make([]*cliSubject, 0, len(subjectsByID))
	for _, subject := range subjectsByID {
		subjects = append(subjects, subject)
	}
	slices.SortFunc(subjects, func(left, right *cliSubject) int {
		if cmp := strings.Compare(left.displayName, right.displayName); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(left.owner, right.owner); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.ownerID, right.ownerID)
	})
	return subjects, nil
}

func providerSurfaceOwnerFromProvider(provider *managementv1.ProviderView, surfaceOwnerByID map[string]string) providerSurfaceOwner {
	if provider == nil {
		return providerSurfaceOwner{}
	}
	for _, endpoint := range provider.GetEndpoints() {
		if cliID := providerv1.EndpointCLIID(endpoint); cliID != "" {
			return providerSurfaceOwner{kind: ownerKindCLI, id: cliID}
		}
	}
	if vendorID := strings.TrimSpace(surfaceOwnerByID[strings.TrimSpace(provider.GetSurfaceId())]); vendorID != "" {
		return providerSurfaceOwner{kind: ownerKindVendor, id: vendorID}
	}
	return providerSurfaceOwner{}
}

func addSubjectProvider(subject *cliSubject, provider *managementv1.ProviderView) {
	if subject == nil || provider == nil {
		return
	}
	if providerID := strings.TrimSpace(provider.GetProviderId()); providerID != "" {
		subject.providerIDs[providerID] = struct{}{}
	}
	if surfaceID := strings.TrimSpace(provider.GetSurfaceId()); surfaceID != "" {
		subject.surfaceIDs[surfaceID] = struct{}{}
	}
}

func buildCLISubject(cliID string, cli *supportv1.CLI) *cliSubject {
	subject := &cliSubject{
		owner:                    ownerKindCLI,
		ownerID:                  strings.TrimSpace(cliID),
		cliID:                    strings.TrimSpace(cliID),
		matcherLabel:             "cli_id",
		probeRunsMetric:          surfaceProbeRunsMetric,
		probeLastRunMetric:       surfaceProbeLastRunMetric,
		probeLastReasonMetric:    surfaceProbeLastReasonMetric,
		probeNextAllowMetric:     surfaceProbeNextAllowedMetric,
		authUsableMetric:         surfaceAuthUsableMetric,
		credentialLastUsedMetric: surfaceCredentialLastUsedMetric,
		displayName:              strings.TrimSpace(cliID),
		providerIDs:              map[string]struct{}{},
		surfaceIDs:               map[string]struct{}{},
		activeProbeIDs:           map[string]struct{}{},
		metricNames:              map[string]struct{}{},
		metricDescriptors:        map[string]runtimeMetricDescriptor{},
	}
	if cli == nil || cli.GetOauth() == nil || cli.GetOauth().GetObservability() == nil {
		return subject
	}
	subject.displayName = strings.TrimSpace(cli.GetDisplayName())
	if subject.displayName == "" {
		subject.displayName = strings.TrimSpace(cliID)
	}
	subject.iconURL = strings.TrimSpace(cli.GetIconUrl())
	applyObservabilityCapability(subject, cli.GetOauth().GetObservability(), subject.ownerID)
	return subject
}

func buildVendorSubject(vendorID string, vendor *supportv1.Vendor) *cliSubject {
	subject := &cliSubject{
		owner:                    ownerKindVendor,
		ownerID:                  strings.TrimSpace(vendorID),
		vendorID:                 strings.TrimSpace(vendorID),
		matcherLabel:             "vendor_id",
		probeRunsMetric:          surfaceProbeRunsMetric,
		probeLastRunMetric:       surfaceProbeLastRunMetric,
		probeLastReasonMetric:    surfaceProbeLastReasonMetric,
		probeNextAllowMetric:     surfaceProbeNextAllowedMetric,
		authUsableMetric:         surfaceAuthUsableMetric,
		credentialLastUsedMetric: surfaceCredentialLastUsedMetric,
		displayName:              strings.TrimSpace(vendorID),
		providerIDs:              map[string]struct{}{},
		surfaceIDs:               map[string]struct{}{},
		activeProbeIDs:           map[string]struct{}{},
		metricNames:              map[string]struct{}{},
		metricDescriptors:        map[string]runtimeMetricDescriptor{},
	}
	if vendor == nil {
		return subject
	}
	if vendor.GetVendor() != nil {
		subject.displayName = strings.TrimSpace(vendor.GetVendor().GetDisplayName())
		if subject.displayName == "" {
			subject.displayName = strings.TrimSpace(vendorID)
		}
		subject.iconURL = strings.TrimSpace(vendor.GetVendor().GetIconUrl())
	}
	for _, surface := range vendor.GetSurfaces() {
		applyObservabilityCapability(subject, surface.GetObservability(), strings.TrimSpace(surface.GetQuotaProbeId()))
	}
	return subject
}

func applyObservabilityCapability(subject *cliSubject, capability *observabilityv1.ObservabilityCapability, defaultProbeID string) {
	if subject == nil || capability == nil {
		return
	}
	for _, profile := range capability.GetProfiles() {
		if profile == nil {
			continue
		}
		quotaQuery := profile.GetQuotaQuery()
		quotaQueryProfile := quotaQuery != nil
		if quotaQueryProfile {
			subject.supportsQuotaQuery = true
			probeID := strings.TrimSpace(quotaQuery.GetCollectorId())
			if probeID == "" {
				probeID = strings.TrimSpace(defaultProbeID)
			}
			if probeID == "" {
				probeID = strings.TrimSpace(subject.ownerID)
			}
			if probeID != "" {
				subject.activeProbeIDs[probeID] = struct{}{}
			}
		}
		for _, metric := range profile.GetMetrics() {
			if metric == nil {
				continue
			}
			name := strings.TrimSpace(metric.GetName())
			if name == "" {
				continue
			}
			subject.metricNames[name] = struct{}{}
			subject.metricDescriptors[name] = runtimeMetricDescriptor{
				name:        name,
				displayName: formatRuntimeMetricDisplayName(name),
				unit:        strings.TrimSpace(metric.GetUnit()),
				kind:        metric.GetKind(),
				category:    metric.GetCategory(),
				quotaQuery:  quotaQueryProfile,
			}
		}
	}
}

func hasMetric(subject *cliSubject, metric string) bool {
	if subject == nil {
		return false
	}
	_, ok := subject.metricNames[metric]
	return ok
}

func runtimeGaugeMetricDescriptors(subject *cliSubject) []runtimeMetricDescriptor {
	if subject == nil || len(subject.metricDescriptors) == 0 {
		return nil
	}
	descriptors := make([]runtimeMetricDescriptor, 0, len(subject.metricDescriptors))
	for _, descriptor := range subject.metricDescriptors {
		if descriptor.kind != observabilityv1.ObservabilityMetricKind_OBSERVABILITY_METRIC_KIND_GAUGE {
			continue
		}
		if descriptor.category != observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_QUOTA &&
			descriptor.category != observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_RATE_LIMIT &&
			descriptor.category != observabilityv1.ObservabilityMetricCategory_OBSERVABILITY_METRIC_CATEGORY_USAGE {
			continue
		}
		if isBuiltinRuntimeMetric(descriptor.name) {
			continue
		}
		descriptors = append(descriptors, descriptor)
	}
	slices.SortFunc(descriptors, func(left, right runtimeMetricDescriptor) int {
		return strings.Compare(left.name, right.name)
	})
	return descriptors
}

func isBuiltinRuntimeMetric(name string) bool {
	switch strings.TrimSpace(name) {
	case refreshReadyMetric,
		runtimeLastSeenMetric:
		return true
	default:
		return false
	}
}

func formatRuntimeMetricDisplayName(metricName string) string {
	trimmed := strings.TrimSpace(metricName)
	if trimmed == "" {
		return ""
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '.' || r == '_'
	})
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		words = append(words, strings.ToUpper(part[:1])+part[1:])
	}
	return strings.Join(words, " ")
}

func providerRegex(subject *cliSubject) string {
	if subject == nil || len(subject.providerIDs) == 0 {
		return ""
	}
	values := make([]string, 0, len(subject.providerIDs))
	for providerID := range subject.providerIDs {
		values = append(values, regexp.QuoteMeta(providerID))
	}
	slices.Sort(values)
	return strings.Join(values, "|")
}

func activeProbeRegex(subject *cliSubject) string {
	if subject == nil || len(subject.activeProbeIDs) == 0 {
		return ""
	}
	values := make([]string, 0, len(subject.activeProbeIDs))
	for probeID := range subject.activeProbeIDs {
		values = append(values, regexp.QuoteMeta(probeID))
	}
	slices.Sort(values)
	return strings.Join(values, "|")
}

func (s *ObservabilityService) queryLabelValues(ctx context.Context, query string, label string) ([]ProviderLabelValue, error) {
	samples, err := s.prom.QueryVector(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: query label values: %w", err)
	}
	items := make([]ProviderLabelValue, 0, len(samples))
	for _, sample := range samples {
		items = append(items, ProviderLabelValue{
			Label: strings.TrimSpace(sample.Metric[label]),
			Value: sample.Value,
		})
	}
	slices.SortFunc(items, func(left, right ProviderLabelValue) int {
		return strings.Compare(left.Label, right.Label)
	})
	return items, nil
}

func (s *ObservabilityService) queryInstanceTimestamps(ctx context.Context, query string) ([]SurfaceTimestamp, error) {
	samples, err := s.prom.QueryVector(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: query instance timestamps: %w", err)
	}
	items := make([]SurfaceTimestamp, 0, len(samples))
	for _, sample := range samples {
		items = append(items, SurfaceTimestamp{
			SurfaceID: metricSurfaceIdentity(sample.Metric),
			Timestamp: formatPromTimestamp(sample.Value),
		})
	}
	slices.SortFunc(items, func(left, right SurfaceTimestamp) int {
		return strings.Compare(left.SurfaceID, right.SurfaceID)
	})
	return items, nil
}

func (s *ObservabilityService) queryRangeLabelSeries(
	ctx context.Context,
	query string,
	label string,
	window string,
) ([]ProviderMetricSeries, error) {
	windowDuration, err := parseWindowDuration(window)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: parse window duration: %w", err)
	}
	end := time.Now().UTC()
	start := end.Add(-windowDuration)
	step := rangeStep(windowDuration)
	samples, err := s.prom.QueryRange(ctx, query, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: query label series range: %w", err)
	}
	items := make([]ProviderMetricSeries, 0, len(samples))
	for _, sample := range samples {
		points := make([]ProviderTimeSeriesPoint, 0, len(sample.Values))
		for _, point := range sample.Values {
			points = append(points, ProviderTimeSeriesPoint{
				Timestamp: point.Timestamp.UTC().Format(time.RFC3339),
				Value:     point.Value,
			})
		}
		items = append(items, ProviderMetricSeries{
			Label:  strings.TrimSpace(sample.Metric[label]),
			Points: points,
		})
	}
	slices.SortFunc(items, func(left, right ProviderMetricSeries) int {
		return strings.Compare(left.Label, right.Label)
	})
	return items, nil
}

func formatPromTimestamp(value float64) string {
	if value <= 0 {
		return ""
	}
	seconds := int64(value)
	return time.Unix(seconds, 0).UTC().Format(time.RFC3339)
}

func durationRange(window string) string {
	return "[" + window + "]"
}

func promActiveDiscoveryMatcher(subject *cliSubject) string {
	if subject == nil {
		return ""
	}
	parts := []string{}
	if providerPattern := providerRegex(subject); providerPattern != "" {
		parts = append(parts, fmt.Sprintf(`provider_id=~%s`, strconv.Quote(providerPattern)))
	}
	if probePattern := activeProbeRegex(subject); probePattern != "" {
		parts = append(parts, fmt.Sprintf(`schema_id=~%s`, strconv.Quote(probePattern)))
	}
	return strings.Join(parts, ",")
}

func promOwnerMatcher(subject *cliSubject) string {
	if subject == nil {
		return ""
	}
	matcherLabel := strings.TrimSpace(subject.matcherLabel)
	if matcherLabel == "" {
		matcherLabel = "cli_id"
	}
	ownerID := strings.TrimSpace(subject.ownerID)
	if ownerID == "" {
		ownerID = strings.TrimSpace(subject.cliID)
	}
	parts := []string{
		fmt.Sprintf(`%s=%s`, matcherLabel, strconv.Quote(ownerID)),
	}
	if providerPattern := providerRegex(subject); providerPattern != "" {
		parts = append(parts, fmt.Sprintf(`provider_id=~%s`, strconv.Quote(providerPattern)))
	}
	return strings.Join(parts, ",")
}

func promRuntimeMatcher(subject *cliSubject) string {
	return promOwnerMatcher(subject)
}

func parseWindowDuration(window string) (time.Duration, error) {
	switch strings.TrimSpace(strings.ToLower(window)) {
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported window %q", window)
	}
}

func rangeStep(window time.Duration) time.Duration {
	switch {
	case window <= 15*time.Minute:
		return 30 * time.Second
	case window <= time.Hour:
		return time.Minute
	case window <= 6*time.Hour:
		return 5 * time.Minute
	default:
		return 10 * time.Minute
	}
}
