package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/helpers"
)

// MigrateDataBase миграция данных до последней версии.
func MigrateDataBase(log *slog.Logger, databaseDSN string) error {
	if databaseDSN == "" {
		return errors.New("DSL required")
	}
	m, err := migrate.New(
		"file://migrations",
		databaseDSN,
	)
	if err != nil {
		return fmt.Errorf("ошибка при попытке найти миграции: %w", err)
	}

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("Миграции не требуются")
			return nil
		}

		return fmt.Errorf("ошибка при попытке применить миграцию: %w", err)
	}

	log.Info("Миграции успешно применены")
	return nil
}

type Pool interface {
	Ping(ctx context.Context) error
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
	Close()
}

// DataBase База данных Postgresql.
type DataBase struct {
	database Pool
}

// BuildDataBase инициализация подключения к БД.
func BuildDataBase(ctx context.Context, log *slog.Logger, config *config.GophKeeperServerConfig) (*DataBase, error) {
	if config.DSN == "" {
		return nil, errors.New("DSL required")
	}
	db, err := pgxpool.New(ctx, config.DSN)
	if err != nil {
		log.Error("Error opening database connection", "error", err)
		return nil, err
	}
	if errPing := db.Ping(ctx); errPing != nil {
		log.Error("Error pinging database connection", "error", errPing)
		return nil, errPing
	}

	log.Info("Successfully connected to database")
	return &DataBase{database: db}, nil
}

// Ping проверка подключения к БД.
func (db *DataBase) Ping(ctx context.Context) error {
	return db.database.Ping(ctx)
}

// retry функционал попытки отправить повторно запрос, если ошибка из списка Retriable.
func (db *DataBase) retry(ctx context.Context, fn func(pgx.Tx) error) error {
	const maxRetries = 3
	delay := 1
	classify := helpers.NewPostgresErrorClassifier()
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		tx, err := db.database.BeginTx(ctx, pgx.TxOptions{})
		defer tx.Rollback(ctx)
		if err != nil && classify.Classify(err) != helpers.Retriable {
			return err
		}
		errFn := fn(tx)
		if errFn == nil {
			if errCommit := tx.Commit(ctx); errCommit == nil {
				return nil
			} else {
				if classify == nil || classify.Classify(errCommit) != helpers.Retriable {
					return errCommit
				}
				lastErr = errCommit
			}
		} else {
			if classify == nil || classify.Classify(errFn) != helpers.Retriable {
				return errFn
			}
			lastErr = errFn
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(delay) * time.Second):
		}
		delay += 2
	}

	return fmt.Errorf("превышено число попыток запросов %s", lastErr)
}
