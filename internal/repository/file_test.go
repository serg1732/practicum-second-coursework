package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"

	"github.com/serg1732/practicum-second-coursework/internal/models"
)

func TestBuildFileRepo(t *testing.T) {
	t.Parallel()

	db := &DataBase{}

	repo := BuildFileRepo(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestFilesRepoUploadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     *models.FileRequest
		prepareMock func(pgxmock.PgxPoolIface)
		wantFile    *models.File
		wantErr     error
	}{
		{
			name: "успешно загружает файл",
			request: &models.FileRequest{
				UserID: 1,
				Name:   "backup.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("INSERT INTO file").
					WithArgs(int64(1), "backup.zip", pgxmock.AnyArg()).
					WillReturnRows(pgxmock.NewRows([]string{"file_id", "name"}).
						AddRow(int64(10), "backup.zip"))
				mock.ExpectCommit()
			},
			wantFile: &models.File{
				ID:   10,
				Name: "backup.zip",
			},
		},
		{
			name: "возвращает ошибку при ошибке вставки",
			request: &models.FileRequest{
				UserID: 2,
				Name:   "error.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("INSERT INTO file").
					WithArgs(int64(2), "error.zip", pgxmock.AnyArg()).
					WillReturnError(errFileRepoTest)
				mock.ExpectRollback()
			},
			wantErr: errFileRepoTest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, mock := newFilesRepoMock(t)
			defer mock.Close()

			tt.prepareMock(mock)

			gotFile, err := repo.UploadFile(context.Background(), tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, gotFile)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantFile, gotFile)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFilesRepoGetListFile(t *testing.T) {
	t.Parallel()

	createdAtFirst := time.Date(2026, time.January, 10, 11, 12, 13, 0, time.UTC)
	createdAtSecond := time.Date(2026, time.January, 11, 12, 13, 14, 0, time.UTC)

	tests := []struct {
		name        string
		userID      int64
		prepareMock func(pgxmock.PgxPoolIface)
		wantFiles   []models.File
		wantErr     bool
	}{
		{
			name:   "возвращает список файлов пользователя",
			userID: 1,
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"file_id", "user_id", "name", "created_at"}).
					AddRow(int64(10), int64(1), "first.zip", createdAtFirst).
					AddRow(int64(11), int64(1), "second.zip", createdAtSecond)

				mock.ExpectBegin()
				mock.ExpectQuery("SELECT").
					WithArgs(int64(1)).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
			wantFiles: []models.File{
				{ID: 10, UserID: 1, Name: "first.zip", CreatedAt: createdAtFirst},
				{ID: 11, UserID: 1, Name: "second.zip", CreatedAt: createdAtSecond},
			},
		},
		{
			name:   "возвращает пустой список если файлов нет",
			userID: 2,
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT").
					WithArgs(int64(2)).
					WillReturnRows(pgxmock.NewRows([]string{"file_id", "user_id", "name", "created_at"}))
				mock.ExpectCommit()
			},
			wantFiles: []models.File{},
		},
		{
			name:   "возвращает ошибку запроса",
			userID: 3,
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT").
					WithArgs(int64(3)).
					WillReturnError(errFileRepoTest)
				mock.ExpectRollback()
			},
			wantErr: true,
		},
		{
			name:   "возвращает ошибку сканирования строки",
			userID: 4,
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"file_id", "user_id", "name", "created_at"}).
					AddRow("bad-id", int64(4), "broken.zip", createdAtFirst)

				mock.ExpectBegin()
				mock.ExpectQuery("SELECT").
					WithArgs(int64(4)).
					WillReturnRows(rows)
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, mock := newFilesRepoMock(t)
			defer mock.Close()

			tt.prepareMock(mock)

			gotFiles, err := repo.GetListFile(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, gotFiles)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantFiles, gotFiles)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFilesRepoFileExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     *models.FileRequest
		prepareMock func(pgxmock.PgxPoolIface)
		wantExists  bool
		wantErr     error
	}{
		{
			name: "возвращает true если файл существует",
			request: &models.FileRequest{
				UserID: 1,
				Name:   "backup.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(int64(1), "backup.zip").
					WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectCommit()
			},
			wantExists: true,
		},
		{
			name: "возвращает false если файла нет",
			request: &models.FileRequest{
				UserID: 1,
				Name:   "missing.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(int64(1), "missing.zip").
					WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
				mock.ExpectCommit()
			},
		},
		{
			name: "возвращает ошибку запроса",
			request: &models.FileRequest{
				UserID: 2,
				Name:   "error.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(int64(2), "error.zip").
					WillReturnError(errFileRepoTest)
				mock.ExpectRollback()
			},
			wantErr: errFileRepoTest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, mock := newFilesRepoMock(t)
			defer mock.Close()

			tt.prepareMock(mock)

			gotExists, err := repo.FileExists(context.Background(), tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.False(t, gotExists)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, gotExists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFilesRepoDeleteFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     *models.FileRequest
		prepareMock func(pgxmock.PgxPoolIface)
		wantID      int64
		wantErr     error
	}{
		{
			name: "успешно удаляет файл",
			request: &models.FileRequest{
				UserID: 1,
				Name:   "backup.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("DELETE FROM file").
					WithArgs(int64(1), "backup.zip").
					WillReturnRows(pgxmock.NewRows([]string{"file_id"}).AddRow(int64(10)))
				mock.ExpectCommit()
			},
			wantID: 10,
		},
		{
			name: "возвращает ErrRecordNotFound если файл не найден",
			request: &models.FileRequest{
				UserID: 1,
				Name:   "missing.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("DELETE FROM file").
					WithArgs(int64(1), "missing.zip").
					WillReturnRows(pgxmock.NewRows([]string{"file_id"}))
				mock.ExpectRollback()
			},
			wantErr: ErrRecordNotFound,
		},
		{
			name: "возвращает ошибку удаления",
			request: &models.FileRequest{
				UserID: 2,
				Name:   "error.zip",
			},
			prepareMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectQuery("DELETE FROM file").
					WithArgs(int64(2), "error.zip").
					WillReturnError(errFileRepoTest)
				mock.ExpectRollback()
			},
			wantErr: errFileRepoTest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, mock := newFilesRepoMock(t)
			defer mock.Close()

			tt.prepareMock(mock)

			gotID, err := repo.DeleteFile(context.Background(), tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, int64(0), gotID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, gotID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

var errFileRepoTest = errors.New("test error")

func newFilesRepoMock(t *testing.T) (*FilesRepo, pgxmock.PgxPoolIface) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if !assert.NoError(t, err) {
		return nil, nil
	}

	return BuildFileRepo(&DataBase{database: mock}), mock
}
