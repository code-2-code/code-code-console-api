package providers

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func (s *ObservabilityService) buildSummaryCard(
	ctx context.Context,
	subject *cliSubject,
	window string,
) (ProviderCLIObservabilityCard, error) {
	card := buildSummaryCardBase(subject)
	if card.InstanceCount == 0 {
		return card, nil
	}
	activeMatcher := promActiveDiscoveryMatcher(subject)
	runtimeMatcher := promRuntimeMatcher(subject)
	if subject.supportsActiveQuery {
		outcomes, err := s.queryLabelValues(
			ctx,
			fmt.Sprintf(
				`sum by (outcome) (increase(%s{%s}%s))`,
				s.metricRepo.StorageName(subject.probeRunsMetric),
				activeMatcher,
				durationRange(window),
			),
			"outcome",
		)
		if err != nil {
			return ProviderCLIObservabilityCard{}, err
		}
		if len(outcomes) > 0 {
			card.Probe = buildProbeSummary(outcomes)
		}
	}
	if hasMetric(subject, refreshReadyMetric) {
		ready, err := s.queryLabelValues(
			ctx,
			fmt.Sprintf(`sum(%s{%s})`, s.metricRepo.StorageName(refreshReadyMetric), runtimeMatcher),
			"__name__",
		)
		if err != nil {
			return ProviderCLIObservabilityCard{}, err
		}
		total, err := s.queryLabelValues(
			ctx,
			fmt.Sprintf(`count(%s{%s})`, s.metricRepo.StorageName(refreshReadyMetric), runtimeMatcher),
			"__name__",
		)
		if err != nil {
			return ProviderCLIObservabilityCard{}, err
		}
		if len(total) > 0 {
			card.Refresh = &ProviderReadinessSummary{
				Ready: firstValue(ready),
				Total: firstValue(total),
			}
		}
	}
	if hasMetric(subject, runtimeRequestsMetric) {
		statuses, err := s.queryLabelValues(
			ctx,
			fmt.Sprintf(
				`sum by (status_class) (increase(%s{%s}%s))`,
				s.metricRepo.StorageName(runtimeRequestsMetric),
				runtimeMatcher,
				durationRange(window),
			),
			"status_class",
		)
		if err != nil {
			return ProviderCLIObservabilityCard{}, err
		}
		if len(statuses) > 0 {
			card.Runtime = buildRuntimeSummary(statuses)
		}
	}
	return card, nil
}

