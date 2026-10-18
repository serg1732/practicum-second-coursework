package server

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestBuildGRPCServerReturnErrorWhenDSNEmpty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.GophKeeperServerConfig{}

	gotServer, gotErr := BuildGRPCServer(context.Background(), logger, cfg)

	assert.Nil(t, gotServer)
	assert.Error(t, gotErr)
}

func TestRunReturnErrorWhenGRPCAddressInvalid(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := &Server{
		config: &config.GophKeeperServerConfig{
			GRPCRunAddr: "invalid-address",
		},
		log: logger,
	}

	gotErr := srv.Run(context.Background(), insecure.NewCredentials())

	assert.Error(t, gotErr)
	assert.NotNil(t, gotErr)
	assert.True(t, strings.Contains(gotErr.Error(), "missing port in address"), "ожидалась ошибка из-за некорректного адреса")
}

func TestShutdownGracefullyStopGRPCServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := &Server{log: logger}
	grpcServer := grpc.NewServer()

	done := make(chan struct{})

	go func() {
		srv.Shutdown(grpcServer)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Errorf("Shutdown не завершился за ожидаемое время")
	}
}
