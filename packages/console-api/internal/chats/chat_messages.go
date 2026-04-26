package chats

import (
	"encoding/json"
	"net/http"

	"code-code.internal/console-api/internal/httpjson"
	"code-code.internal/go-contract/agui"
	"google.golang.org/protobuf/types/known/structpb"
)

type chatMessagesView struct {
	Messages      []json.RawMessage `json:"messages"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
}

func handleListChatMessages(w http.ResponseWriter, r *http.Request, chats chatService, chatID string) {
	if r.Method != http.MethodGet {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	pageSize, err := parseChatListPageSize(r.URL.Query().Get("pageSize"))
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_page_size", err.Error())
		return
	}
	messages, nextPageToken, err := chats.ListChatMessages(r.Context(), chatID, pageSize, r.URL.Query().Get("pageToken"))
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "list_chat_messages_failed", err)
		return
	}
	view, err := buildChatMessagesView(messages, nextPageToken)
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "build_chat_messages_failed", err.Error())
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, view)
}

func buildChatMessagesView(messages []*structpb.Struct, nextPageToken string) (*chatMessagesView, error) {
	view := &chatMessagesView{Messages: make([]json.RawMessage, 0, len(messages)), NextPageToken: nextPageToken}
	for _, message := range messages {
		item, err := agui.MessageFromStruct(message)
		if err != nil {
			return nil, err
		}
		raw, err := agui.MessageRaw(item)
		if err != nil {
			return nil, err
		}
		view.Messages = append(view.Messages, raw)
	}
	return view, nil
}