func (s *ObservabilityService) buildProviderItem(
	ctx context.Context,
	subject *cliSubject,
	window string,
	view providerObservabilityView,
) (ProviderCLIObservabilityItem, error) {
	item := buildProviderItemBase(subject)
	if len(item.ProviderSurfaceBindingIDs) == 0 {
		return item, nil
	}
	activeMatcher := promActiveDiscoveryMatcher(subject)
	runtimeMatcher := promRuntimeMatcher(subject)
	var err error
	if subject.supportsActiveQuery && view.includesFullDetail() {
		item.ProbeOutcomes, err = s.queryLabelValues(
			ctx,
			fmt.Sprintf(
				`sum by (outcome) (increase(%s{%s}%s))`,
				s.metricRepo.StorageName(subject.probeRunsMetric),
				activeMatcher,
				durationRange(window),
			),
			"outcome",
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
		item.ProbeOutcomeSeries, err = s.queryRangeLabelSeries(
			ctx,
			fmt.Sprintf(`sum by (outcome) (rate(%s{%s}[5m]))`, s.metricRepo.StorageName(subject.probeRunsMetric), activeMatcher),
			"outcome",
			window,
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
	}
	if subject.supportsActiveQuery && view.includesObservedAt() {
		item.LastProbeRun, item.LastProbeOutcome, item.LastProbeReason, item.NextProbeAllowed, err = s.queryLatestActiveDiscoveryStatus(ctx, subject, activeMatcher, window)
	}
	if subject.supportsActiveQuery && view.includesStatus() {
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
		item.AuthUsable, item.CredentialLastUsed, err = s.queryLatestCredentialHealth(ctx, subject, activeMatcher, window)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
	}
	if hasMetric(subject, refreshAttemptsMetric) && view.includesFullDetail() {
		item.RefreshAttempts, err = s.queryLabelValues(
			ctx,
			fmt.Sprintf(
				`sum by (result) (increase(%s{%s}%s))`,
				s.metricRepo.StorageName(refreshAttemptsMetric),
				runtimeMatcher,
				durationRange(window),
			),
			"result",
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
		item.RefreshAttemptSeries, err = s.queryRangeLabelSeries(
			ctx,
			fmt.Sprintf(`sum by (result) (rate(%s{%s}[5m]))`, s.metricRepo.StorageName(refreshAttemptsMetric), runtimeMatcher),
			"result",
			window,
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
	}
	if hasMetric(subject, refreshReadyMetric) && view.includesFullDetail() {
		item.RefreshReady, err = s.queryInstanceReadiness(
			ctx,
			fmt.Sprintf(`max(%s{%s})`, s.metricRepo.StorageName(refreshReadyMetric), runtimeMatcher),
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
	}
	if hasMetric(subject, runtimeRequestsMetric) && view.includesFullDetail() {
		item.RuntimeRequests, err = s.queryLabelValues(
			ctx,
			fmt.Sprintf(
				`sum by (status_class) (increase(%s{%s}%s))`,
				s.metricRepo.StorageName(runtimeRequestsMetric),
				runtimeMatcher,
				durationRange(window),
			),
			"status_class",
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
		item.RuntimeRequestSeries, err = s.queryRangeLabelSeries(
			ctx,
			fmt.Sprintf(`sum by (status_class) (rate(%s{%s}[5m]))`, s.metricRepo.StorageName(runtimeRequestsMetric), runtimeMatcher),
			"status_class",
			window,
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
	}
	if hasMetric(subject, runtimeRateLimitMetric) && view.includesFullDetail() {
		item.RuntimeRateLimits, err = s.queryLabelValues(
			ctx,
			fmt.Sprintf(
				`sum by (limit_kind) (increase(%s{%s}%s))`,
				s.metricRepo.StorageName(runtimeRateLimitMetric),
				runtimeMatcher,
				durationRange(window),
			),
			"limit_kind",
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
		item.RuntimeRateLimitSeries, err = s.queryRangeLabelSeries(
			ctx,
			fmt.Sprintf(`sum by (limit_kind) (rate(%s{%s}[5m]))`, s.metricRepo.StorageName(runtimeRateLimitMetric), runtimeMatcher),
			"limit_kind",
			window,
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
	}
	if view.includesCard() {
		descriptors := runtimeGaugeMetricDescriptors(subject)
		metricRows, rowsErr := s.queryRuntimeGaugeMetrics(ctx, descriptors, runtimeMatcher, activeMatcher, window, view)
		if rowsErr != nil {
			return ProviderCLIObservabilityItem{}, rowsErr
		}
		for _, descriptor := range descriptors {
			rows := metricRows[descriptor.name]
			if len(rows) == 0 {
				continue
			}
			item.RuntimeMetrics = append(item.RuntimeMetrics, ProviderRuntimeMetricRows{
				MetricName:  canonicalObservabilityMetricName(descriptor.name),
				DisplayName: descriptor.displayName,
				Unit:        descriptor.unit,
				Category:    descriptor.category.String(),
				Rows:        rows,
			})
		}
	}
	if hasMetric(subject, runtimeLastSeenMetric) && view.includesFullDetail() {
		item.LastRuntimeSeen, err = s.queryInstanceTimestamps(
			ctx,
			fmt.Sprintf(
				`max(last_over_time(%s{%s}%s))`,
				s.metricRepo.StorageName(runtimeLastSeenMetric),
				runtimeMatcher,
				durationRange(window),
			),
		)
		if err != nil {
			return ProviderCLIObservabilityItem{}, err
		}
	}
	return item, nil
}

func (s *ObservabilityService) queryLatestCredentialHealth(
	ctx context.Context,
	subject *cliSubject,
	matcher string,
	window string,
) ([]ProviderSurfaceBindingValue, []ProviderSurfaceBindingTimestamp, error) {
	if subject == nil || strings.TrimSpace(subject.authUsableMetric) == "" || strings.TrimSpace(subject.credentialLastUsedMetric) == "" {
		return nil, nil, nil
	}
	windowRange := durationRange(window)
	authUsable, err := s.queryInstanceValues(
		ctx,
		fmt.Sprintf(
			`last_over_time(%s{%s}%s)`,
			s.metricRepo.StorageName(subject.authUsableMetric),
			matcher,
			windowRange,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("consoleapi/providers: query latest auth usable: %w", err)
	}
	credentialLastUsed, err := s.queryInstanceTimestamps(
		ctx,
		fmt.Sprintf(
			`last_over_time(%s{%s}%s)`,
			s.metricRepo.StorageName(subject.credentialLastUsedMetric),
			matcher,
			windowRange,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("consoleapi/providers: query latest credential last used: %w", err)
	}
	return authUsable, credentialLastUsed, nil
}

func (s *ObservabilityService) queryRuntimeGaugeMetrics(
	ctx context.Context,
	descriptors []runtimeMetricDescriptor,
	runtimeMatcher string,
	activeMatcher string,
	window string,
	view providerObservabilityView,
) (map[string][]ProviderMetricRow, error) {
	rowsByMetric := map[string][]ProviderMetricRow{}
	if len(descriptors) == 0 {
		return rowsByMetric, nil
	}
	runtimeDescriptors, activeDescriptors := splitRuntimeGaugeDescriptors(descriptors)
	for _, group := range []struct {
		descriptors []runtimeMetricDescriptor
		matcher     string
	}{
		{descriptors: runtimeDescriptors, matcher: runtimeMatcher},
		{descriptors: activeDescriptors, matcher: activeMatcher},
	} {
		if len(group.descriptors) == 0 {
			continue
		}
		groupRows, err := s.queryRuntimeGaugeMetricGroup(ctx, group.descriptors, group.matcher, window, view)
		if err != nil {
			return nil, err
		}
		for metricName, rows := range groupRows {
			rowsByMetric[metricName] = rows
		}
	}
	return rowsByMetric, nil
}

func splitRuntimeGaugeDescriptors(descriptors []runtimeMetricDescriptor) ([]runtimeMetricDescriptor, []runtimeMetricDescriptor) {
	runtimeDescriptors := make([]runtimeMetricDescriptor, 0, len(descriptors))
	activeDescriptors := make([]runtimeMetricDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.activeQuery {
			activeDescriptors = append(activeDescriptors, descriptor)
			continue
		}
		runtimeDescriptors = append(runtimeDescriptors, descriptor)
	}
	return runtimeDescriptors, activeDescriptors
}

func (s *ObservabilityService) queryRuntimeGaugeMetricGroup(
	ctx context.Context,
	descriptors []runtimeMetricDescriptor,
	matcher string,
	window string,
	view providerObservabilityView,
) (map[string][]ProviderMetricRow, error) {
	if len(descriptors) == 1 {
		descriptor := descriptors[0]
		query := s.metricRepo.LatestGaugeQuery(descriptor.name, matcher)
		if view == providerObservabilityViewCard {
			query = s.metricRepo.LatestGaugeRangeQuery(descriptor.name, matcher, window)
		}
		rows, err := s.queryMetricRows(ctx, query)
		if err != nil {
			return nil, err
		}
		return map[string][]ProviderMetricRow{descriptor.name: rows}, nil
	}
	query := runtimeGaugeMetricGroupQuery(s.metricRepo, descriptors, matcher, window, view)
	samples, err := s.prom.QueryVector(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: query runtime gauge metric group: %w", err)
	}
	return metricRowsByName(samples), nil
}

func runtimeGaugeMetricGroupQuery(
	metricRepo metricRepo,
	descriptors []runtimeMetricDescriptor,
	matcher string,
	window string,
	view providerObservabilityView,
) string {
	selector := runtimeGaugeMetricSelector(metricRepo, descriptors, matcher)
	if view == providerObservabilityViewCard {
		return fmt.Sprintf(`last_over_time(%s[%s])`, selector, strings.TrimSpace(window))
	}
	return selector
}

func runtimeGaugeMetricSelector(metricRepo metricRepo, descriptors []runtimeMetricDescriptor, matcher string) string {
	storageNames := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		storageName := strings.TrimSpace(metricRepo.StorageName(descriptor.name))
		if storageName == "" {
			continue
		}
		storageNames = append(storageNames, regexp.QuoteMeta(storageName))
	}
	slices.Sort(storageNames)
	parts := []string{fmt.Sprintf(`__name__=~%s`, strconv.Quote(strings.Join(storageNames, "|")))}
	if trimmedMatcher := strings.TrimSpace(matcher); trimmedMatcher != "" {
		parts = append(parts, trimmedMatcher)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func metricRowsByName(samples []promVectorSample) map[string][]ProviderMetricRow {
	bestRows := map[string]map[string]ProviderMetricRow{}
	for _, sample := range samples {
		metricName := canonicalObservabilityMetricName(sample.Metric["__name__"])
		if strings.TrimSpace(metricName) == "" {
			continue
		}
		labels := copyMetricLabels(sample.Metric)
		rowKey := metricRowSortKey(ProviderMetricRow{Labels: labels})
		rows := bestRows[metricName]
		if rows == nil {
			rows = map[string]ProviderMetricRow{}
			bestRows[metricName] = rows
		}
		if existing, ok := rows[rowKey]; ok && existing.Value >= sample.Value {
			continue
		}
		rows[rowKey] = ProviderMetricRow{
			Labels: labels,
			Value:  sample.Value,
		}
	}
	result := make(map[string][]ProviderMetricRow, len(bestRows))
	for metricName, rows := range bestRows {
		values := make([]ProviderMetricRow, 0, len(rows))
		for _, row := range rows {
			values = append(values, row)
		}
		slices.SortFunc(values, func(left, right ProviderMetricRow) int {
			return strings.Compare(metricRowSortKey(left), metricRowSortKey(right))
		})
		result[metricName] = values
	}
	return result
}

func buildSummaryCardBase(subject *cliSubject) ProviderCLIObservabilityCard {
	if subject == nil {
		return ProviderCLIObservabilityCard{}
	}
	return ProviderCLIObservabilityCard{
		Owner:         subject.owner,
		CLIID:         subject.cliID,
		VendorID:      subject.vendorID,
		DisplayName:   subject.displayName,
		IconURL:       subject.iconURL,
		ProviderCount: len(subject.providerIDs),
		InstanceCount: len(subject.providerSurfaceBindingIDs),
	}
}

func buildProviderItemBase(subject *cliSubject) ProviderCLIObservabilityItem {
	if subject == nil {
		return ProviderCLIObservabilityItem{}
	}
	return ProviderCLIObservabilityItem{
		Owner:                     subject.owner,
		CLIID:                     subject.cliID,
		VendorID:                  subject.vendorID,
		DisplayName:               subject.displayName,
		IconURL:                   subject.iconURL,
		ProviderSurfaceBindingIDs: subjectInstanceIDs(subject),
	}
}

func subjectInstanceIDs(subject *cliSubject) []string {
	values := make([]string, 0, len(subject.providerSurfaceBindingIDs))
	for instanceID := range subject.providerSurfaceBindingIDs {
		values = append(values, instanceID)
	}
	slices.Sort(values)
	return values
}

func buildProbeSummary(values []ProviderLabelValue) *ProviderProbeOutcomeSummary {
	summary := &ProviderProbeOutcomeSummary{}
	for _, item := range values {
		summary.Total += item.Value
		switch item.Label {
		case "executed":
			summary.Executed = item.Value
		case "throttled":
			summary.Throttled = item.Value
		case "auth_blocked":
			summary.AuthBlocked = item.Value
		case "unsupported":
			summary.Unsupported = item.Value
		case "failed":
			summary.Failed = item.Value
		}
	}
	return summary
}

func buildRuntimeSummary(values []ProviderLabelValue) *ProviderRuntimeRequestStatusSummary {
	summary := &ProviderRuntimeRequestStatusSummary{}
	for _, item := range values {
		summary.Total += item.Value
		switch item.Label {
		case "2xx":
			summary.Status2xx = item.Value
		case "3xx":
			summary.Status3xx = item.Value
		case "4xx":
			summary.Status4xx = item.Value
		case "5xx":
			summary.Status5xx = item.Value
		}
	}
	return summary
}

func firstValue(values []ProviderLabelValue) float64 {
	if len(values) == 0 {
		return 0
	}
	return values[0].Value
}

func (s *ObservabilityService) queryLatestActiveDiscoveryStatus(
	ctx context.Context,
	subject *cliSubject,
	activeMatcher string,
	window string,
) ([]ProviderSurfaceBindingTimestamp, []ProviderSurfaceBindingValue, []ProviderSurfaceBindingReason, []ProviderSurfaceBindingTimestamp, error) {
	windowRange := durationRange(window)
	lastRunSamples, err := s.prom.QueryVector(ctx, fmt.Sprintf(
		`last_over_time(%s{%s}%s)`,
		s.metricRepo.StorageName(subject.probeLastRunMetric),
		activeMatcher,
		windowRange,
	))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("consoleapi/providers: query latest active probe runs: %w", err)
	}
	outcomeSamples, err := s.prom.QueryVector(ctx, fmt.Sprintf(
		`last_over_time(%s{%s}%s)`,
		s.metricRepo.StorageName(probeLastOutcomeMetric(subject)),
		activeMatcher,
		windowRange,
	))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("consoleapi/providers: query latest active probe outcomes: %w", err)
	}
	nextSamples, err := s.prom.QueryVector(ctx, fmt.Sprintf(
		`last_over_time(%s{%s}%s)`,
		s.metricRepo.StorageName(subject.probeNextAllowMetric),
		activeMatcher,
		windowRange,
	))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("consoleapi/providers: query latest active probe next allowed: %w", err)
	}
	var reasonSamples []promVectorSample
	if strings.TrimSpace(subject.probeLastReasonMetric) != "" {
		reasonSamples, err = s.prom.QueryVector(ctx, fmt.Sprintf(
			`last_over_time(%s{%s}%s)`,
			s.metricRepo.StorageName(subject.probeLastReasonMetric),
			activeMatcher,
			windowRange,
		))
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("consoleapi/providers: query latest active probe reason: %w", err)
		}
	}

	selectedRuns := selectLatestProbeRunSamples(lastRunSamples)
	outcomesBySeries := vectorSamplesBySeriesKey(outcomeSamples)
	nextBySeries := vectorSamplesBySeriesKey(nextSamples)
	reasonsByGroup := reasonSamplesByActiveDiscoveryGroup(reasonSamples)
	runs := make([]ProviderSurfaceBindingTimestamp, 0, len(selectedRuns))
	outcomes := make([]ProviderSurfaceBindingValue, 0, len(selectedRuns))
	reasons := make([]ProviderSurfaceBindingReason, 0, len(selectedRuns))
	nextAllowed := make([]ProviderSurfaceBindingTimestamp, 0, len(selectedRuns))
	for _, sample := range selectedRuns {
		seriesKey := promSeriesKey(sample.Metric)
		groupKey := activeDiscoveryGroupKey(sample.Metric)
		providerSurfaceBindingID := strings.TrimSpace(sample.Metric["provider_surface_binding_id"])
		runs = append(runs, ProviderSurfaceBindingTimestamp{
			ProviderSurfaceBindingID: providerSurfaceBindingID,
			Timestamp:                formatPromTimestamp(sample.Value),
		})
		outcome, hasOutcome := outcomesBySeries[seriesKey]
		if hasOutcome {
			outcomes = append(outcomes, ProviderSurfaceBindingValue{
				ProviderSurfaceBindingID: providerSurfaceBindingID,
				Value:                    outcome.Value,
			})
		}
		if hasOutcome && activeProbeOutcomeHasReason(outcome.Value) {
			if reason, ok := reasonsByGroup[groupKey]; ok {
				reasons = append(reasons, ProviderSurfaceBindingReason{
					ProviderSurfaceBindingID: providerSurfaceBindingID,
					Reason:                   reason,
				})
			}
		}
		if next, ok := nextBySeries[seriesKey]; ok {
			nextAllowed = append(nextAllowed, ProviderSurfaceBindingTimestamp{
				ProviderSurfaceBindingID: providerSurfaceBindingID,
				Timestamp:                formatPromTimestamp(next.Value),
			})
		}
	}
	return runs, outcomes, reasons, nextAllowed, nil
}

func activeProbeOutcomeHasReason(value float64) bool {
	switch value {
	case 3, 5:
		return true
	default:
		return false
	}
}

func reasonSamplesByActiveDiscoveryGroup(samples []promVectorSample) map[string]string {
	items := make(map[string]string, len(samples))
	for _, sample := range samples {
		reason := strings.TrimSpace(sample.Metric["reason"])
		if reason == "" {
			continue
		}
		items[activeDiscoveryGroupKey(sample.Metric)] = reason
	}
	return items
}

func selectLatestProbeRunSamples(samples []promVectorSample) []promVectorSample {
	selectedByGroup := map[string]promVectorSample{}
	for _, sample := range samples {
		groupKey := activeDiscoveryGroupKey(sample.Metric)
		if existing, ok := selectedByGroup[groupKey]; ok && existing.Value >= sample.Value {
			continue
		}
		selectedByGroup[groupKey] = sample
	}
	items := make([]promVectorSample, 0, len(selectedByGroup))
	for _, sample := range selectedByGroup {
		items = append(items, sample)
	}
	slices.SortFunc(items, func(left, right promVectorSample) int {
		return strings.Compare(activeDiscoveryGroupKey(left.Metric), activeDiscoveryGroupKey(right.Metric))
	})
	return items
}

func activeDiscoveryGroupKey(metric map[string]string) string {
	if instanceID := strings.TrimSpace(metric["provider_surface_binding_id"]); instanceID != "" {
		return "instance:" + instanceID
	}
	if providerID := strings.TrimSpace(metric["provider_id"]); providerID != "" {
		return "provider:" + providerID
	}
	return promSeriesKey(metric)
}

func vectorSamplesBySeriesKey(samples []promVectorSample) map[string]promVectorSample {
	items := make(map[string]promVectorSample, len(samples))
	for _, sample := range samples {
		items[promSeriesKey(sample.Metric)] = sample
	}
	return items
}

func promSeriesKey(metric map[string]string) string {
	if len(metric) == 0 {
		return ""
	}
	keys := make([]string, 0, len(metric))
	for key := range metric {
		if key == "__name__" {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+metric[key])
	}
	return strings.Join(parts, ",")
}

func probeLastOutcomeMetric(subject *cliSubject) string {
	if subject == nil {
		return ""
	}
	if subject.owner == ownerKindVendor {
		return vendorProbeLastOutcomeMetric
	}
	return cliProbeLastOutcomeMetric
}

func (s *ObservabilityService) queryInstanceReadiness(ctx context.Context, query string) ([]ProviderSurfaceBindingReadiness, error) {
	samples, err := s.prom.QueryVector(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: query instance readiness: %w", err)
	}
	items := make([]ProviderSurfaceBindingReadiness, 0, len(samples))
	for _, sample := range samples {
		items = append(items, ProviderSurfaceBindingReadiness{
			ProviderSurfaceBindingID: strings.TrimSpace(sample.Metric["provider_surface_binding_id"]),
			Value:                    sample.Value,
		})
	}
	slices.SortFunc(items, func(left, right ProviderSurfaceBindingReadiness) int {
		return strings.Compare(left.ProviderSurfaceBindingID, right.ProviderSurfaceBindingID)
	})
	return items, nil
}

func (s *ObservabilityService) queryInstanceValues(ctx context.Context, query string) ([]ProviderSurfaceBindingValue, error) {
	samples, err := s.prom.QueryVector(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: query instance values: %w", err)
	}
	items := make([]ProviderSurfaceBindingValue, 0, len(samples))
	for _, sample := range samples {
		items = append(items, ProviderSurfaceBindingValue{
			ProviderSurfaceBindingID: strings.TrimSpace(sample.Metric["provider_surface_binding_id"]),
			Value:                    sample.Value,
		})
	}
	slices.SortFunc(items, func(left, right ProviderSurfaceBindingValue) int {
		return strings.Compare(left.ProviderSurfaceBindingID, right.ProviderSurfaceBindingID)
	})
	return items, nil
}

func (s *ObservabilityService) queryMetricRows(ctx context.Context, query string) ([]ProviderMetricRow, error) {
	samples, err := s.prom.QueryVector(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("consoleapi/providers: query metric rows: %w", err)
	}
	items := make([]ProviderMetricRow, 0, len(samples))
	for _, sample := range samples {
		items = append(items, ProviderMetricRow{
			Labels: copyMetricLabels(sample.Metric),
			Value:  sample.Value,
		})
	}
	slices.SortFunc(items, func(left, right ProviderMetricRow) int {
		return strings.Compare(metricRowSortKey(left), metricRowSortKey(right))
	})
	return items, nil
}

func copyMetricLabels(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	labels := make(map[string]string, len(source))
	for key, value := range source {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" ||
			trimmedKey == "__name__" ||
			trimmedKey == "provider_surface_binding_id" ||
			isRuntimeGaugeInfrastructureLabel(trimmedKey) {
			continue
		}
		labels[trimmedKey] = strings.TrimSpace(value)
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func isRuntimeGaugeInfrastructureLabel(label string) bool {
	switch strings.TrimSpace(label) {
	case "job", "instance", "pod", "namespace", "service", "endpoint", "container":
		return true
	default:
		return false
	}
}

func metricRowSortKey(row ProviderMetricRow) string {
	if len(row.Labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(row.Labels))
	for key := range row.Labels {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+row.Labels[key])
	}
	return strings.Join(parts, ",")
}
