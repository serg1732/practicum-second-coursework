package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/serg1732/practicum-second-coursework/internal/models"
)

// UserRepo - репозиторий по работе с таблицей UserRepo.
type UserRepo struct {
	db *DataBase
}

// BuildUserRepo - создание репозитория по работе с таблицей UserRepo.
func BuildUserRepo(db *DataBase) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

// Registration - регистрация пользователя.
func (u *UserRepo) Registration(ctx context.Context, user *models.UserRequest) (*models.User, error) {
	registeredUser := &models.User{}

	const sqlText = `
		INSERT INTO users (
			username,
			password,
			created_at
		)
		VALUES ($1, $2, $3)
		RETURNING user_id, username
	`

	if err := u.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			user.Username,
			user.Password,
			time.Now(),
		).Scan(&registeredUser.ID, &registeredUser.Username)
	}); err != nil {
		return nil, err
	}
	return registeredUser, nil
}

// Authentication - авторизация пользователя.
func (u *UserRepo) Authentication(ctx context.Context, userRequest *models.UserRequest) (*models.User, error) {
	authenticatedUser := &models.User{}

	const sqlText = `
		SELECT user_id, username
		FROM users
		WHERE username = $1
			AND password = $2
	`

	if err := u.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			userRequest.Username,
			userRequest.Password,
		).Scan(
			&authenticatedUser.ID,
			&authenticatedUser.Username,
		)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWrongUsernameOrPassword
		}
		return nil, err
	}
	return authenticatedUser, nil
}

// UserExists - проверяет, существует ли пользователь с username.
func (u *UserRepo) UserExists(ctx context.Context, username string) (bool, error) {
	const sqlText = `
		SELECT TRUE
			FROM users AS u
			WHERE u.username = $1
		LIMIT 1
	`

	var exists bool
	if err := u.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, sqlText, username).Scan(&exists)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}
