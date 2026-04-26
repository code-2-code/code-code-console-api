package providers

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

type providerService interface {
	ListProviderSurfaceMetadata(context.Context) ([]*providerv1.ProviderSurface, error)
	ListProviders(context.Context) ([]*managementv1.ProviderView, error)
	ListProviderSurfaceBindings(context.Context) ([]*managementv1.ProviderSurfaceBindingView, error)
	UpdateProvider(context.Context, string, *managementv1.UpdateProviderRequest) (*managementv1.ProviderView, error)
	UpdateProviderAuthentication(context.Context, string, *managementv1.UpdateProviderAuthenticationRequest) (*managementv1.UpdateProviderAuthenticationResponse, error)
	UpdateProviderObservabilityAuthentication(context.Context, string, *managementv1.UpdateProviderObservabilityAuthenticationRequest) (*managementv1.ProviderView, error)
	DeleteProvider(context.Context, string) error
	CreateProviderSurfaceBinding(context.Context, *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error)
	UpdateProviderSurfaceBinding(context.Context, string, *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error)
	DeleteProviderSurfaceBinding(context.Context, string) error
	Connect(context.Context, *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error)
	GetConnectSession(context.Context, string) (*managementv1.ProviderConnectSessionView, error)
	WatchStatusEvents(context.Context, []string, func(*managementv1.ProviderStatusEvent) error) error
}

// RegisterHandlers registers provider routes onto the provided mux.
func RegisterHandlers(mux *http.ServeMux, service providerService) {
	mux.HandleFunc("/api/providers/surfaces", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.ListProviderSurfaceMetadata(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_provider_surfaces_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListProviderSurfacesResponse{Items: items})
	})

	mux.HandleFunc("/api/providers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.ListProviders(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_providers_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListProvidersResponse{Items: items})
	})

	mux.HandleFunc("/api/providers/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		writeProviderStatusEvents(w, r, service, providerIDsFromQuery(r))
	})

	mux.HandleFunc("/api/providers/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/providers/")
		if rest == "" {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider route not found")
			return
		}
		if strings.HasSuffix(rest, "/observability-authentication") {
			providerID := strings.TrimSuffix(rest, "/observability-authentication")
			if providerID == "" || strings.Contains(providerID, "/") {
				httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider route not found")
				return
			}
			if r.Method != http.MethodPost {
				httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
				return
			}
			var request managementv1.UpdateProviderObservabilityAuthenticationRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			item, err := service.UpdateProviderObservabilityAuthentication(r.Context(), providerID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_provider_observability_authentication_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, item)
			return
		}
		if strings.HasSuffix(rest, "/authentication") {
			providerID := strings.TrimSuffix(rest, "/authentication")
			if providerID == "" || strings.Contains(providerID, "/") {
				httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider route not found")
				return
			}
			if r.Method != http.MethodPost {
				httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
				return
			}
			var request managementv1.UpdateProviderAuthenticationRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			response, err := service.UpdateProviderAuthentication(r.Context(), providerID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_provider_authentication_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, response)
			return
		}
		if strings.Contains(rest, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider route not found")
			return
		}
		providerID := rest
		switch r.Method {
		case http.MethodPut:
			var request managementv1.UpdateProviderRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			item, err := service.UpdateProvider(r.Context(), providerID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_provider_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if err := service.DeleteProvider(r.Context(), providerID); err != nil {
				httpjson.WriteServiceError(w, http.StatusConflict, "delete_provider_failed", err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/providers/surface-bindings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := service.ListProviderSurfaceBindings(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_provider_surface_bindings_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListProviderSurfaceBindingsResponse{Items: items})
		case http.MethodPost:
			var request managementv1.UpsertProviderSurfaceBindingRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			item, err := service.CreateProviderSurfaceBinding(r.Context(), &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "create_provider_surface_binding_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusCreated, item)
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/providers/surface-bindings/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/providers/surface-bindings/")
		if rest == "" {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider surface binding route not found")
			return
		}
		if strings.Contains(rest, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider surface binding route not found")
			return
		}
		surfaceID := rest
		switch r.Method {
		case http.MethodPut:
			var request managementv1.UpsertProviderSurfaceBindingRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			item, err := service.UpdateProviderSurfaceBinding(r.Context(), surfaceID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_provider_surface_binding_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if err := service.DeleteProviderSurfaceBinding(r.Context(), surfaceID); err != nil {
				httpjson.WriteServiceError(w, http.StatusConflict, "delete_provider_surface_binding_failed", err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/providers/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		var request managementv1.ConnectProviderRequest
		if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		response, err := service.Connect(r.Context(), &request)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "connect_provider_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusCreated, response)
	})

	mux.HandleFunc("/api/providers/connect/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		sessionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/providers/connect/sessions/"))
		if sessionID == "" || strings.Contains(sessionID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "provider connect session route not found")
			return
		}
		session, err := service.GetConnectSession(r.Context(), sessionID)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "get_provider_connect_session_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.GetProviderConnectSessionResponse{Session: session})
	})
}
