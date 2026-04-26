package skills

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	skillv1 "code-code.internal/go-contract/platform/skill/v1"
)

type service interface {
	List(context.Context) ([]*managementv1.SkillListItem, error)
	Get(context.Context, string) (*skillv1.Skill, error)
	Create(context.Context, *managementv1.UpsertSkillRequest) (*skillv1.Skill, error)
	Update(context.Context, string, *managementv1.UpsertSkillRequest) (*skillv1.Skill, error)
	Delete(context.Context, string) error
}

func RegisterHandlers(mux *http.ServeMux, service service) {
	mux.HandleFunc("/api/skills", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := service.List(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_skills_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListSkillsResponse{Items: items})
		case http.MethodPost:
			var request managementv1.UpsertSkillRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			skill, err := service.Create(r.Context(), &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "create_skill_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusCreated, skill)
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/skills/", func(w http.ResponseWriter, r *http.Request) {
		skillID := strings.TrimPrefix(r.URL.Path, "/api/skills/")
		if skillID == "" || strings.Contains(skillID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "skill route not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			skill, err := service.Get(r.Context(), skillID)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "get_skill_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, skill)
		case http.MethodPut:
			var request managementv1.UpsertSkillRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			skill, err := service.Update(r.Context(), skillID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_skill_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, skill)
		case http.MethodDelete:
			if err := service.Delete(r.Context(), skillID); err != nil {
				httpjson.WriteServiceError(w, http.StatusConflict, "delete_skill_failed", err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}
