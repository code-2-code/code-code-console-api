package referencedata

import (
	"context"
	"net/http"

	"code-code.internal/console-api/internal/httpjson"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	productinfov1 "code-code.internal/go-contract/product_info/v1"
)

type supportResourceService interface {
	ListVendors(context.Context) ([]*supportv1.Vendor, error)
	ListCLIs(context.Context) ([]*supportv1.CLI, error)
	ListProductInfos(context.Context) ([]*productinfov1.ProductInfo, error)
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
	mux.HandleFunc("/api/support/product-infos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.ListProductInfos(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_support_product_infos_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &supportv1.ListProductInfosResponse{Items: items})
	})
}
