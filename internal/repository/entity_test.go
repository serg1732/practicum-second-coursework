package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"

	"github.com/serg1732/practicum-second-coursework/internal/models"
)

const (
	createEntitySQL = `(?s)INSERT INTO entity.*user_id.*data.*metadata.*created_at.*updated_at.*RETURNING entity_id`
	getListSQL      = `(?s)SELECT.*entity_id.*user_id.*data.*metadata.*created_at.*updated_at.*FROM entity.*WHERE user_id = \$1.*metadata->>'Type' = \$2`
	existsSQL       = `(?s)SELECT TRUE.*FROM entity AS e.*WHERE e.user_id = \$1.*e.metadata->>'Name' = \$2.*e.metadata->>'Type' = \$3.*LIMIT 1`
	removeEntitySQL = `(?s)DELETE FROM entity AS e.*WHERE e.user_id = \$1.*e.metadata->>'Name' = \$2.*e.metadata->>'Type' = \$3.*RETURNING e.entity_id`
	updateEntitySQL = `(?s)UPDATE entity.*SET.*data = \$1.*updated_at = \$2.*WHERE entity.user_id = \$3.*entity.metadata->>'Name' = \$4.*entity.metadata->>'Type' = \$5.*RETURNING entity_id`
)

func TestBuildEntityRepo(t *testing.T) {
	db := &DataBase{}

	repo := BuildEntityRepo(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestEntityRepoCreateEntity(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock pgxmock.PgxPoolIface, request *models.CreateEntityRequest)
		wantID    int64
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "успешное создание сущности",
			setupMock: func(mock pgxmock.PgxPoolIface, request *models.CreateEntityRequest) {
				mock.ExpectBegin()
				mock.ExpectQuery(createEntitySQL).
					WithArgs(
						request.UserID,
						request.Data,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnRows(pgxmock.NewRows([]string{"entity_id"}).AddRow(int64(10)))
				mock.ExpectCommit()
			},
			wantID: 10,
		},
		{
			name: "ошибка при insert",
			setupMock: func(mock pgxmock.PgxPoolIface, request *models.CreateEntityRequest) {
				expectedErr := errors.New("insert error")

				mock.ExpectBegin()
				mock.ExpectQuery(createEntitySQL).
					WithArgs(
						request.UserID,
						request.Data,
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(expectedErr)
				mock.ExpectRollback()
			},
			wantErr:   true,
			wantErrIs: errors.New("insert error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, closeMock := newEntityRepo(t)
			defer closeMock()

			request := buildCreateEntityRequest()
			tt.setupMock(mock, request)

			gotID, err := repo.CreateEntity(context.Background(), request)

			if tt.wantErrIs != nil {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.wantID, gotID)
		})
	}
}

func TestEntityRepoGetList(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock pgxmock.PgxPoolIface)
		want      []models.Entity
		wantErr   bool
	}{
		{
			name: "успешное получение списка сущностей",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				createdAt := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2026, 6, 23, 11, 0, 0, 0, time.UTC)

				rows := pgxmock.NewRows([]string{
					"entity_id",
					"user_id",
					"data",
					"metadata",
					"created_at",
					"updated_at",
				}).AddRow(
					int64(1),
					int64(100),
					[]byte("encrypted-data"),
					[]byte(`{"Name":"card","Description":"bank card","Type":"card"}`),
					createdAt,
					updatedAt,
				)

				mock.ExpectBegin()
				mock.ExpectQuery(getListSQL).
					WithArgs(int64(100), "card").
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
			want: []models.Entity{
				{
					ID:        1,
					UserID:    100,
					Data:      []byte("encrypted-data"),
					Metadata:  models.MetadataEntity{Name: "card", Description: "bank card", Type: "card"},
					CreatedAt: time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, 6, 23, 11, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "ошибка запроса списка сущностей",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery(getListSQL).
					WithArgs(int64(100), "card").
					WillReturnError(errors.New("select error"))
				mock.ExpectRollback()
			},
			wantErr: true,
		},
		{
			name: "ошибка при разборе metadata",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"entity_id",
					"user_id",
					"data",
					"metadata",
					"created_at",
					"updated_at",
				}).AddRow(
					int64(1),
					int64(100),
					[]byte("encrypted-data"),
					[]byte(`{bad json`),
					time.Now(),
					time.Now(),
				)

				mock.ExpectBegin()
				mock.ExpectQuery(getListSQL).
					WithArgs(int64(100), "card").
					WillReturnRows(rows)
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, closeMock := newEntityRepo(t)
			defer closeMock()

			tt.setupMock(mock)

			got, err := repo.GetList(context.Background(), 100, "card")

			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Equal(t, []models.Entity(nil), got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEntityRepoExists(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock pgxmock.PgxPoolIface, request *models.CreateEntityRequest)
		want      bool
		wantErr   bool
	}{
		{
			name: "сущность существует",
			setupMock: func(mock pgxmock.PgxPoolIface, request *models.CreateEntityRequest) {
				mock.ExpectBegin()
				mock.ExpectQuery(existsSQL).
					WithArgs(request.UserID, request.Metadata.Name, request.Metadata.Type).
					WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectCommit()
			},
			want: true,
		},
		{
			name: "сущность не найдена",
			setupMock: func(mock pgxmock.PgxPoolIface, request *models.CreateEntityRequest) {
				mock.ExpectBegin()
				mock.ExpectQuery(existsSQL).
					WithArgs(request.UserID, request.Metadata.Name, request.Metadata.Type).
					WillReturnError(pgx.ErrNoRows)
				mock.ExpectRollback()
			},
			want: false,
		},
		{
			name: "ошибка проверки существования",
			setupMock: func(mock pgxmock.PgxPoolIface, request *models.CreateEntityRequest) {
				mock.ExpectBegin()
				mock.ExpectQuery(existsSQL).
					WithArgs(request.UserID, request.Metadata.Name, request.Metadata.Type).
					WillReturnError(errors.New("exists error"))
				mock.ExpectRollback()
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, closeMock := newEntityRepo(t)
			defer closeMock()

			request := buildCreateEntityRequest()
			tt.setupMock(mock, request)

			got, err := repo.Exists(context.Background(), request)

			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEntityRepoRemoveEntity(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock pgxmock.PgxPoolIface)
		wantID    int64
		wantErrIs error
	}{
		{
			name: "успешное удаление сущности",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery(removeEntitySQL).
					WithArgs(int64(100), "name", "text").
					WillReturnRows(pgxmock.NewRows([]string{"entity_id"}).AddRow(int64(7)))
				mock.ExpectCommit()
			},
			wantID: 7,
		},
		{
			name: "сущность для удаления не найдена",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery(removeEntitySQL).
					WithArgs(int64(100), "name", "text").
					WillReturnError(pgx.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErrIs: ErrRecordNotFound,
		},
		{
			name: "ошибка удаления сущности",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery(removeEntitySQL).
					WithArgs(int64(100), "name", "text").
					WillReturnError(errors.New("delete error"))
				mock.ExpectRollback()
			},
			wantErrIs: errors.New("delete error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, closeMock := newEntityRepo(t)
			defer closeMock()

			tt.setupMock(mock)

			gotID, err := repo.RemoveEntity(context.Background(), 100, "name", "text")

			if tt.wantErrIs != nil {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.wantID, gotID)
		})
	}
}

