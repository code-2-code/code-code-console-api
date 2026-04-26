package chats

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	agentcorev1 "code-code.internal/go-contract/agent/core/v1"
	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type chatService interface {
	GetChat(context.Context, string) (*chatv1.Chat, error)
	CreateChat(context.Context, string, string, string, *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error)
	UpdateChatSessionSetup(context.Context, string, *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error)
	RenameChat(context.Context, string, string) (*chatv1.Chat, error)
	ListChats(context.Context, string, int32, string) ([]*chatv1.Chat, string, error)
	ListChatMessages(context.Context, string, int32, string) ([]*structpb.Struct, string, error)
}

type sessionControlService interface {
	Get(context.Context, string) (*agentsessionv1.AgentSessionState, error)
	CreateTurn(context.Context, string, string, string, *agentcorev1.RunRequest) (*managementv1.CreateAgentSessionActionResponse, error)
	ResetWarmState(context.Context, string, string) (*managementv1.ResetAgentSessionWarmStateResponse, error)
}

type turnService interface {
	Get(context.Context, string) (*managementv1.GetAgentSessionActionResponse, error)
	Stop(context.Context, string) (*managementv1.StopAgentSessionActionResponse, error)
	Retry(context.Context, string, string) (*managementv1.RetryAgentSessionActionResponse, error)
}

type runService interface {
	Get(context.Context, string) (*managementv1.GetAgentRunResponse, error)
}

type retryTurnRequest struct {
	NewTurnID string `json:"newTurnId"`
}

type resetWarmStateRequest struct {
	ActionID string `json:"actionId"`
}

type renameChatRequest struct {
	DisplayName string `json:"displayName"`
}

func RegisterHandlers(
	mux *http.ServeMux,
	chats chatService,
	sessions sessionControlService,
	turns turnService,
	runs runService,
	runOutputs runOutputStreamService,
	sessionRuntimeOptions sessionRuntimeOptionsService,
) {
	runOutputs = newRunOutputHub(runOutputs)

	mux.HandleFunc("/api/chats", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListChats(w, r, chats)
		case http.MethodPost:
			handlePostChat(w, r, chats, sessionRuntimeOptions)
		default:
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	if sessionRuntimeOptions != nil {
		mux.HandleFunc("/api/chats/session-runtime-options", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
				return
			}
			view, err := sessionRuntimeOptions.View(r.Context())
			if err != nil {
				httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_session_runtime_options_failed", err)
				return
			}
			httpjson.WriteProtoJSON(w, http.StatusOK, sessionRuntimeOptionsToProto(view))
		})
	}

	mux.HandleFunc("/api/chats/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/chats/")
		if path == "" {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat route not found")
			return
		}
		if strings.HasSuffix(path, ":rename") {
			handleRenameChat(w, r, chats, strings.TrimSuffix(path, ":rename"))
			return
		}
		if strings.HasSuffix(path, ":reset-warm-state") {
			handleResetWarmState(w, r, chats, sessions, strings.TrimSuffix(path, ":reset-warm-state"))
			return
		}
		parts := strings.Split(path, "/")
		if len(parts) == 1 {
			handleChat(w, r, chats, sessionRuntimeOptions, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "turns" {
			handleCreateTurn(w, r, chats, sessions, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "messages" {
			handleListChatMessages(w, r, chats, parts[0])
			return
		}
		if len(parts) == 3 && parts[1] == "session" && parts[2] == "ag-ui" {
			handleAGUIRun(w, r, chats, sessions, turns, runs, runOutputs, parts[0])
			return
		}
		if len(parts) == 4 && parts[1] == "session" && parts[2] == "ag-ui" && parts[3] == "capabilities" {
			handleAGUICapabilities(w, r, chats, parts[0])
			return
		}
		if len(parts) != 3 {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat route not found")
			return
		}
		chatID := strings.TrimSpace(parts[0])
		if chatID == "" {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat route not found")
			return
		}
		switch parts[1] {
		case "turns":
			handleTurn(w, r, chats, turns, chatID, parts[2])
		default:
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat route not found")
		}
	})
}

func handleListChats(w http.ResponseWriter, r *http.Request, chats chatService) {
	pageSize, err := parseChatListPageSize(r.URL.Query().Get("pageSize"))
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_page_size", err.Error())
		return
	}
	items, nextPageToken, err := chats.ListChats(
		r.Context(),
		r.URL.Query().Get("scopeId"),
		pageSize,
		r.URL.Query().Get("pageToken"),
	)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "list_chats_failed", err)
		return
	}
	view, err := buildChatListView(items, nextPageToken)
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "build_chat_list_failed", err.Error())
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, view)
}

