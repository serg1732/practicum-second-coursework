package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServerConfigDefaultValues(t *testing.T) {
	defer resetFlagsAndArgs(t)()

	cfg, err := GetServerConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, ":8081", cfg.GRPCRunAddr)
	assert.Equal(t, "", cfg.DSN)
	assert.Equal(t, "", cfg.TLSCertPath)
	assert.Equal(t, "", cfg.TLSKeyPath)
	assert.Equal(t, "/tmp/", cfg.LocalFileStoragePath)
	assert.Equal(t, int64(600), cfg.TokenExpired)
	assert.Equal(t, "INFO", cfg.LogLevel)
}

func TestGetServerConfigFlags(t *testing.T) {
	defer resetFlagsAndArgs(
		t,
		"-g", "localhost:9001",
		"-d", "postgres://user:password@localhost:5432/db?sslmode=disable",
		"-tc", "/tmp/server.crt",
		"-tk", "/tmp/server.key",
		"-lp", "/tmp/files/",
		"-ll", "info",
		"-e", "600",
	)()

	cfg, err := GetServerConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "localhost:9001", cfg.GRPCRunAddr)
	assert.Equal(t, "postgres://user:password@localhost:5432/db?sslmode=disable", cfg.DSN)
	assert.Equal(t, "/tmp/server.crt", cfg.TLSCertPath)
	assert.Equal(t, "/tmp/server.key", cfg.TLSKeyPath)
	assert.Equal(t, "/tmp/files/", cfg.LocalFileStoragePath)
	assert.Equal(t, int64(600), cfg.TokenExpired)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestGetServerConfigEnvOverridesDefaults(t *testing.T) {
	defer resetFlagsAndArgs(t)()

	t.Setenv("GRPC_ADDRESS", "env-host:9002")
	t.Setenv("DATABASE_DSN", "postgres://env:password@localhost:5432/env_db?sslmode=disable")
	t.Setenv("CRYPTO_KEY", "/env/server-private.pem")
	t.Setenv("TLS_CERT_SERVER_PATH", "/env/server.crt")
	t.Setenv("TLS_KEY_SERVER_PATH", "/env/server.key")
	t.Setenv("LOCAL_FILE_STORAGE_PATH", "/env/files/")
	t.Setenv("TOKEN_EXP", "900")
	t.Setenv("LOG_LEVEL", "warn")

	cfg, err := GetServerConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "env-host:9002", cfg.GRPCRunAddr)
	assert.Equal(t, "postgres://env:password@localhost:5432/env_db?sslmode=disable", cfg.DSN)
	assert.Equal(t, "/env/server.crt", cfg.TLSCertPath)
	assert.Equal(t, "/env/server.key", cfg.TLSKeyPath)
	assert.Equal(t, "/env/files/", cfg.LocalFileStoragePath)
	assert.Equal(t, int64(900), cfg.TokenExpired)
	assert.Equal(t, "warn", cfg.LogLevel)
}

func TestGetServerConfigEnvOverridesFlags(t *testing.T) {
	defer resetFlagsAndArgs(
		t,
		"-g", "flag-host:9001",
		"-d", "postgres://flag:password@localhost:5432/flag_db?sslmode=disable",
		"-crypto-key", "/flag/server-private.pem",
		"-tc", "/flag/server.crt",
		"-tk", "/flag/server.key",
		"-lp", "/flag/files/",
		"-ll", "error",
		"-e", "600",
	)()

	t.Setenv("GRPC_ADDRESS", "env-host:9002")
	t.Setenv("DATABASE_DSN", "postgres://env:password@localhost:5432/env_db?sslmode=disable")
	t.Setenv("CRYPTO_KEY", "/env/server-private.pem")
	t.Setenv("TLS_CERT_SERVER_PATH", "/env/server.crt")
	t.Setenv("TLS_KEY_SERVER_PATH", "/env/server.key")
	t.Setenv("LOCAL_FILE_STORAGE_PATH", "/env/files/")
	t.Setenv("TOKEN_EXP", "900")
	t.Setenv("LOG_LEVEL", "warn")

	cfg, err := GetServerConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "env-host:9002", cfg.GRPCRunAddr)
	assert.Equal(t, "postgres://env:password@localhost:5432/env_db?sslmode=disable", cfg.DSN)
	assert.Equal(t, "/env/server.crt", cfg.TLSCertPath)
	assert.Equal(t, "/env/server.key", cfg.TLSKeyPath)
	assert.Equal(t, "/env/files/", cfg.LocalFileStoragePath)
	assert.Equal(t, int64(900), cfg.TokenExpired)
	assert.Equal(t, "warn", cfg.LogLevel)
}

func TestGetServerConfigInvalidTokenExpiredEnv(t *testing.T) {
	defer resetFlagsAndArgs(t)()

	t.Setenv("TOKEN_EXP", "not-int")

	cfg, err := GetServerConfig()

	assert.Error(t, err)
	if cfg != nil {
		t.Errorf("ожидалось, что конфиг будет nil при ошибке парсинга env, получено: %+v", cfg)
	}
}
