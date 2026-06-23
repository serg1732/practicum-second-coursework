package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/logger"
	"github.com/serg1732/practicum-second-coursework/internal/repository"
	grpcserver "github.com/serg1732/practicum-second-coursework/internal/service/grpc/server"
	"google.golang.org/grpc/credentials"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	log := logger.NewSlogLogger(slog.LevelInfo)
	log.Info("Запуск сервера", "version", buildVersion, "commit", buildCommit, "date", buildDate)

	serverConfig, err := config.GetServerConfig()
	if err != nil {
		log.Error("Ошибка при парсинге конфига", "error", err)
		os.Exit(1)
	}

	logLevel, errLevel := logger.ParseLevel(serverConfig.LogLevel)
	if errLevel != nil {
		log.Error("Ошибка парсинга уровня логирования", "error", errLevel)
		os.Exit(1)
	}

	log = logger.NewSlogLogger(logLevel)
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT,
	)
	defer stop()

	if errMigrate := repository.MigrateDataBase(log, serverConfig.DSN); errMigrate != nil {
		log.Error("Ошибка при миграции данных", "error", errMigrate)
		os.Exit(1)
	}

	creds, errCreds := credentials.NewServerTLSFromFile(serverConfig.TLSCertPath, serverConfig.TLSKeyPath)
	if errCreds != nil {
		log.Error("ошибка при инициализации TLS", "error", errCreds)
		os.Exit(1)
	}

	grpcServer, err := grpcserver.BuildGRPCServer(ctx, log, serverConfig)
	if err != nil {
		log.Error("ошибка при инициализации GRPC сервера")
		os.Exit(1)
	}

	if errGrpc := grpcServer.Run(ctx, creds); errGrpc != nil {
		log.Error("Ошибка при работе GRPC сервера", "error", errGrpc)
		os.Exit(1)
	}
}
