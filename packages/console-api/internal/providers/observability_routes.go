package providers

import (
	"encoding/json"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
)

func RegisterObservabilityHandlers(mux *http.ServeMux, service *ObservabilityService) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc("/api/providers/observability:probe-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		payload, err := service.ProbeAll(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "provider_observability_probe_all_failed", err)
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/api/providers/observability:probe", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		var request ProviderObservabilityProbeRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		payload, err := service.ProbeProviders(r.Context(), request.ProviderIDs)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "provider_observability_probe_failed", err)
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/api/providers/observability/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		window, err := parseWindowQuery(r.URL.Query().Get("window"))
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_window", err.Error())
			return
		}
		payload, err := service.Summary(r.Context(), window)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "provider_observability_summary_failed", err)
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/api/providers/observability/providers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		providerID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/providers/observability/providers/"))
		if providerID == "" || strings.Contains(providerID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider observability route not found")
			return
		}
		window, err := parseWindowQuery(r.URL.Query().Get("window"))
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_window", err.Error())
			return
		}
		view, err := parseProviderObservabilityView(r.URL.Query().Get("view"))
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_view", err.Error())
			return
		}
		payload, err := service.Provider(r.Context(), providerID, window, view)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "provider_observability_failed", err)
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, payload)
	})
}

func parseWindowQuery(raw string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "15m":
		return "15m", nil
	case "5m":
		return "5m", nil
	case "1h":
		return "1h", nil
	case "6h":
		return "6h", nil
	case "24h":
		return "24h", nil
	default:
		return "", &windowError{raw: raw}
	}
}

type windowError struct {
	raw string
}

func (e *windowError) Error() string {
	return "window must be one of 5m, 15m, 1h, 6h, 24h"
}
