package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/alonsoF100/reporting-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Repository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:   pool,
		logger: slog.With("component", "postgres_repository"),
	}
}

func NewPool(cfg *config.Config) (*pgxpool.Pool, error) {
	const op = "postgres.NewPool"

	logger := slog.With(
		slog.String("op", op),
		slog.String("host", cfg.Database.Host),
		slog.Int("port", cfg.Database.Port),
		slog.String("db", cfg.Database.Name),
	)

	logger.Info("initializing database connection pool")

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.ConStr())
	if err != nil {
		logger.Error("failed to parse pgx pool config",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Настройки пула
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Error("failed to create pgx pool",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Проверяем соединение
	err = pool.Ping(context.Background())
	if err != nil {
		logger.Error("failed to ping database",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("database connection established")

	// Миграции - только если директория указана и существует
	if cfg.Migration.Dir != "" {
		if _, err := os.Stat(cfg.Migration.Dir); err == nil {
			logger.Info("running database migrations",
				slog.String("migrations_dir", cfg.Migration.Dir))

			connConfig := poolConfig.ConnConfig
			db := stdlib.OpenDB(*connConfig)

			if err := goose.Up(db, cfg.Migration.Dir); err != nil {
				logger.Error("failed to run migrations",
					slog.String("error", err.Error()),
					slog.String("migrations_dir", cfg.Migration.Dir),
				)
				return nil, fmt.Errorf("%s: migrations failed: %w", op, err)
			}

			logger.Info("migrations completed successfully",
				slog.String("migrations_dir", cfg.Migration.Dir))
		} else {
			logger.Warn("migrations directory does not exist, skipping",
				slog.String("migrations_dir", cfg.Migration.Dir),
				slog.String("error", err.Error()),
			)
		}
	} else {
		logger.Info("migrations directory not specified, skipping")
	}

	return pool, nil
}

// Close закрывает пул соединений
func (r *Repository) Close() {
	r.logger.Info("closing database connection pool")
	r.pool.Close()
}

// Ping проверяет соединение с БД
func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
