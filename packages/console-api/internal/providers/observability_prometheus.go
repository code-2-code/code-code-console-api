package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type promVectorSample struct {
	Metric map[string]string
	Value  float64
}

type promRangeSample struct {
	Metric map[string]string
	Values []promRangePoint
}

type promRangePoint struct {
	Timestamp time.Time
	Value     float64
}

type promQueryExecutor interface {
	QueryVector(context.Context, string) ([]promVectorSample, error)
	QueryRange(context.Context, string, time.Time, time.Time, time.Duration) ([]promRangeSample, error)
}

type PrometheusQueryClient struct {
	baseURL    string
	httpClient *http.Client
}

var (
	promQueryTelemetryOnce sync.Once
	promQueryTracer        trace.Tracer
	promQueryRequests      metric.Int64Counter
	promQueryErrors        metric.Int64Counter
	promQueryLatency       metric.Float64Histogram
)

type promQueryResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type promQueryRangeResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]any           `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func NewPrometheusQueryClient(baseURL string, httpClient *http.Client) (*PrometheusQueryClient, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("consoleapi/providers: prometheus base url is empty")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	initPromQueryTelemetry()
	return &PrometheusQueryClient{
		baseURL:    strings.TrimRight(trimmed, "/"),
		httpClient: httpClient,
	}, nil
}

func (c *PrometheusQueryClient) QueryVector(ctx context.Context, query string) ([]promVectorSample, error) {
	startedAt := time.Now()
	initPromQueryTelemetry()
	ctx, span := promQueryTracer.Start(ctx, "prometheus.query.vector")
	defer span.End()
	queryTypeAttr := attribute.String("prometheus.query_type", "vector")
	promQueryRequests.Add(ctx, 1, metric.WithAttributes(queryTypeAttr))

	if c == nil {
		err := fmt.Errorf("consoleapi/providers: prometheus query client is nil")
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	endpoint, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	values := endpoint.Query()
	values.Set("query", query)
	endpoint.RawQuery = values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("consoleapi/providers: prometheus query status %d", response.StatusCode)
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	var payload promQueryResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	if payload.Status != "success" {
		err := fmt.Errorf("consoleapi/providers: prometheus query failed: %s %s", payload.ErrorType, payload.Error)
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	if payload.Data.ResultType != "vector" {
		err := fmt.Errorf("consoleapi/providers: unsupported prometheus result type %q", payload.Data.ResultType)
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	samples := make([]promVectorSample, 0, len(payload.Data.Result))
	for _, item := range payload.Data.Result {
		number, ok := parsePromValue(item.Value)
		if !ok {
			continue
		}
		samples = append(samples, promVectorSample{
			Metric: item.Metric,
			Value:  number,
		})
	}
	promQueryLatency.Record(ctx, time.Since(startedAt).Seconds(), metric.WithAttributes(queryTypeAttr, attribute.String("result", "ok")))
	return samples, nil
}

func (c *PrometheusQueryClient) QueryRange(
	ctx context.Context,
	query string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]promRangeSample, error) {
	startedAt := time.Now()
	initPromQueryTelemetry()
	ctx, span := promQueryTracer.Start(ctx, "prometheus.query.range")
	defer span.End()
	queryTypeAttr := attribute.String("prometheus.query_type", "range")
	promQueryRequests.Add(ctx, 1, metric.WithAttributes(queryTypeAttr))

	if c == nil {
		err := fmt.Errorf("consoleapi/providers: prometheus query client is nil")
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	if step <= 0 {
		err := fmt.Errorf("consoleapi/providers: prometheus query range step must be positive")
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	endpoint, err := url.Parse(c.baseURL + "/api/v1/query_range")
	if err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	values := endpoint.Query()
	values.Set("query", query)
	values.Set("start", strconv.FormatFloat(float64(start.Unix()), 'f', -1, 64))
	values.Set("end", strconv.FormatFloat(float64(end.Unix()), 'f', -1, 64))
	values.Set("step", strconv.FormatFloat(step.Seconds(), 'f', -1, 64))
	endpoint.RawQuery = values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("consoleapi/providers: prometheus query range status %d", response.StatusCode)
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	var payload promQueryRangeResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	if payload.Status != "success" {
		err := fmt.Errorf("consoleapi/providers: prometheus query range failed: %s %s", payload.ErrorType, payload.Error)
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	if payload.Data.ResultType != "matrix" {
		err := fmt.Errorf("consoleapi/providers: unsupported prometheus query range result type %q", payload.Data.ResultType)
		recordPromQueryFailure(ctx, span, queryTypeAttr, err)
		return nil, err
	}
	samples := make([]promRangeSample, 0, len(payload.Data.Result))
	for _, item := range payload.Data.Result {
		points := make([]promRangePoint, 0, len(item.Values))
		for _, rawPoint := range item.Values {
			if len(rawPoint) < 2 {
				continue
			}
			tsValue, ok := parsePromTimestamp(rawPoint[0])
			if !ok {
				continue
			}
			number, ok := parsePromNumber(rawPoint[1])
			if !ok {
				continue
			}
			points = append(points, promRangePoint{
				Timestamp: time.Unix(tsValue, 0).UTC(),
				Value:     number,
			})
		}
		samples = append(samples, promRangeSample{
			Metric: item.Metric,
			Values: points,
		})
	}
	promQueryLatency.Record(ctx, time.Since(startedAt).Seconds(), metric.WithAttributes(queryTypeAttr, attribute.String("result", "ok")))
	return samples, nil
}

func initPromQueryTelemetry() {
	promQueryTelemetryOnce.Do(func() {
		promQueryTracer = otel.Tracer("console-api/providers/prometheus")
		meter := otel.Meter("console-api/providers/prometheus")
		promQueryRequests, _ = meter.Int64Counter(
			"gen_ai.provider.prometheus.query.requests",
			metric.WithUnit("{request}"),
			metric.WithDescription("Number of provider observability Prometheus queries."),
		)
		promQueryErrors, _ = meter.Int64Counter(
			"gen_ai.provider.prometheus.query.errors",
			metric.WithUnit("{error}"),
			metric.WithDescription("Number of failed provider observability Prometheus queries."),
		)
		promQueryLatency, _ = meter.Float64Histogram(
			"gen_ai.provider.prometheus.query.duration",
			metric.WithUnit("s"),
			metric.WithDescription("Duration of provider observability Prometheus queries."),
		)
		if promQueryRequests == nil || promQueryErrors == nil || promQueryLatency == nil {
			noopMeter := noop.NewMeterProvider().Meter("console-api/providers/prometheus")
			promQueryRequests, _ = noopMeter.Int64Counter("gen_ai.provider.prometheus.query.requests")
			promQueryErrors, _ = noopMeter.Int64Counter("gen_ai.provider.prometheus.query.errors")
			promQueryLatency, _ = noopMeter.Float64Histogram("gen_ai.provider.prometheus.query.duration")
		}
	})
}

func recordPromQueryFailure(ctx context.Context, span trace.Span, queryTypeAttr attribute.KeyValue, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	promQueryErrors.Add(ctx, 1, metric.WithAttributes(queryTypeAttr))
}

func parsePromValue(rawValue []any) (float64, bool) {
	if len(rawValue) < 2 {
		return 0, false
	}
	return parsePromNumber(rawValue[1])
}

func parsePromTimestamp(raw any) (int64, bool) {
	switch value := raw.(type) {
	case float64:
		return int64(value), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, false
		}
		return int64(parsed), true
	default:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(raw)), 64)
		if err != nil {
			return 0, false
		}
		return int64(parsed), true
	}
}

func parsePromNumber(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case string:
		parsed, err := strconv.ParseFloat(strings.Trim(strings.TrimSpace(value), `"`), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		parsed, err := strconv.ParseFloat(strings.Trim(strings.TrimSpace(fmt.Sprint(raw)), `"`), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
}
