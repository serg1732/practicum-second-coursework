package grpc_handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/grpc_handler/mocks"
	grpc "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testAccessToken = "access-token"
	testUserID      = int64(42)
)

var errTest = errors.New("test error")

type testDeps struct {
	database    *mocks.MockDatabaseRepository
	user        *mocks.MockUserRepository
	token       *mocks.MockTokenRepository
	file        *mocks.MockFileRepository
	entity      *mocks.MockEntityRepository
	fileManager *mocks.MockFileManagerRepository
}

func newTestHandler(t *testing.T) (*Handler, *testDeps) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	deps := &testDeps{
		database:    mocks.NewMockDatabaseRepository(ctrl),
		user:        mocks.NewMockUserRepository(ctrl),
		token:       mocks.NewMockTokenRepository(ctrl),
		file:        mocks.NewMockFileRepository(ctrl),
		entity:      mocks.NewMockEntityRepository(ctrl),
		fileManager: mocks.NewMockFileManagerRepository(ctrl),
	}

	h := &Handler{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		config: &config.GophKeeperServerConfig{
			TokenExpired:         60,
			LocalFileStoragePath: "test-storage",
		},
		database:    deps.database,
		user:        deps.user,
		token:       deps.token,
		file:        deps.file,
		entity:      deps.entity,
		fileManager: deps.fileManager,
	}

	return h, deps
}

func newAccessToken() *grpc.Token {
	return grpc.Token_builder{
		Token:  testAccessToken,
		UserId: testUserID,
	}.Build()
}

func expectValidToken(deps *testDeps) {
	exp := time.Now().Add(time.Hour)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(exp, nil)
	deps.token.EXPECT().Validate(exp).Return(true)
}

func expectInvalidToken(deps *testDeps) {
	exp := time.Now().Add(-time.Hour)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(exp, nil)
	deps.token.EXPECT().Validate(exp).Return(false)
}

func assertErrorCode(t *testing.T, want codes.Code, err error) bool {
	t.Helper()
	got := status.Code(err)
	if got != want {
		t.Errorf("ожидался grpc-код: %s, получен: %s, ошибка: %v", want, got, err)
		return false
	}
	return true
}

func TestPingSuccess(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.database.EXPECT().Ping(gomock.Any()).Return(nil)

	resp, err := h.Ping(context.Background(), grpc.PingRequest_builder{}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "CONNECTED", resp.GetMessage())
}

func TestPingError(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.database.EXPECT().Ping(gomock.Any()).Return(errTest)

	resp, err := h.Ping(context.Background(), grpc.PingRequest_builder{}.Build())

	assertErrorCode(t, codes.Internal, err)
	assert.Equal(t, "NOT CONNECTED", resp.GetMessage())
}

func zeroTime() time.Time {
	return time.Time{}
}
