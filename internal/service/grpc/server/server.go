package server

import (
	"context"
	"log/slog"
	"net"
	"sync"

	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/grpc_handler"
	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/repository"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server - описание сервера.
type Server struct {
	config  *config.GophKeeperServerConfig
	storage *repository.DataBase
	wg      sync.WaitGroup
	log     *slog.Logger
}

// BuildGRPCServer - создание GRPC сервера.
func BuildGRPCServer(ctx context.Context, logger *slog.Logger, config *config.GophKeeperServerConfig) (*Server, error) {
	db, err := repository.BuildDataBase(ctx, slog.Default(), config)
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
	handlerGrpc := grpc_handler.BuildGRPCHandler(s.log, s.storage, s.config, userRepository, binaryRepository, entityRepository, tokenRepository)
	pb.RegisterGophKeeperServer(grpcServer, handlerGrpc)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		s.log.Info("Запуск сервера по адресу", "address", s.config.GRPCRunAddr)
		return grpcServer.Serve(lis)
	})

	s.wg.Add(1)
	go (func() {
		<-gCtx.Done()
		s.Shutdown(grpcServer)
	})()
	if err = g.Wait(); err != nil {
		s.log.Error("Получена ошибка при работе GRPC сервера")
	}
	s.wg.Wait()
	return nil
}

// Shutdown - завершение работы GRPC сервера.
func (s *Server) Shutdown(srvGRPC *grpc.Server) {
	defer s.wg.Done()
	srvGRPC.GracefulStop()
	s.log.Info("Успешное завершение работы GRPC сервера")
}
