package chats

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PostgresChatStore struct {
	db postgresChatDB
}

const defaultChatListPageSize int32 = 20
const maxChatListPageSize int32 = 100

type postgresChatDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewPostgresChatStore(ctx context.Context, pool *pgxpool.Pool) (*PostgresChatStore, error) {
	if pool == nil {
		return nil, status.Error(codes.InvalidArgument, "postgres pool is required")
	}
	store := newPostgresChatStore(pool)
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func newPostgresChatStore(db postgresChatDB) *PostgresChatStore {
	return &PostgresChatStore{db: db}
}

func (s *PostgresChatStore) withTx(tx pgx.Tx) *PostgresChatStore {
	return newPostgresChatStore(tx)
}

func (s *PostgresChatStore) Create(ctx context.Context, chat *chatv1.Chat) (*chatv1.Chat, error) {
	next, err := normalizeStoredChat(chat)
	if err != nil {
		return nil, err
	}
	now := timestamppb.Now()
	next.CreatedAt = now
	next.UpdatedAt = now
	payload, err := marshalChatPayload(next)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(ctx, `
insert into platform_chats (id, payload, created_at, updated_at)
values ($1, $2, now(), now())
`, next.GetChatId(), payload); err != nil {
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "chat already exists")
		}
		return nil, err
	}
	return cloneChat(next), nil
}

func (s *PostgresChatStore) Get(ctx context.Context, chatID string) (*chatv1.Chat, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	var payload []byte
	if err := s.db.QueryRow(ctx, `
select payload
from platform_chats
where id = $1
`, chatID).Scan(&payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "chat not found")
		}
		return nil, err
	}
	return unmarshalChatPayload(payload)
}

func (s *PostgresChatStore) Update(ctx context.Context, chat *chatv1.Chat) (*chatv1.Chat, error) {
	next, err := normalizeStoredChat(chat)
	if err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, next.GetChatId())
	if err != nil {
		return nil, err
	}
	next.CreatedAt = current.GetCreatedAt()
	next.UpdatedAt = timestamppb.Now()
	payload, err := marshalChatPayload(next)
	if err != nil {
		return nil, err
	}
	tag, err := s.db.Exec(ctx, `
update platform_chats
set payload = $2,
	updated_at = now()
where id = $1
`, next.GetChatId(), payload)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "chat not found")
	}
	return cloneChat(next), nil
}

func (s *PostgresChatStore) List(ctx context.Context, scopeID string, pageSize int32, pageToken string) ([]*chatv1.Chat, string, error) {
	scopeID = firstNonEmpty(scopeID, defaultChatScopeID)
	limit := normalizeChatListPageSize(pageSize)
	cursorUpdatedAt, cursorID, hasCursor, err := parseChatListPageToken(pageToken)
	if err != nil {
		return nil, "", err
	}
	rows, err := s.db.Query(ctx, `
select payload, updated_at, id
from platform_chats
where coalesce(payload->>'scope_id', 'default') = $1
	and (not $2::boolean or (updated_at, id) < ($3::timestamptz, $4::text))
order by updated_at desc, id desc
limit $5
`, scopeID, hasCursor, cursorUpdatedAt, cursorID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	type listedChat struct {
		chat      *chatv1.Chat
		updatedAt time.Time
		id        string
	}
	rowsOut := make([]listedChat, 0, limit+1)
	for rows.Next() {
		var payload []byte
		var updatedAt time.Time
		var id string
		if err := rows.Scan(&payload, &updatedAt, &id); err != nil {
			return nil, "", err
		}
		chat, err := unmarshalChatPayload(payload)
		if err != nil {
			return nil, "", err
		}
		rowsOut = append(rowsOut, listedChat{chat: chat, updatedAt: updatedAt, id: id})
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	chats := make([]*chatv1.Chat, 0, minInt(len(rowsOut), int(limit)))
	for i := range rowsOut {
		if int32(i) >= limit {
			break
		}
		chats = append(chats, rowsOut[i].chat)
	}
	if int32(len(rowsOut)) <= limit {
		return chats, "", nil
	}
	last := rowsOut[limit-1]
	return chats, encodeChatListPageToken(last.updatedAt, last.id), nil
}

func (s *PostgresChatStore) ensureSchema(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
create table if not exists platform_chats (
	id text primary key,
	payload jsonb not null,
	generation bigint not null default 1,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create index if not exists platform_chats_scope_updated_id_idx on platform_chats (
	((coalesce(payload->>'scope_id', 'default'))),
	updated_at desc,
	id desc
);
`)
	return err
}

func normalizeStoredChat(chat *chatv1.Chat) (*chatv1.Chat, error) {
	next := storedChat(chat)
	if next == nil {
		return nil, status.Error(codes.InvalidArgument, "chat is required")
	}
	next.ChatId = strings.TrimSpace(next.GetChatId())
	if next.GetChatId() == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	next.ScopeId = firstNonEmpty(next.GetScopeId(), defaultChatScopeID)
	next.DisplayName = strings.TrimSpace(next.GetDisplayName())
	next.SessionId = firstNonEmpty(next.GetSessionId(), next.GetChatId())
	return next, nil
}

func normalizeChatListPageSize(pageSize int32) int32 {
	if pageSize <= 0 {
		return defaultChatListPageSize
	}
	if pageSize > maxChatListPageSize {
		return maxChatListPageSize
	}
	return pageSize
}

func parseChatListPageToken(token string) (time.Time, string, bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return time.Time{}, "", false, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, "", false, status.Error(codes.InvalidArgument, "page_token is invalid")
	}
	updatedAtText, id, ok := strings.Cut(string(decoded), "|")
	if !ok || strings.TrimSpace(id) == "" {
		return time.Time{}, "", false, status.Error(codes.InvalidArgument, "page_token is invalid")
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtText)
	if err != nil {
		return time.Time{}, "", false, status.Error(codes.InvalidArgument, "page_token is invalid")
	}
	return updatedAt, id, true, nil
}

func encodeChatListPageToken(updatedAt time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(updatedAt.UTC().Format(time.RFC3339Nano) + "|" + strings.TrimSpace(id)))
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func marshalChatPayload(chat *chatv1.Chat) ([]byte, error) {
	return (protojson.MarshalOptions{UseProtoNames: true}).Marshal(storedChat(chat))
}

func unmarshalChatPayload(payload []byte) (*chatv1.Chat, error) {
	chat := &chatv1.Chat{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, chat); err != nil {
		return nil, err
	}
	return storedChat(chat), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
