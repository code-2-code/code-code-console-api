package chats

import (
	"context"
	"fmt"

	sessiondomain "code-code.internal/session"
	"github.com/jackc/pgx/v5/pgxpool"
)

type State struct {
	chats    chatStore
	sessions sessiondomain.Repository
	setup    setupUnitOfWork
}

type setupUnitOfWork interface {
	Do(context.Context, func(chatStore, sessiondomain.Repository) error) error
}

func NewPostgresState(ctx context.Context, pool *pgxpool.Pool, sessionConfig sessiondomain.PostgresRepositoryConfig) (*State, error) {
	if pool == nil {
		return nil, fmt.Errorf("consoleapi/chats: postgres pool is nil")
	}
	chatStore, err := NewPostgresChatStore(ctx, pool)
	if err != nil {
		return nil, err
	}
	sessionRepo, err := sessiondomain.NewPostgresRepository(ctx, pool, sessionConfig)
	if err != nil {
		return nil, err
	}
	return &State{
		chats:    chatStore,
		sessions: sessionRepo,
		setup:    postgresSetupUnitOfWork{pool: pool, chats: chatStore, sessions: sessionRepo},
	}, nil
}

func newDirectState(chats chatStore, sessions sessiondomain.Repository) *State {
	return &State{
		chats:    chats,
		sessions: sessions,
		setup:    directSetupUnitOfWork{chats: chats, sessions: sessions},
	}
}

type directSetupUnitOfWork struct {
	chats    chatStore
	sessions sessiondomain.Repository
}

func (u directSetupUnitOfWork) Do(ctx context.Context, fn func(chatStore, sessiondomain.Repository) error) error {
	return fn(u.chats, u.sessions)
}

type postgresSetupUnitOfWork struct {
	pool     *pgxpool.Pool
	chats    *PostgresChatStore
	sessions *sessiondomain.PostgresRepository
}

func (u postgresSetupUnitOfWork) Do(ctx context.Context, fn func(chatStore, sessiondomain.Repository) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(u.chats.withTx(tx), u.sessions.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
