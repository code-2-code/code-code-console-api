package mcpservers

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	mcpv1 "code-code.internal/go-contract/platform/mcp/v1"
)

type service interface {
	List(context.Context) ([]*managementv1.MCPServerListItem, error)
	Get(context.Context, string) (*mcpv1.MCPServer, error)
	Create(context.Context, *managementv1.UpsertMCPServerRequest) (*mcpv1.MCPServer, error)
	Update(context.Context, string, *managementv1.UpsertMCPServerRequest) (*mcpv1.MCPServer, error)
	Delete(context.Context, string) error
}

func RegisterHandlers(mux *http.ServeMux, service service) {
	mux.HandleFunc("/api/mcps", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := service.List(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_mcps_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListMCPServersResponse{Items: items})
		case http.MethodPost:
			var request managementv1.UpsertMCPServerRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			mcp, err := service.Create(r.Context(), &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "create_mcp_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusCreated, mcp)
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/mcps/", func(w http.ResponseWriter, r *http.Request) {
		mcpID := strings.TrimPrefix(r.URL.Path, "/api/mcps/")
		if mcpID == "" || strings.Contains(mcpID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "mcp route not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			mcp, err := service.Get(r.Context(), mcpID)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "get_mcp_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, mcp)
		case http.MethodPut:
			var request managementv1.UpsertMCPServerRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			mcp, err := service.Update(r.Context(), mcpID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_mcp_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, mcp)
		case http.MethodDelete:
			if err := service.Delete(r.Context(), mcpID); err != nil {
				httpjson.WriteServiceError(w, http.StatusConflict, "delete_mcp_failed", err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}
