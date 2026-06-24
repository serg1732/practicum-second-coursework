package grpc_handler

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/serg1732/practicum-second-coursework/internal/models"
	grpc "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func newTestToken(userID int64) *models.Token {
	now := time.Now()
	return &models.Token{
		AccessToken: "created-token",
		UserID:      userID,
		Iat:         now,
		Exp:         now.Add(time.Hour),
	}
}

func TestAuthenticationSuccess(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().Authentication(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *models.UserRequest) (*models.User, error) {
			assert.Equal(t, "user", req.Username)
			assert.Equal(t, "password", req.Password)
			return &models.User{}, nil
		},
	)
	deps.token.EXPECT().Create(gomock.Any(), gomock.Any(), time.Second*time.Duration(h.config.TokenExpired)).Return(newTestToken(777), nil)

	resp, err := h.Authentication(context.Background(), grpc.AuthenticationRequest_builder{
		Username: "user",
		Password: "password",
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, int64(777), resp.GetAccessToken().GetUserId())
	assert.Equal(t, "created-token", resp.GetAccessToken().GetToken())
}

func TestAuthenticationError(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().Authentication(gomock.Any(), gomock.Any()).Return(nil, errTest)

	_, err := h.Authentication(context.Background(), grpc.AuthenticationRequest_builder{
		Username: "user",
		Password: "bad-password",
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestAuthenticationErrorCreateToken(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().Authentication(gomock.Any(), gomock.Any()).Return(&models.User{}, nil)
	deps.token.EXPECT().Create(gomock.Any(), gomock.Any(), time.Second*time.Duration(h.config.TokenExpired)).Return(nil, errTest)

	_, err := h.Authentication(context.Background(), grpc.AuthenticationRequest_builder{
		Username: "user",
		Password: "password",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestUserExistErrorUserExist(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(true, nil)

	resp, err := h.UserExist(context.Background(), grpc.UserExistRequest_builder{Username: "user"}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, true, resp.GetExist())
}

func TestUserExistErrorUserNotExist(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(false, nil)

	resp, err := h.UserExist(context.Background(), grpc.UserExistRequest_builder{Username: "user"}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, false, resp.GetExist())
}

func TestUserExistError(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(false, errTest)

	resp, err := h.UserExist(context.Background(), grpc.UserExistRequest_builder{Username: "user"}.Build())

	assertErrorCode(t, codes.Internal, err)
	assert.Equal(t, false, resp.GetExist())
}

func TestRegistrationSuccess(t *testing.T) {
	h, deps := newTestHandler(t)

	gomock.InOrder(
		deps.user.EXPECT().UserExists(gomock.Any(), "new-user").Return(false, nil),
		deps.user.EXPECT().Registration(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, req *models.UserRequest) (*models.User, error) {
				assert.Equal(t, "new-user", req.Username)
				assert.Equal(t, "password", req.Password)
				return &models.User{}, nil
			},
		),
		deps.token.EXPECT().Create(gomock.Any(), gomock.Any(), time.Duration(h.config.TokenExpired)).Return(newTestToken(777), nil),
		deps.fileManager.EXPECT().CreateStorageUser(int64(777)).Return(nil),
	)

	resp, err := h.Registration(context.Background(), grpc.RegistrationRequest_builder{
		Username: "new-user",
		Password: "password",
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, int64(777), resp.GetAccessToken().GetUserId())
	assert.Equal(t, "created-token", resp.GetAccessToken().GetToken())
}

func TestRegistrationUserExist(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(true, nil)

	_, err := h.Registration(context.Background(), grpc.RegistrationRequest_builder{
		Username: "user",
		Password: "password",
	}.Build())

	assertErrorCode(t, codes.AlreadyExists, err)
}

func TestRegistrationErrorCheckExist(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(false, errTest)

	_, err := h.Registration(context.Background(), grpc.RegistrationRequest_builder{
		Username: "user",
		Password: "password",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestRegistrationError(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(false, nil)
	deps.user.EXPECT().Registration(gomock.Any(), gomock.Any()).Return(nil, errTest)

	_, err := h.Registration(context.Background(), grpc.RegistrationRequest_builder{
		Username: "user",
		Password: "password",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestRegistrationErrorCreateToken(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(false, nil)
	deps.user.EXPECT().Registration(gomock.Any(), gomock.Any()).Return(&models.User{}, nil)
	deps.token.EXPECT().Create(gomock.Any(), gomock.Any(), time.Duration(h.config.TokenExpired)).Return(nil, errTest)

	_, err := h.Registration(context.Background(), grpc.RegistrationRequest_builder{
		Username: "user",
		Password: "password",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestRegistrationErrorCreateStorage(t *testing.T) {
	h, deps := newTestHandler(t)

	deps.user.EXPECT().UserExists(gomock.Any(), "user").Return(false, nil)
	deps.user.EXPECT().Registration(gomock.Any(), gomock.Any()).Return(&models.User{}, nil)
	deps.token.EXPECT().Create(gomock.Any(), gomock.Any(), time.Duration(h.config.TokenExpired)).Return(newTestToken(777), nil)
	deps.fileManager.EXPECT().CreateStorageUser(int64(777)).Return(errTest)

	_, err := h.Registration(context.Background(), grpc.RegistrationRequest_builder{
		Username: "user",
		Password: "password",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}
