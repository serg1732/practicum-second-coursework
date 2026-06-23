package config

import (
	"flag"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetFlagsAndArgs(t *testing.T, args ...string) func() {
	t.Helper()

	oldCommandLine := flag.CommandLine
	oldArgs := os.Args

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = append([]string{"test"}, args...)

	return func() {
		flag.CommandLine = oldCommandLine
		os.Args = oldArgs
	}
}

func TestGetClientConfigDefaultValues(t *testing.T) {
	defer resetFlagsAndArgs(t)()

	cfg, err := GetClientConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "localhost:8081", cfg.GRPCRemoteAddr)
	assert.Equal(t, "", cfg.TLSCertPath)
	assert.Equal(t, "/tmp/client/", cfg.LocalStoragePath)
	assert.Equal(t, "INFO", cfg.LogLevel)
}

func TestGetClientConfigFlags(t *testing.T) {
	defer resetFlagsAndArgs(
		t,
		"-g", "localhost:9001",
		"-tc", "/tmp/server.crt",
		"-ll", "debug",
	)()

	cfg, err := GetClientConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "localhost:9001", cfg.GRPCRemoteAddr)
	assert.Equal(t, "/tmp/server.crt", cfg.TLSCertPath)
	assert.Equal(t, "/tmp/client/", cfg.LocalStoragePath)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestGetClientConfigEnvOverridesDefaults(t *testing.T) {
	defer resetFlagsAndArgs(t)()

	t.Setenv("GRPC_ADDRESS", "env-host:9002")
	t.Setenv("CRYPTO_KEY", "/env/client-private.pem")
	t.Setenv("TLS_AGENT_CERT_PATH", "/env/server.crt")
	t.Setenv("LOCAL_STORAGE_PATH", "/env/client-storage/")
	t.Setenv("LOG_LEVEL", "warn")

	cfg, err := GetClientConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "env-host:9002", cfg.GRPCRemoteAddr)
	assert.Equal(t, "/env/server.crt", cfg.TLSCertPath)
	assert.Equal(t, "/env/client-storage/", cfg.LocalStoragePath)
	assert.Equal(t, "warn", cfg.LogLevel)
}

func TestGetClientConfigEnvOverridesFlags(t *testing.T) {
	defer resetFlagsAndArgs(
		t,
		"-g", "flag-host:9001",
		"-crypto-key", "/flag/client-private.pem",
		"-tc", "/flag/server.crt",
		"-ll", "error",
	)()

	t.Setenv("GRPC_ADDRESS", "env-host:9002")
	t.Setenv("CRYPTO_KEY", "/env/client-private.pem")
	t.Setenv("TLS_AGENT_CERT_PATH", "/env/server.crt")
	t.Setenv("LOG_LEVEL", "warn")

	cfg, err := GetClientConfig()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "env-host:9002", cfg.GRPCRemoteAddr)
	assert.Equal(t, "/env/server.crt", cfg.TLSCertPath)
	assert.Equal(t, "warn", cfg.LogLevel)
}
