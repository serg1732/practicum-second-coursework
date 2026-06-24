package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/caarlos0/env/v11"
	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/logger"
	"github.com/serg1732/practicum-second-coursework/internal/service/grpc/client"
	"github.com/serg1732/practicum-second-coursework/internal/tui"
	"google.golang.org/grpc/credentials"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	log := logger.NewSlogLogger(slog.LevelInfo)
	clientConfig := config.GetClientConfig()

	flag.Parse()
	if err := env.Parse(clientConfig); err != nil {
		log.Error("Ошибка при парсинге env значений конфига", "error", err)
		os.Exit(1)
	}

	level, errLevel := logger.ParseLevel(clientConfig.LogLevel)
	if errLevel != nil {
		log.Error("ошибка при чтении уровня логирования из конфига", "error", errLevel)
		os.Exit(1)
	}

	log = logger.NewSlogLogger(level)
	creds, errCreds := credentials.NewClientTLSFromFile(clientConfig.TLSCertPath, "")
	if errCreds != nil {
		log.Error("ошибка при инициализации TLS", "error", errCreds)
		os.Exit(1)
	}
	client, err := client.BuildGRPCClient(log, clientConfig, creds)
	if err != nil {
		log.Error("ошибка при инициализации GRPC", "error", err)
		os.Exit(1)
	}

	adapter := tui.BuildGRPCAdapter(client, clientConfig.LocalStoragePath)
	ctx := context.Background()
	p := tea.NewProgram(
		tui.NewModel(ctx, adapter, buildVersion, buildCommit, buildDate),
	)
	if _, errTui := p.Run(); errTui != nil {
		log.Error("ошибка при работе TUI", "error", errTui)
		os.Exit(1)
	}
}
