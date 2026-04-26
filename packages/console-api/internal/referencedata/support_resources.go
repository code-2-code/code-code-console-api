package referencedata

import (
	"context"
	"net/http"

	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/console-api/internal/httpjson"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

type supportResourceService interface {
	ListVendors(context.Context) ([]*supportv1.Vendor, error)
	ListCLIs(context.Context) ([]*supportv1.CLI, error)
	ListProviderSurfaces(context.Context) ([]*providerv1.ProviderSurface, error)
}

// RegisterSupportResourceHandlers registers support-service backed resource
// metadata routes.
func RegisterSupportResourceHandlers(mux *http.ServeMux, service supportResourceService) {
	mux.HandleFunc("/api/support/vendors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.ListVendors(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_support_vendors_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &supportv1.ListVendorsResponse{Items: items})
	})
	mux.HandleFunc("/api/support/clis", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.ListCLIs(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_support_clis_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &supportv1.ListCLIsResponse{Items: items})
	})
	mux.HandleFunc("/api/support/provider-surfaces", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.ListProviderSurfaces(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_provider_surfaces_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &supportv1.ListProviderSurfacesResponse{Items: items})
	})
}
