package grpc_handler

import (
	"context"
	"time"

	"github.com/serg1732/practicum-second-coursework/internal/models"
	grpc "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/repository"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Authentication - обработчик авторизации пользователя.
func (h *Handler) Authentication(ctx context.Context, req *grpc.AuthenticationRequest) (*grpc.AuthenticationResponse, error) {
	h.log.Debug("Запрос на авторизацию пользователя", "username", req.GetUsername())
	UserData := &models.UserRequest{
		Username: req.GetUsername(),
		Password: req.GetPassword(),
	}

	authenticatedUser, err := h.user.Authentication(ctx, UserData)
	if err != nil {
		h.log.Error("ошибка при получении данных пользователя", "error", err)
		return grpc.AuthenticationResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, err.Error(),
		)
	}

	user := models.GetUserData(authenticatedUser)
	token, err := h.token.Create(ctx, user.GetUserId(), time.Second*time.Duration(h.config.TokenExpired))
	if err != nil {
		h.log.Error("ошибка при записи токена в БД", "error", err)
		return grpc.AuthenticationResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)
	return grpc.AuthenticationResponse_builder{
		AccessToken: grpc.Token_builder{
			Token:  token.AccessToken,
			UserId: token.UserID,
			Iat:    createdToken,
			Exp:    expToken,
		}.Build(),
	}.Build(), nil
}

// UserExist - обработчик проверки на существование пользователя.
func (h *Handler) UserExist(ctx context.Context, req *grpc.UserExistRequest) (*grpc.UserExistResponse, error) {
	h.log.Debug("Запрос на проверку существования пользователя в БД")
	exist, err := h.user.UserExists(ctx, req.GetUsername())
	if err != nil {
		h.log.Error("ошибка при проверке существования пользователя в БД", "error", err)
		return grpc.UserExistResponse_builder{Exist: false}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.UserExistResponse_builder{Exist: exist}.Build(), nil
}

// Registration - обработчик регистрации пользователя.
func (h *Handler) Registration(ctx context.Context, req *grpc.RegistrationRequest) (*grpc.RegistrationResponse, error) {
	h.log.Debug("Запрос на регистрацию нового пользователя")
	UserData := &models.UserRequest{}
	UserData.Username = req.GetUsername()
	UserData.Password = req.GetPassword()

	exists, err := h.user.UserExists(ctx, UserData.Username)
	if err != nil {
		h.log.Error("ошибка при проверке существования пользователя в БД", "error", err)
		return grpc.RegistrationResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	if exists {
		h.log.Error("ошибка при регистрации! Пользователя с таким username существует")
		return grpc.RegistrationResponse_builder{}.Build(), status.Error(
			codes.AlreadyExists, repository.ErrUsernameAlreadyExists.Error(),
		)
	}
	registeredUser, err := h.user.Registration(ctx, UserData)
	if err != nil {
		h.log.Error("ошибка при регистрации пользователя", "error", err)
		return grpc.RegistrationResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}

	user := models.GetUserData(registeredUser)
	token, err := h.token.Create(ctx, user.GetUserId(), time.Duration(h.config.TokenExpired))
	if err != nil {
		h.log.Error("ошибка при создании токена после регистрации", "error", err)
		return grpc.RegistrationResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)
	if err = h.fileManager.CreateStorageUser(token.UserID); err != nil {
		h.log.Error("ошибка при создании локального хранилища", "error", err)
		return grpc.RegistrationResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.RegistrationResponse_builder{
		AccessToken: grpc.Token_builder{
			Token:  token.AccessToken,
			UserId: token.UserID,
			Iat:    createdToken,
			Exp:    expToken,
		}.Build(),
	}.Build(), nil
}
