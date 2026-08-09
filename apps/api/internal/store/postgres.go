package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/agungardiyanta/UrlShorter/apps/api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("link not found")

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	store := &PostgresStore{pool: pool}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) CreateLink(ctx context.Context, code, targetURL string, expiresAt time.Time) (domain.Link, error) {
	query := `
		insert into links (code, target_url, expires_at)
		values ($1, $2, $3)
		returning code, target_url, clicks, expires_at, deleted_at, created_at, updated_at
	`
	var link domain.Link
	err := scanLink(s.pool.QueryRow(ctx, query, code, targetURL, expiresAt), &link)
	return link, err
}

func (s *PostgresStore) GetLink(ctx context.Context, code string) (domain.Link, error) {
	query := `
		select code, target_url, clicks, expires_at, deleted_at, created_at, updated_at
		from links
		where (code = $1 or lower(code) = lower($1))
			and deleted_at is null
			and (expires_at is null or expires_at > now())
		order by code = $1 desc
		limit 1
	`
	var link domain.Link
	err := scanLink(s.pool.QueryRow(ctx, query, code), &link)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Link{}, ErrNotFound
	}
	return link, err
}

func (s *PostgresStore) SoftDeleteLink(ctx context.Context, code string) error {
	commandTag, err := s.pool.Exec(ctx, `
		update links
		set deleted_at = now(),
			expires_at = least(coalesce(expires_at, now()), now()),
			updated_at = now()
		where (code = $1 or lower(code) = lower($1))
			and deleted_at is null
	`, code)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) IncrementClicks(ctx context.Context, code string) error {
	commandTag, err := s.pool.Exec(ctx, `
		update links
		set clicks = clicks + 1, updated_at = now()
		where code = $1
			and deleted_at is null
			and (expires_at is null or expires_at > now())
	`, code)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListLinks(ctx context.Context, limit int) ([]domain.Link, error) {
	rows, err := s.pool.Query(ctx, `
		select code, target_url, clicks, expires_at, deleted_at, created_at, updated_at
		from links
		where deleted_at is null
			and (expires_at is null or expires_at > now())
		order by created_at desc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]domain.Link, 0)
	for rows.Next() {
		var link domain.Link
		if err := scanLink(rows, &link); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		create table if not exists links (
			code text primary key,
			target_url text not null,
			clicks bigint not null default 0,
			expires_at timestamptz,
			deleted_at timestamptz,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		);
		alter table links add column if not exists expires_at timestamptz;
		alter table links add column if not exists deleted_at timestamptz;
		update links set expires_at = created_at + interval '7 days' where expires_at is null;
		create index if not exists links_created_at_idx on links (created_at desc);
		create index if not exists links_expires_at_idx on links (expires_at);
		create index if not exists links_deleted_at_idx on links (deleted_at);
	`)
	if err != nil {
		return fmt.Errorf("migrate postgres: %w", err)
	}
	return nil
}

type linkScanner interface {
	Scan(dest ...any) error
}

func scanLink(scanner linkScanner, link *domain.Link) error {
	var expiresAt sql.NullTime
	var deletedAt sql.NullTime
	err := scanner.Scan(
		&link.Code,
		&link.TargetURL,
		&link.Clicks,
		&expiresAt,
		&deletedAt,
		&link.CreatedAt,
		&link.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if expiresAt.Valid {
		link.ExpiresAt = &expiresAt.Time
	}
	if deletedAt.Valid {
		link.DeletedAt = &deletedAt.Time
	}
	link.Expired = link.IsExpired(time.Now().UTC())
	return nil
}
