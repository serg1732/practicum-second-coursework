package config

import (
	"flag"
)

// GophKeeperServerConfig - конфиг сервер.
type GophKeeperServerConfig struct {
	// GRPCRunAddr - адрес обработки запросов по работе с метриками в хранилище.
	GRPCRunAddr string `env:"GRPC_ADDRESS"`
	// DSN - подключение к БД.
	DSN string `env:"DATABASE_DSN"`
	// TLSCertPath путь до сертификата сервер
	TLSCertPath string `env:"TLS_CERT_SERVER_PATH"`
	// TLSCertPath путь до ключа TLS сервера
	TLSKeyPath string `env:"TLS_KEY_SERVER_PATH"`
	// LocalFileStoragePath путь до места локального хранения файлов
	LocalFileStoragePath string `env:"LOCAL_FILE_STORAGE_PATH"`
	// TokenExpired время жизни токена в секундах
	TokenExpired int64 `env:"TOKEN_EXP"`
	// LogLevel уровень логирования
	LogLevel string `env:"LOG_LEVEL"`
}

// GetServerConfig - получение конфига сервера.
func GetServerConfig() *GophKeeperServerConfig {
	var serverConfig GophKeeperServerConfig
	flag.StringVar(&serverConfig.GRPCRunAddr, "g", ":8081", "address and port to run grpc server")
	flag.StringVar(&serverConfig.DSN, "d", "", "database connection string")
	flag.StringVar(&serverConfig.TLSCertPath, "tc", "", "tls cert path")
	flag.StringVar(&serverConfig.TLSKeyPath, "tk", "", "tls key path")
	flag.StringVar(&serverConfig.LocalFileStoragePath, "lp", "/tmp/", "local path")
	flag.StringVar(&serverConfig.LogLevel, "ll", "INFO", "log level")
	flag.Int64Var(&serverConfig.TokenExpired, "e", 600, "token lifetime sec")

	return &serverConfig
}
