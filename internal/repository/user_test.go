package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"

	"github.com/serg1732/practicum-second-coursework/internal/models"
)

const (
	registrationSQLPattern   = `INSERT\s+INTO\s+users\s*\(\s*username,\s*password,\s*created_at\s*\)\s*VALUES\s*\(\$1,\s*\$2,\s*\$3\s*\)\s*RETURNING\s+user_id,\s+username`
	authenticationSQLPattern = `SELECT\s+user_id,\s+username\s+FROM\s+users\s+WHERE\s+username\s+=\s+\$1\s+AND\s+password\s+=\s+\$2`
	userExistsSQLPattern     = `SELECT\s+TRUE\s+FROM\s+users\s+AS\s+u\s+WHERE\s+u\.username\s+=\s+\$1\s+LIMIT\s+1`
)

func buildUserRepoMock(t *testing.T) (*UserRepo, pgxmock.PgxPoolIface) {
	t.Helper()

	mockPool, err := pgxmock.NewPool()
	if !assert.NoError(t, err) {
		return nil, nil
	}

	return BuildUserRepo(&DataBase{database: mockPool}), mockPool
}

func TestUserRepoRegistrationSuccess(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	userRequest := &models.UserRequest{
		Username: "ivan",
		Password: "password_hash",
	}

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(registrationSQLPattern).
		WithArgs(userRequest.Username, userRequest.Password, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "username"}).AddRow(1, userRequest.Username))
	mockPool.ExpectCommit()

	actual, err := repo.Registration(context.Background(), userRequest)

	assert.NoError(t, err)
	assert.Equal(t, &models.User{ID: 1, Username: userRequest.Username}, actual)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserRepoRegistrationErrorDB(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	expectedErr := errors.New("insert user error")
	userRequest := &models.UserRequest{
		Username: "ivan",
		Password: "password_hash",
	}

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(registrationSQLPattern).
		WithArgs(userRequest.Username, userRequest.Password, pgxmock.AnyArg()).
		WillReturnError(expectedErr)
	mockPool.ExpectRollback()

	actual, err := repo.Registration(context.Background(), userRequest)

	assert.Nil(t, actual)
	assert.ErrorIs(t, err, expectedErr)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestPositiveUserRepoAuthentication(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	userRequest := &models.UserRequest{
		Username: "ivan",
		Password: "password_hash",
	}

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(authenticationSQLPattern).
		WithArgs(userRequest.Username, userRequest.Password).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "username"}).AddRow(1, userRequest.Username))
	mockPool.ExpectCommit()

	actual, err := repo.Authentication(context.Background(), userRequest)

	assert.NoError(t, err)
	assert.Equal(t, &models.User{ID: 1, Username: userRequest.Username}, actual)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestNegativeUserAuthRepoWrongPassword(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	userRequest := &models.UserRequest{
		Username: "ivan",
		Password: "wrong_password",
	}

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(authenticationSQLPattern).
		WithArgs(userRequest.Username, userRequest.Password).
		WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectRollback()

	actual, err := repo.Authentication(context.Background(), userRequest)

	assert.Nil(t, actual)
	assert.ErrorIs(t, err, ErrWrongUsernameOrPassword)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestNegativeUserAuthRepoErrorDB(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	expectedErr := errors.New("select user error")
	userRequest := &models.UserRequest{
		Username: "ivan",
		Password: "password_hash",
	}

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(authenticationSQLPattern).
		WithArgs(userRequest.Username, userRequest.Password).
		WillReturnError(expectedErr)
	mockPool.ExpectRollback()

	actual, err := repo.Authentication(context.Background(), userRequest)

	assert.Nil(t, actual)
	assert.ErrorIs(t, err, expectedErr)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestPositiveUserRepoUserExists(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	username := "ivan"

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(userExistsSQLPattern).
		WithArgs(username).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mockPool.ExpectCommit()

	actual, err := repo.UserExists(context.Background(), username)

	assert.NoError(t, err)
	assert.True(t, actual)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestNegativeUserRepoNotUserExists(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	username := "unknown"

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(userExistsSQLPattern).
		WithArgs(username).
		WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectRollback()

	actual, err := repo.UserExists(context.Background(), username)

	assert.NoError(t, err)
	assert.False(t, actual)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestNegativeUserRepoErrorDB(t *testing.T) {
	t.Parallel()

	repo, mockPool := buildUserRepoMock(t)
	if repo == nil {
		return
	}
	defer mockPool.Close()

	expectedErr := errors.New("check user exists error")
	username := "ivan"

	mockPool.ExpectBegin()
	mockPool.ExpectQuery(userExistsSQLPattern).
		WithArgs(username).
		WillReturnError(expectedErr)
	mockPool.ExpectRollback()

	actual, err := repo.UserExists(context.Background(), username)

	assert.False(t, actual)
	assert.ErrorIs(t, err, expectedErr)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}
