package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	providerStatusEventName              = "provider-status"
	providerStatusEventsHeartbeat        = 15 * time.Second
	providerStatusEventsChannelBufferLen = 8
)

var providerStatusEventMarshaler = protojson.MarshalOptions{EmitUnpopulated: true}

type providerStatusStreamItem struct {
	payload string
	err     error
}

func writeProviderStatusEvents(w http.ResponseWriter, r *http.Request, service providerService, providerIDs []string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	items := make(chan providerStatusStreamItem, providerStatusEventsChannelBufferLen)
	go streamProviderStatusEvents(ctx, service, providerIDs, items)

	heartbeat := time.NewTicker(providerStatusEventsHeartbeat)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case item, ok := <-items:
			if !ok || item.err != nil {
				return
			}
			if err := writeProviderStatusSSEMessage(w, item.payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func streamProviderStatusEvents(
	ctx context.Context,
	service providerService,
	providerIDs []string,
	items chan<- providerStatusStreamItem,
) {
	defer close(items)
	err := service.WatchStatusEvents(ctx, providerIDs, func(event *managementv1.ProviderStatusEvent) error {
		payload, err := providerStatusEventPayload(event)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case items <- providerStatusStreamItem{payload: payload}:
			return nil
		}
	})
	if err != nil && ctx.Err() == nil {
		items <- providerStatusStreamItem{err: err}
	}
}

func providerStatusEventPayload(event *managementv1.ProviderStatusEvent) (string, error) {
	data, err := providerStatusEventMarshaler.Marshal(event)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeProviderStatusSSEMessage(w http.ResponseWriter, payload string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", providerStatusEventName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}

func providerIDsFromQuery(r *http.Request) []string {
	values := append([]string(nil), r.URL.Query()["provider_id"]...)
	for _, value := range r.URL.Query()["provider_ids"] {
		values = append(values, strings.Split(value, ",")...)
	}
	return normalizeProviderIDs(values)
}

func normalizeProviderIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
