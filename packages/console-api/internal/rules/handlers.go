package rules

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	rulev1 "code-code.internal/go-contract/platform/rule/v1"
)

type service interface {
	List(context.Context) ([]*managementv1.RuleListItem, error)
	Get(context.Context, string) (*rulev1.Rule, error)
	Create(context.Context, *managementv1.UpsertRuleRequest) (*rulev1.Rule, error)
	Update(context.Context, string, *managementv1.UpsertRuleRequest) (*rulev1.Rule, error)
	Delete(context.Context, string) error
}

func RegisterHandlers(mux *http.ServeMux, service service) {
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := service.List(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_rules_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListRulesResponse{Items: items})
		case http.MethodPost:
			var request managementv1.UpsertRuleRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			rule, err := service.Create(r.Context(), &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "create_rule_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusCreated, rule)
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/rules/", func(w http.ResponseWriter, r *http.Request) {
		ruleID := strings.TrimPrefix(r.URL.Path, "/api/rules/")
		if ruleID == "" || strings.Contains(ruleID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "rule route not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			rule, err := service.Get(r.Context(), ruleID)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "get_rule_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, rule)
		case http.MethodPut:
			var request managementv1.UpsertRuleRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			rule, err := service.Update(r.Context(), ruleID, &request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_rule_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, rule)
		case http.MethodDelete:
			if err := service.Delete(r.Context(), ruleID); err != nil {
				httpjson.WriteServiceError(w, http.StatusConflict, "delete_rule_failed", err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}
