package chats

import (
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
)

func handleAGUICapabilities(w http.ResponseWriter, r *http.Request, chats chatService, chatID string) {
	if r.Method != http.MethodGet {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat not found")
		return
	}
	chat, err := chats.GetChat(r.Context(), chatID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "get_chat_failed", err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, aguiCapabilities(chatID, chat))
}

func aguiCapabilities(chatID string, chat *chatv1.Chat) map[string]any {
	name := ""
	if chat != nil {
		name = strings.TrimSpace(chat.GetDisplayName())
	}
	if name == "" {
		name = strings.TrimSpace(chatID)
	}
	return map[string]any{
		"identity": map[string]any{
			"name":        name,
			"type":        "code-code.chat-session",
			"provider":    "code-code",
			"description": "Chat-bound AG-UI session agent",
		},
		"transport": map[string]any{
			"streaming": true,
		},
		"tools": map[string]any{
			"supported": true,
		},
		"state": map[string]any{
			"snapshots":       true,
			"persistentState": true,
		},
		"reasoning": map[string]any{
			"supported": true,
			"streaming": true,
		},
		"custom": map[string]any{
			"routeOwner": "chat",
			"activity": map[string]any{
				"snapshots": true,
				"turnSteps": true,
			},
			"serialization": map[string]any{
				"messagesSnapshot": true,
				"runStartedInput":  true,
			},
		},
	}
}
