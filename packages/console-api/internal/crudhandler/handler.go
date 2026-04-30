// Package crudhandler provides a generic HTTP CRUD handler factory for
// management resources that share the same List/Get/Create/Update/Delete
// interface shape.
package crudhandler

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	"google.golang.org/protobuf/proto"
)

// Service defines the CRUD operations for a management resource.
// L is the list item type, E is the entity type, R is the upsert request type.
type Service[L any, E proto.Message, R proto.Message] interface {
	List(context.Context) ([]L, error)
	Get(context.Context, string) (E, error)
	Create(context.Context, R) (E, error)
	Update(context.Context, string, R) (E, error)
	Delete(context.Context, string) error
}

// Config defines the resource-specific naming and response wrapping.
type Config[L any, E proto.Message, R proto.Message] struct {
	// ResourceName is the URL path segment, e.g. "skills".
	ResourceName string
	// WrapList wraps the list items into a proto response message.
	WrapList func([]L) proto.Message
	// NewRequest allocates a zero-value upsert request for JSON decoding.
	NewRequest func() R
}

// Register registers the standard CRUD endpoints on the given mux.
func Register[L any, E proto.Message, R proto.Message](
	mux *http.ServeMux,
	svc Service[L, E, R],
	cfg Config[L, E, R],
) {
	basePath := "/api/" + cfg.ResourceName
	itemPath := basePath + "/"

	mux.HandleFunc(basePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := svc.List(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_"+cfg.ResourceName+"_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, cfg.WrapList(items))
		case http.MethodPost:
			request := cfg.NewRequest()
			if err := httpjson.DecodeProtoJSON(r, request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			entity, err := svc.Create(r.Context(), request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "create_"+cfg.ResourceName+"_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusCreated, entity)
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc(itemPath, func(w http.ResponseWriter, r *http.Request) {
		resourceID := strings.TrimPrefix(r.URL.Path, itemPath)
		if resourceID == "" || strings.Contains(resourceID, "/") {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", cfg.ResourceName+" route not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			entity, err := svc.Get(r.Context(), resourceID)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "get_"+cfg.ResourceName+"_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, entity)
		case http.MethodPut:
			request := cfg.NewRequest()
			if err := httpjson.DecodeProtoJSON(r, request); err != nil {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			entity, err := svc.Update(r.Context(), resourceID, request)
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusBadRequest, "update_"+cfg.ResourceName+"_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, entity)
		case http.MethodDelete:
			if err := svc.Delete(r.Context(), resourceID); err != nil {
				httpjson.WriteServiceError(w, http.StatusConflict, "delete_"+cfg.ResourceName+"_failed", err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}
