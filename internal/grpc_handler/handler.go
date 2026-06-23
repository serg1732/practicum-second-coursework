package grpc_handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/models"
	grpc "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DatabaseRepository interface {
	Ping(ctx context.Context) error
}

type UserRepository interface {
	Authentication(ctx context.Context, req *models.UserRequest) (*models.User, error)
	UserExists(ctx context.Context, username string) (bool, error)
	Registration(ctx context.Context, req *models.UserRequest) (*models.User, error)
}

type TokenRepository interface {
	Create(ctx context.Context, userID int64, exp time.Duration) (*models.Token, error)
	GetExpToken(ctx context.Context, accessToken string) (time.Time, error)
	Validate(exp time.Time) bool
}

type FileRepository interface {
	FileExists(ctx context.Context, req *models.FileRequest) (bool, error)
	GetListFile(ctx context.Context, userID int64) ([]models.File, error)
	DeleteFile(ctx context.Context, req *models.FileRequest) (int64, error)
	UploadFile(ctx context.Context, req *models.FileRequest) (*models.File, error)
}

type EntityRepository interface {
	GetList(ctx context.Context, userID int64, entityType string) ([]models.Entity, error)
	Exists(ctx context.Context, req *models.CreateEntityRequest) (bool, error)
	CreateEntity(ctx context.Context, req *models.CreateEntityRequest) (int64, error)
	RemoveEntity(ctx context.Context, userID int64, name string, entityType string) (int64, error)
	UpdateEntity(ctx context.Context, userID int64, name string, entityType string, data []byte) (int64, error)
}

type FileManagerRepository interface {
	CreateStorageUser(userID int64) error
	UploadFile(userID int64, name string, data []byte) error
	DownloadFile(userID int64, name string) ([]byte, error)
	RemoveFile(userID int64, name string) error
}

// Handler - обработчик запросов GRPC сервера.
type Handler struct {
	log         *slog.Logger
	database    DatabaseRepository
	config      *config.GophKeeperServerConfig
	user        UserRepository
	file        FileRepository
	entity      EntityRepository
	token       TokenRepository
	fileManager FileManagerRepository
	grpc.UnimplementedGophKeeperServer
}

// BuildGRPCHandler - создание обработчика GRPC сервера.
func BuildGRPCHandler(
	logger *slog.Logger,
	db *repository.DataBase,
	config *config.GophKeeperServerConfig,
	userRepository *repository.UserRepo,
	binaryRepository *repository.FilesRepo,
	entityRepository *repository.EntityRepo,
	tokenRepository *repository.TokenRepo) *Handler {
	return &Handler{log: logger, database: db, config: config, user: userRepository, file: binaryRepository,
		entity: entityRepository, token: tokenRepository, fileManager: repository.BuildFileManager(config.LocalFileStoragePath)}
}

// Ping - обработчик проверки соединения.
func (h *Handler) Ping(ctx context.Context, req *grpc.PingRequest) (*grpc.PingResponse, error) {
	h.log.Debug("Получен запрос на проверку соединения")
	var msg string
	err := h.database.Ping(ctx)
	if err != nil {
		msg = "NOT CONNECTED"
		h.log.Error("ошибка при проверке соединения", "error", err)
		return grpc.PingResponse_builder{Message: msg}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	msg = "CONNECTED"
	h.log.Debug("Успешная проверка соединения", "message", req.String())
	return grpc.PingResponse_builder{Message: msg}.Build(), nil
}
