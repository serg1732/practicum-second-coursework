package repository

import (
	"context"
	"encoding/hex"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	t.Run("генерирует hex строку нужной длины", func(t *testing.T) {
		token, err := generateToken(lengthToken)

		assert.NoError(t, err)
		assert.Equal(t, lengthToken*2, len(token))

		_, err = hex.DecodeString(token)
		assert.NoError(t, err)
	})

	t.Run("для нулевой длины возвращает пустой токен", func(t *testing.T) {
		token, err := generateToken(0)

		assert.NoError(t, err)
		assert.Equal(t, "", token)
	})
}

func TestTokenRepoValidate(t *testing.T) {
	repo := &TokenRepo{}

	t.Run("возвращает true если срок действия еще не истек", func(t *testing.T) {
		expTime := time.Now().Add(time.Hour)

		assert.True(t, repo.Validate(expTime))
	})

	t.Run("возвращает false если срок действия истек", func(t *testing.T) {
		expTime := time.Now().Add(-time.Hour)

		assert.False(t, repo.Validate(expTime))
	})
}

func TestTokenRepoCreate(t *testing.T) {
	const userID int64 = 10

	sqlText := regexp.QuoteMeta(`
		INSERT INTO access_token (
			access_token,
			user_id,
			iat,
			exp
		)
		VALUES ($1, $2, $3, $4)
		RETURNING access_token, user_id, iat, exp
	`)

	t.Run("создает токен", func(t *testing.T) {
		ctx := context.Background()
		mock := newPgxMock(t)
		if mock == nil {
			return
		}
		defer mock.Close()

		iat := time.Now().UTC().Truncate(time.Second)
		exp := iat.Add(time.Hour)

		mock.ExpectBegin()
		mock.ExpectQuery(sqlText).
			WithArgs(pgxmock.AnyArg(), userID, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(
				pgxmock.NewRows([]string{"access_token", "user_id", "iat", "exp"}).
					AddRow("access-token", userID, iat, exp),
			)
		mock.ExpectCommit()

		repo := BuildTokenRepo(&DataBase{database: mock})

		token, err := repo.Create(ctx, userID, time.Hour)

		assert.NoError(t, err)
		if err != nil {
			return
		}
		assert.Equal(t, "access-token", token.AccessToken)
		assert.Equal(t, userID, token.UserID)
		assert.Equal(t, iat, token.Iat)
		assert.Equal(t, exp, token.Exp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("возвращает ошибку если запрос в БД завершился ошибкой", func(t *testing.T) {
		ctx := context.Background()
		mock := newPgxMock(t)
		if mock == nil {
			return
		}
		defer mock.Close()

		expectedErr := errors.New("insert token error")

		mock.ExpectBegin()
		mock.ExpectQuery(sqlText).
			WithArgs(pgxmock.AnyArg(), userID, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(expectedErr)
		mock.ExpectRollback()

		repo := BuildTokenRepo(&DataBase{database: mock})

		token, err := repo.Create(ctx, userID, time.Hour)

		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, token)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTokenRepoGetExpToken(t *testing.T) {
	const accessToken = "access-token"

	sqlText := regexp.QuoteMeta(`
		SELECT exp
		FROM access_token
		WHERE access_token = $1
	`)

	t.Run("возвращает время окончания действия токена", func(t *testing.T) {
		ctx := context.Background()
		mock := newPgxMock(t)
		if mock == nil {
			return
		}
		defer mock.Close()

		expectedExp := time.Now().UTC().Add(time.Hour).Truncate(time.Second)

		mock.ExpectBegin()
		mock.ExpectQuery(sqlText).
			WithArgs(accessToken).
			WillReturnRows(
				pgxmock.NewRows([]string{"exp"}).
					AddRow(expectedExp),
			)
		mock.ExpectCommit()

		repo := BuildTokenRepo(&DataBase{database: mock})

		actualExp, err := repo.GetExpToken(ctx, accessToken)

		assert.NoError(t, err)
		assert.Equal(t, expectedExp, actualExp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("возвращает ErrRecordNotFound если токен не найден", func(t *testing.T) {
		ctx := context.Background()
		mock := newPgxMock(t)
		if mock == nil {
			return
		}
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectQuery(sqlText).
			WithArgs(accessToken).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectRollback()

		repo := BuildTokenRepo(&DataBase{database: mock})

		_, err := repo.GetExpToken(ctx, accessToken)

		assert.ErrorIs(t, err, ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("возвращает ошибку БД", func(t *testing.T) {
		ctx := context.Background()
		mock := newPgxMock(t)
		if mock == nil {
			return
		}
		defer mock.Close()

		expectedErr := errors.New("select token error")

		mock.ExpectBegin()
		mock.ExpectQuery(sqlText).
			WithArgs(accessToken).
			WillReturnError(expectedErr)
		mock.ExpectRollback()

		repo := BuildTokenRepo(&DataBase{database: mock})

		_, err := repo.GetExpToken(ctx, accessToken)

		assert.ErrorIs(t, err, expectedErr)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func newPgxMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()

	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	if err != nil {
		return nil
	}
	return mock
}