func handlePostChat(w http.ResponseWriter, r *http.Request, chats chatService, sessionRuntimeOptions sessionRuntimeOptionsService) {
	var request putChatRequest
	if err := httpjson.DecodeJSON(r, &request); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	chatID, err := newChatID()
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "generate_chat_id_failed", err)
		return
	}
	view, err := upsertChat(r.Context(), chats, sessionRuntimeOptions, chatID, request)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "create_chat_failed", err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, view)
}

func handleChat(w http.ResponseWriter, r *http.Request, chats chatService, sessionRuntimeOptions sessionRuntimeOptionsService, chatID string) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat route not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		chat, err := chats.GetChat(r.Context(), chatID)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "get_chat_failed", err)
			return
		}
		view, err := buildChatView(chat)
		if err != nil {
			httpjson.WriteError(w, http.StatusInternalServerError, "build_chat_failed", err.Error())
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, view)
	case http.MethodPut:
		var request putChatRequest
		if err := httpjson.DecodeJSON(r, &request); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		view, err := upsertChat(r.Context(), chats, sessionRuntimeOptions, chatID, request)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "upsert_chat_failed", err)
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, view)
	default:
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func handleRenameChat(w http.ResponseWriter, r *http.Request, chats chatService, chatID string) {
	if r.Method != http.MethodPost {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" || strings.Contains(chatID, "/") {
		httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat route not found")
		return
	}
	var request renameChatRequest
	if err := httpjson.DecodeJSON(r, &request); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	chat, err := chats.RenameChat(r.Context(), chatID, request.DisplayName)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "rename_chat_failed", err)
		return
	}
	view, err := buildChatView(chat)
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "build_chat_failed", err.Error())
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, view)
}

func handleResetWarmState(w http.ResponseWriter, r *http.Request, chats chatService, sessions sessionControlService, chatID string) {
	if r.Method != http.MethodPost {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" || strings.Contains(chatID, "/") {
		httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat route not found")
		return
	}
	var request resetWarmStateRequest
	if r.ContentLength > 0 {
		if err := httpjson.DecodeJSON(r, &request); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	sessionID, err := boundSessionID(r.Context(), chats, chatID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "get_chat_failed", err)
		return
	}
	response, err := sessions.ResetWarmState(r.Context(), sessionID, request.ActionID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "reset_chat_warm_state_failed", err)
		return
	}
	httpjson.WriteProtoJSON(w, http.StatusOK, response.GetAction())
}

func handleTurn(w http.ResponseWriter, r *http.Request, chats chatService, turns turnService, chatID string, turnPath string) {
	if turnPath == "" {
		httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat turn route not found")
		return
	}
	switch {
	case strings.HasSuffix(turnPath, ":stop"):
		handleStopTurn(w, r, chats, turns, chatID, strings.TrimSuffix(turnPath, ":stop"))
	case strings.HasSuffix(turnPath, ":retry"):
		handleRetryTurn(w, r, chats, turns, chatID, strings.TrimSuffix(turnPath, ":retry"))
	default:
		handleGetTurn(w, r, chats, turns, chatID, turnPath)
	}
}

