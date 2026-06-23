package repository

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMigrateDataBase(t *testing.T) {
	t.Run("возвращает ошибку, если DSN пустой", func(t *testing.T) {
		log := slog.New(slog.NewTextHandler(io.Discard, nil))

		err := MigrateDataBase(log, "")

		assert.Error(t, err)
		if err != nil {
			assert.Equal(t, "DSL required", err.Error())
		}
	})
}

func TestBuildDataBase(t *testing.T) {
	t.Run("возвращает ошибку, если DSN пустой", func(t *testing.T) {
		log := slog.New(slog.NewTextHandler(io.Discard, nil))

		db, err := BuildDataBase(context.Background(), log, &config.GophKeeperServerConfig{})

		assert.Error(t, err)
		assert.Equal(t, (*DataBase)(nil), db)
		if err != nil {
			assert.Equal(t, "DSL required", err.Error())
		}
	})
}

func TestDataBasePing(t *testing.T) {
	t.Run("вызывает Ping у пула и возвращает nil", func(t *testing.T) {
		pool := &mockedPool{}
		db := &DataBase{database: pool}

		err := db.Ping(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 1, pool.pingCalls)
	})

	t.Run("возвращает ошибку Ping", func(t *testing.T) {
		expectedErr := errors.New("ошибка ping")
		pool := &mockedPool{pingErr: expectedErr}
		db := &DataBase{database: pool}

		err := db.Ping(context.Background())

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, pool.pingCalls)
	})
}

func TestDataBaseRetry(t *testing.T) {
	t.Run("коммитит транзакцию, если функция выполнилась без ошибки", func(t *testing.T) {
		tx := &mockedTx{}
		pool := &mockedPool{tx: tx}
		db := &DataBase{database: pool}
		calls := 0

		err := db.retry(context.Background(), func(tx pgx.Tx) error {
			calls++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
		assert.Equal(t, 1, pool.beginCalls)
		assert.Equal(t, 1, tx.commitCalls)
		assert.Equal(t, 1, tx.rollbackCalls)
	})

	t.Run("возвращает ошибку функции и не коммитит транзакцию", func(t *testing.T) {
		expectedErr := errors.New("ошибка запроса")
		tx := &mockedTx{}
		pool := &mockedPool{tx: tx}
		db := &DataBase{database: pool}

		err := db.retry(context.Background(), func(tx pgx.Tx) error {
			return expectedErr
		})

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, pool.beginCalls)
		assert.Equal(t, 0, tx.commitCalls)
		assert.Equal(t, 1, tx.rollbackCalls)
	})

	t.Run("возвращает ошибку Commit", func(t *testing.T) {
		expectedErr := errors.New("ошибка commit")
		tx := &mockedTx{commitErr: expectedErr}
		pool := &mockedPool{tx: tx}
		db := &DataBase{database: pool}

		err := db.retry(context.Background(), func(tx pgx.Tx) error {
			return nil
		})

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, pool.beginCalls)
		assert.Equal(t, 1, tx.commitCalls)
		assert.Equal(t, 1, tx.rollbackCalls)
	})

	t.Run("возвращает ошибку BeginTx и не вызывает функцию", func(t *testing.T) {
		expectedErr := errors.New("ошибка begin")
		tx := &mockedTx{}
		pool := &mockedPool{tx: tx, beginErr: expectedErr}
		db := &DataBase{database: pool}
		calls := 0

		err := db.retry(context.Background(), func(tx pgx.Tx) error {
			calls++
			return nil
		})

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 0, calls)
		assert.Equal(t, 1, pool.beginCalls)
		assert.Equal(t, 0, tx.commitCalls)
		assert.Equal(t, 1, tx.rollbackCalls)
	})
}

type mockedPool struct {
	pingErr  error
	beginErr error
	tx       pgx.Tx

	pingCalls  int
	beginCalls int
	closeCalls int
}

func (p *mockedPool) Ping(ctx context.Context) error {
	p.pingCalls++
	return p.pingErr
}

func (p *mockedPool) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	p.beginCalls++
	if p.tx == nil {
		p.tx = &mockedTx{}
	}
	return p.tx, p.beginErr
}

func (p *mockedPool) Close() {
	p.closeCalls++
}

type mockedTx struct {
	commitErr   error
	rollbackErr error

	commitCalls   int
	rollbackCalls int
	beginCalls    int
}

func (tx *mockedTx) Begin(ctx context.Context) (pgx.Tx, error) {
	tx.beginCalls++
	return tx, nil
}

func (tx *mockedTx) Commit(ctx context.Context) error {
	tx.commitCalls++
	return tx.commitErr
}

func (tx *mockedTx) Rollback(ctx context.Context) error {
	tx.rollbackCalls++
	return tx.rollbackErr
}

func (tx *mockedTx) CopyFrom(
	ctx context.Context,
	tableName pgx.Identifier,
	columnNames []string,
	rowSrc pgx.CopyFromSource,
) (int64, error) {
	return 0, nil
}

func (tx *mockedTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *mockedTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *mockedTx) Prepare(ctx context.Context, name string, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (tx *mockedTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (tx *mockedTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (tx *mockedTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (tx *mockedTx) Conn() *pgx.Conn {
	return nil
}
