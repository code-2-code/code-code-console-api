package templates

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

type templateService interface {
	List(context.Context) ([]*managementv1.TemplateView, error)
	Apply(context.Context, string, *managementv1.ApplyTemplateRequest) (*managementv1.ApplyTemplateResult, error)
}

// RegisterHandlers registers template routes onto the provided mux.
func RegisterHandlers(mux *http.ServeMux, service templateService) {
	mux.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.List(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_templates_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListTemplatesResponse{Items: items})
	})

	mux.HandleFunc("/api/templates/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/templates/")
		templateID, found := strings.CutSuffix(path, "/apply")
		if !found || templateID == "" || strings.Contains(templateID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "template route not found")
			return
		}
		var request managementv1.ApplyTemplateRequest
		if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		result, err := service.Apply(r.Context(), templateID, &request)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "apply_template_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, result)
	})
}
