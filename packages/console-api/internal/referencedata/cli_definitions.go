package referencedata

import (
	"context"
	"net/http"

	"code-code.internal/console-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

type cliDefinitionService interface {
	List(context.Context) ([]*managementv1.CLIDefinitionView, error)
}

// RegisterCLIDefinitionHandlers registers read-only CLI definition HTTP routes.
func RegisterCLIDefinitionHandlers(mux *http.ServeMux, service cliDefinitionService) {
	mux.HandleFunc("/api/cli-definitions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		items, err := service.List(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_cli_definitions_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListCLIDefinitionsResponse{Items: items})
	})
}