func handleCreateTurn(w http.ResponseWriter, r *http.Request, chats chatService, sessions sessionControlService, chatID string) {
	if r.Method != http.MethodPost {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat not found")
		return
	}
	var request managementv1.CreateAgentSessionActionRequest
	if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	turnID := strings.TrimSpace(request.GetTurnId())
	actionID := strings.TrimSpace(request.GetActionId())
	if turnID == "" && actionID == "" {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_turn_id", "turnId or actionId is required")
		return
	}
	if actionID == "" {
		actionID = turnID
	}
	if turnID == "" {
		turnID = actionID
	}
	sessionID, err := boundSessionID(r.Context(), chats, chatID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "get_chat_failed", err)
		return
	}
	response, err := sessions.CreateTurn(r.Context(), sessionID, actionID, turnID, request.GetRunRequest())
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "create_turn_failed", err)
		return
	}
	httpjson.WriteProtoJSON(w, http.StatusCreated, response.GetAction())
}

func handleGetTurn(w http.ResponseWriter, r *http.Request, chats chatService, turns turnService, chatID string, turnID string) {
	if r.Method != http.MethodGet {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	response, err := turns.Get(r.Context(), strings.TrimSpace(turnID))
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "get_turn_failed", err)
		return
	}
	if !belongsToChat(r.Context(), chats, response.GetAction().GetSpec().GetSessionId(), chatID) {
		httpjson.WriteServiceError(w, http.StatusNotFound, "get_turn_failed", status.Error(codes.NotFound, "turn not found"))
		return
	}
	httpjson.WriteProtoJSON(w, http.StatusOK, response.GetAction())
}

func handleStopTurn(w http.ResponseWriter, r *http.Request, chats chatService, turns turnService, chatID string, turnID string) {
	if r.Method != http.MethodPost {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !turnBelongsToChat(r.Context(), chats, turns, chatID, turnID) {
		httpjson.WriteServiceError(w, http.StatusNotFound, "stop_turn_failed", status.Error(codes.NotFound, "turn not found"))
		return
	}
	response, err := turns.Stop(r.Context(), strings.TrimSpace(turnID))
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "stop_turn_failed", err)
		return
	}
	httpjson.WriteProtoJSON(w, http.StatusOK, response.GetAction())
}

func handleRetryTurn(w http.ResponseWriter, r *http.Request, chats chatService, turns turnService, chatID string, turnID string) {
	if r.Method != http.MethodPost {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !turnBelongsToChat(r.Context(), chats, turns, chatID, turnID) {
		httpjson.WriteServiceError(w, http.StatusNotFound, "retry_turn_failed", status.Error(codes.NotFound, "turn not found"))
		return
	}
	var request retryTurnRequest
	if err := httpjson.DecodeJSON(r, &request); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	newTurnID := strings.TrimSpace(request.NewTurnID)
	if newTurnID == "" {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_turn_id", "newTurnId is required")
		return
	}
	response, err := turns.Retry(r.Context(), strings.TrimSpace(turnID), newTurnID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "retry_turn_failed", err)
		return
	}
	httpjson.WriteProtoJSON(w, http.StatusOK, response.GetAction())
}

func turnBelongsToChat(ctx context.Context, chats chatService, turns turnService, chatID string, turnID string) bool {
	response, err := turns.Get(ctx, strings.TrimSpace(turnID))
	if err != nil {
		return false
	}
	return belongsToChat(ctx, chats, response.GetAction().GetSpec().GetSessionId(), chatID)
}

func belongsToChat(ctx context.Context, chats chatService, sessionID string, chatID string) bool {
	bound, err := boundSessionID(ctx, chats, chatID)
	return err == nil && strings.TrimSpace(sessionID) != "" && strings.TrimSpace(sessionID) == bound
}

func boundSessionID(ctx context.Context, chats chatService, chatID string) (string, error) {
	chat, err := chats.GetChat(ctx, strings.TrimSpace(chatID))
	if err != nil {
		return "", err
	}
	return currentSessionID(chatID, chat), nil
}

func newRandomID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(buf), nil
}

func newChatID() (string, error) {
	return newRandomID("chat")
}

func parseChatListPageSize(value string) (int32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed < 0 {
		return 0, status.Error(codes.InvalidArgument, "pageSize is invalid")
	}
	return int32(parsed), nil
}