func TestEntityRepoUpdateEntity(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock pgxmock.PgxPoolIface)
		wantID    int64
		wantErrIs error
	}{
		{
			name: "успешное обновление сущности",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery(updateEntitySQL).
					WithArgs([]byte("new-data"), pgxmock.AnyArg(), int64(100), "name", "text").
					WillReturnRows(pgxmock.NewRows([]string{"entity_id"}).AddRow(int64(9)))
				mock.ExpectCommit()
			},
			wantID: 9,
		},
		{
			name: "сущность для обновления не найдена",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery(updateEntitySQL).
					WithArgs([]byte("new-data"), pgxmock.AnyArg(), int64(100), "name", "text").
					WillReturnError(pgx.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErrIs: ErrRecordNotFound,
		},
		{
			name: "ошибка обновления сущности",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery(updateEntitySQL).
					WithArgs([]byte("new-data"), pgxmock.AnyArg(), int64(100), "name", "text").
					WillReturnError(errors.New("update error"))
				mock.ExpectRollback()
			},
			wantErrIs: errors.New("update error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, closeMock := newEntityRepo(t)
			defer closeMock()

			tt.setupMock(mock)

			gotID, err := repo.UpdateEntity(context.Background(), 100, "name", "text", []byte("new-data"))

			if tt.wantErrIs != nil {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.wantID, gotID)
		})
	}
}

func buildCreateEntityRequest() *models.CreateEntityRequest {
	return &models.CreateEntityRequest{
		UserID: 100,
		Data:   []byte("encrypted-data"),
		Metadata: models.MetadataEntity{
			Name:        "name",
			Description: "description",
			Type:        "text",
		},
	}
}

func newEntityRepo(t *testing.T) (*EntityRepo, pgxmock.PgxPoolIface, func()) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool(): %v", err)
	}

	repo := BuildEntityRepo(&DataBase{database: mock})

	return repo, mock, func() {
		t.Helper()

		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания pgxmock были выполнены: %v", err)
		}
		mock.Close()
	}
}
