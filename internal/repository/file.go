package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/serg1732/practicum-second-coursework/internal/models"
)

// FilesRepo - репозиторий по работе с файлами.
type FilesRepo struct {
	db *DataBase
}

// BuildFileRepo - создание репозитория по работе с файлами.
func BuildFileRepo(db *DataBase) *FilesRepo {
	return &FilesRepo{
		db: db,
	}
}

// UploadFile - получение данных о файле из БД.
func (f *FilesRepo) UploadFile(ctx context.Context, binaryRequest *models.FileRequest) (*models.File, error) {
	binary := &models.File{}

	const sqlText = `
		INSERT INTO file (
			user_id,
			name,
			created_at
		)
		VALUES ($1, $2, $3)
		RETURNING file_id, name
	`

	err := f.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			binaryRequest.UserID,
			binaryRequest.Name,
			time.Now(),
		).Scan(&binary.ID, &binary.Name)
	})
	if err != nil {
		return nil, err
	}

	return binary, nil
}

// GetListFile - получение списка файла пользователя на сервере.
func (f *FilesRepo) GetListFile(ctx context.Context, userID int64) ([]models.File, error) {
	listFile := make([]models.File, 0)

	const sqlText = `
		SELECT 
			file_id,
			user_id,
			name,
			created_at
		FROM file
		WHERE user_id = $1
	`

	err := f.db.retry(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sqlText, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			binary := models.File{}

			if err = rows.Scan(
				&binary.ID,
				&binary.UserID,
				&binary.Name,
				&binary.CreatedAt,
			); err != nil {
				return err
			}

			listFile = append(listFile, binary)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return listFile, nil
}

// FileExists проверка на наличие файла на сервере.
func (f *FilesRepo) FileExists(ctx context.Context, binaryRequest *models.FileRequest) (bool, error) {
	var exists bool

	const sqlText = `
		SELECT EXISTS(
			SELECT 1
			FROM file
			WHERE file.user_id = $1
				AND file.name = $2
		)
	`

	err := f.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			binaryRequest.UserID,
			binaryRequest.Name,
		).Scan(&exists)
	})
	if err != nil {
		return false, err
	}

	return exists, nil
}

// DeleteFile - удаление файла с сервера.
func (f *FilesRepo) DeleteFile(ctx context.Context, binaryRequest *models.FileRequest) (int64, error) {
	var id int64

	const sqlText = `
	DELETE FROM file AS f
	WHERE f.user_id = $1
		AND f.name = $2
	RETURNING f.file_id
`

	err := f.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			binaryRequest.UserID,
			binaryRequest.Name,
		).Scan(&id)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrRecordNotFound
		}

		return 0, err
	}
	return id, nil
}
