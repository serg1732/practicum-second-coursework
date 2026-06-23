package config

import (
	"flag"

	"github.com/caarlos0/env/v11"
)

// GophKeeperClientConfig - конфиг клиента.
type GophKeeperClientConfig struct {
	// GRPCRemoteAddr - адрес GRPC сервера отправки метрик.
	GRPCRemoteAddr string `env:"GRPC_ADDRESS"`
	// TLSCertPath путь до сертификата сервера
	TLSCertPath string `env:"TLS_AGENT_CERT_PATH"`
	// LocalStoragePath путь где хранятся локальные файлы
	LocalStoragePath string `env:"LOCAL_STORAGE_PATH"`
	// LogLevel уровень логирования
	LogLevel string `env:"LOG_LEVEL"`
}

// GetClientConfig - получение конфига клиента.
func GetClientConfig() (*GophKeeperClientConfig, error) {
	var serverConfig GophKeeperClientConfig
	flag.StringVar(&serverConfig.GRPCRemoteAddr, "g", "localhost:8081", "address and port to run grpc server")
	flag.StringVar(&serverConfig.TLSCertPath, "tc", "", "tls cert path")
	flag.StringVar(&serverConfig.LocalStoragePath, "s", "/tmp/client/", "local storage path")
	flag.StringVar(&serverConfig.LogLevel, "ll", "INFO", "log level")

	flag.Parse()

	if err := env.Parse(&serverConfig); err != nil {
		return nil, err
	}
	return &serverConfig, nil
}
