package config

import (
	"flag"
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
func GetClientConfig() *GophKeeperClientConfig {
	var clientConfig GophKeeperClientConfig
	flag.StringVar(&clientConfig.GRPCRemoteAddr, "g", "localhost:8081", "address and port to run grpc server")
	flag.StringVar(&clientConfig.TLSCertPath, "tc", "", "tls cert path")
	flag.StringVar(&clientConfig.LocalStoragePath, "s", "/tmp/client/", "local storage path")
	flag.StringVar(&clientConfig.LogLevel, "ll", "INFO", "log level")

	return &clientConfig
}
