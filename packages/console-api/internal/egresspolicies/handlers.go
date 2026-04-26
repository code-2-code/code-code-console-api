package egresspolicies

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	egressv1 "code-code.internal/go-contract/egress/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

type service interface {
	List(context.Context) ([]*managementv1.EgressPolicyView, error)
	Update(context.Context, string, *egressv1.EgressPolicy) (*managementv1.EgressPolicyView, error)
}

func RegisterHandlers(mux *http.ServeMux, service service) {
	mux.HandleFunc("/api/network/egress-policies", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := service.List(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_egress_policies_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListEgressPoliciesResponse{Items: items})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/api/network/egress-policies/", func(w http.ResponseWriter, r *http.Request) {
		policyID := strings.TrimPrefix(r.URL.Path, "/api/network/egress-policies/")
		if policyID == "" || strings.Contains(policyID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "egress policy route not found")
			return
		}
		switch r.Method {
		case http.MethodPut:
			var request managementv1.UpdateEgressPolicyRequest
			if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_egress_policy_failed", err)
				return
			}
			item, err := service.Update(r.Context(), policyID, request.GetPolicy())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_egress_policy_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.UpdateEgressPolicyResponse{Item: item})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}
