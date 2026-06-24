package server

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/grpc_handler"
	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server - описание сервера.
type Server struct {
	config  *config.GophKeeperServerConfig
	storage *repository.DataBase
	log     *slog.Logger
}

// BuildGRPCServer - создание GRPC сервера.
func BuildGRPCServer(ctx context.Context, logger *slog.Logger, config *config.GophKeeperServerConfig) (*Server, error) {
	db, err := repository.BuildDataBase(ctx, logger, config)
	if err != nil {
		return nil, err
	}
	return &Server{config: config, storage: db, log: logger}, nil
}

// Run - запуск GRPC сервера.
func (s *Server) Run(ctx context.Context, creds credentials.TransportCredentials) error {
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	lis, err := net.Listen("tcp", s.config.GRPCRunAddr)
	if err != nil {
		s.log.Error("Ошибка при запуске GRPC сервера", "error", err)
		return err
	}
	userRepository := repository.BuildUserRepo(s.storage)
	binaryRepository := repository.BuildFileRepo(s.storage)
	entityRepository := repository.BuildEntityRepo(s.storage)
	tokenRepository := repository.BuildTokenRepo(s.storage)
	handlerGrpc := grpc_handler.BuildGRPCHandler(s.log, s.config, s.storage, userRepository, binaryRepository, entityRepository, tokenRepository)
	pb.RegisterGophKeeperServer(grpcServer, handlerGrpc)

	errChannel := make(chan error, 1)
	go func() {
		s.log.Info("Запуск сервера по адресу", "address", s.config.GRPCRunAddr)
		errChannel <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		s.log.Info("Остановка GRPC сервера")
		s.Shutdown(grpcServer)
		if errServer := <-errChannel; errServer != nil && !errors.Is(errServer, grpc.ErrServerStopped) {
			s.log.Error("Ошибка при остановке GRPC сервера", "error", errServer)
			return errServer
		}
	case errServer := <-errChannel:
		if errServer != nil && !errors.Is(errServer, grpc.ErrServerStopped) {
			s.log.Error("Получена ошибка от GRPC сервера", "error", errServer)
			return errServer
		}
	}
	return nil
}

// Shutdown - завершение работы GRPC сервера.
func (s *Server) Shutdown(srvGRPC *grpc.Server) {
	srvGRPC.GracefulStop()
	s.log.Info("Успешное завершение работы GRPC сервера")
}
