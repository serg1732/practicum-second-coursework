package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/serg1732/practicum-second-coursework/internal/models"
)

// EntityRepo - репозиторий по работе с сущностями (текст, креды, карта).
type EntityRepo struct {
	db *DataBase
}

// BuildEntityRepo - создание репозитория по работе с сущностями.
func BuildEntityRepo(db *DataBase) *EntityRepo {
	return &EntityRepo{
		db: db,
	}
}

// CreateEntity - сохранение данных на сервере.
func (e *EntityRepo) CreateEntity(ctx context.Context, entityRequest *models.CreateEntityRequest) (int64, error) {
	var id int64

	metadata := models.MetadataEntity{
		Name:        entityRequest.Metadata.Name,
		Description: entityRequest.Metadata.Description,
		Type:        entityRequest.Metadata.Type,
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return 0, err
	}

	const sqlText = `
		INSERT INTO entity (
			user_id,
			data,
			metadata,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING entity_id
	`

	if err = e.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			entityRequest.UserID,
			entityRequest.Data,
			jsonMetadata,
			time.Now(),
			time.Now(),
		).Scan(&id)
	}); err != nil {
		return 0, err
	}

	return id, nil
}

// GetList - получение списка сущностей для пользователя.
func (e *EntityRepo) GetList(ctx context.Context, userID int64, typeEntity string) ([]models.Entity, error) {
	entities := make([]models.Entity, 0)

	const sqlText = `
		SELECT 
			entity_id,
			user_id,
			data,
			metadata,
			created_at,
			updated_at
		FROM entity
		WHERE user_id = $1
			AND metadata->>'Type' = $2
	`

	err := e.db.retry(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sqlText, userID, typeEntity)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			entity := models.Entity{}
			entityMetadata := models.MetadataEntity{}

			var jsonEntity []byte

			if err = rows.Scan(
				&entity.ID,
				&entity.UserID,
				&entity.Data,
				&jsonEntity,
				&entity.CreatedAt,
				&entity.UpdatedAt,
			); err != nil {
				return err
			}

			if err = json.Unmarshal(jsonEntity, &entityMetadata); err != nil {
				return err
			}

			entity.Metadata = entityMetadata
			entities = append(entities, entity)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return entities, nil
}

// Exists - проверка на существование сущности.
func (e *EntityRepo) Exists(ctx context.Context, entityRequest *models.CreateEntityRequest) (bool, error) {
	var exists bool

	const sqlText = `
		SELECT TRUE
		FROM entity AS e
		WHERE e.user_id = $1
			AND e.metadata->>'Name' = $2
			AND e.metadata->>'Type' = $3
		LIMIT 1
	`

	if err := e.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			entityRequest.UserID,
			entityRequest.Metadata.Name,
			entityRequest.Metadata.Type,
		).Scan(&exists)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}

// RemoveEntity - удаление сущности с сервера.
func (e *EntityRepo) RemoveEntity(ctx context.Context, userID int64, name string, typeEntity string) (int64, error) {
	var id int64

	const sqlText = `
		DELETE FROM entity AS e
		WHERE e.user_id = $1
			AND e.metadata->>'Name' = $2
			AND e.metadata->>'Type' = $3
		RETURNING e.entity_id
	`

	if err := e.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			userID,
			name,
			typeEntity,
		).Scan(&id)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrRecordNotFound
		}
		return 0, err
	}
	return id, nil
}

// UpdateEntity - обновление данных сущности.
func (e *EntityRepo) UpdateEntity(ctx context.Context, userID int64, name string, typeEntity string, data []byte) (int64, error) {
	var id int64

	const sqlText = `
		UPDATE entity
		SET 
			data = $1,
			updated_at = $2
		WHERE entity.user_id = $3
			AND entity.metadata->>'Name' = $4
			AND entity.metadata->>'Type' = $5
		RETURNING entity_id
	`

	if err := e.db.retry(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			sqlText,
			data,
			time.Now(),
			userID,
			name,
			typeEntity,
		).Scan(&id)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrRecordNotFound
		}
		return 0, err
	}
	return id, nil
}
