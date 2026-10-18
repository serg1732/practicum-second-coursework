package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/serg1732/practicum-second-coursework/internal/models"
)

const lengthToken = 32

// TokenRepo - репозиторий по работе с токенами.
type TokenRepo struct {
	db *DataBase
}

// BuildTokenRepo - создание репозитория по работе с токенами.
func BuildTokenRepo(db *DataBase) *TokenRepo {
	return &TokenRepo{
		db: db,
	}
}

// Create - Запись токена в БД.
func (t *TokenRepo) Create(ctx context.Context, userID int64, exp time.Duration) (*models.Token, error) {
	token := &models.Token{}

	accessToken, err := generateToken(lengthToken)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()

	const sqlText = `
		INSERT INTO access_token (
			access_token,
			user_id,
			iat,
			exp
		)
		VALUES ($1, $2, $3, $4)
		RETURNING access_token, user_id, iat, exp
	`

	err = t.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			accessToken,
			userID,
			currentTime,
			currentTime.Add(exp),
		).Scan(
			&token.AccessToken,
			&token.UserID,
			&token.Iat,
			&token.Exp,
		)
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

// Validate - проверка даты истечения срока действия токена.
func (t *TokenRepo) Validate(expTime time.Time) bool {
	return time.Now().Before(expTime)
}

// GetExpToken - получение даты истечения срока действия токена.
func (t *TokenRepo) GetExpToken(ctx context.Context, accessToken string) (time.Time, error) {
	var end time.Time

	const sqlText = `
		SELECT exp
		FROM access_token
		WHERE access_token = $1
	`

	if err := t.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, sqlText, accessToken).Scan(&end)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return end, ErrRecordNotFound
		}
		return end, err
	}
	return end, nil
}

// generateToken - генерация нового токена.
func generateToken(length int) (string, error) {
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
