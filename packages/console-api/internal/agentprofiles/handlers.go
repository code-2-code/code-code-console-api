package agentprofiles

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	agentprofilev1 "code-code.internal/go-contract/platform/agent_profile/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

type service interface {
	List(context.Context) ([]*managementv1.AgentProfileListItem, error)
	Get(context.Context, string) (*agentprofilev1.AgentProfile, error)
	Create(context.Context, *managementv1.UpsertAgentProfileRequest) (*agentprofilev1.AgentProfile, error)
	Update(context.Context, string, *managementv1.UpsertAgentProfileRequest) (*agentprofilev1.AgentProfile, error)
	Delete(context.Context, string) error
}

func RegisterHandlers(mux *http.ServeMux, service service) {
	mux.HandleFunc("/api/agent-profiles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := service.List(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_agent_profiles_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListAgentProfilesResponse{Items: items})
		case http.MethodPost:
			var request managementv1.UpsertAgentProfileRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			profile, err := service.Create(r.Context(), &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "create_agent_profile_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusCreated, profile)
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/agent-profiles/", func(w http.ResponseWriter, r *http.Request) {
		profileID := strings.TrimPrefix(r.URL.Path, "/api/agent-profiles/")
		if profileID == "" || strings.Contains(profileID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "agent profile route not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			profile, err := service.Get(r.Context(), profileID)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "get_agent_profile_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, profile)
		case http.MethodPut:
			var request managementv1.UpsertAgentProfileRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			profile, err := service.Update(r.Context(), profileID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_agent_profile_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, profile)
		case http.MethodDelete:
			if err := service.Delete(r.Context(), profileID); err != nil {
				httpjson.WriteServiceError(w, http.StatusConflict, "delete_agent_profile_failed", err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}
