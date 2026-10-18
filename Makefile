APP_NAME := gophkeeper

SERVER_MAIN := ./cmd/gophkeeper_server
CLIENT_MAIN := ./cmd/gophkeeper_client

BIN_DIR := bin

GO := go

SERVER_ADDR ?= localhost:8081
DATABASE_DSN ?= postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable

VERSION ?= dev
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

LDFLAGS := -X main.buildVersion=$(VERSION) \
           -X main.buildDate=$(BUILD_DATE) \
           -X main.buildCommit=$(BUILD_COMMIT)


CERT_DIR := certs
CERT_FILE := $(CERT_DIR)/server.crt
KEY_FILE := $(CERT_DIR)/server.key
CERT_CONFIG := $(CERT_DIR)/server.cnf

CERT_DAYS ?= 365
CERT_HOST ?= localhost
CERT_IP ?= 127.0.0.1


.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build              Build client and server for current OS"
	@echo "  make build-server       Build server for current OS"
	@echo "  make build-client       Build client for current OS"
	@echo "  make build-all          Build client and server for Windows, Linux, macOS"
	@echo "  make build-linux        Build client and server for Linux"
	@echo "  make build-windows      Build client and server for Windows"
	@echo "  make build-darwin       Build client and server for macOS"
	@echo "  make run-server         Run server"
	@echo "  make run-client         Run client"
	@echo "  make test               Run tests"
	@echo "  make tidy               Run go mod tidy"
	@echo "  make clean              Remove binaries"
	@echo "  make certs              Generate self-signed TLS certs(local use)"
	@echo "  make docker-up          Run server and database"
	@echo "  make docker-down        Stop server and database"
	@echo "  make docker-build       Build image server"

.PHONY: build
build: build-server build-client

.PHONY: build-server
build-server:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/server $(SERVER_MAIN)

.PHONY: build-client
build-client:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/client $(CLIENT_MAIN)

.PHONY: build-all
build-all: build-linux build-windows build-darwin

.PHONY: build-linux
build-linux:
	@mkdir -p $(BIN_DIR)/linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/linux/server $(SERVER_MAIN)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/linux/client $(CLIENT_MAIN)

.PHONY: build-windows
build-windows:
	@mkdir -p $(BIN_DIR)/windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/windows/server.exe $(SERVER_MAIN)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/windows/client.exe $(CLIENT_MAIN)

.PHONY: build-darwin
build-darwin:
	@mkdir -p $(BIN_DIR)/darwin
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/darwin/server $(SERVER_MAIN)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/darwin/client $(CLIENT_MAIN)

.PHONY: run-server
run-server:
	DATABASE_DSN="$(DATABASE_DSN)" \
	SERVER_ADDR="$(SERVER_ADDR)" \
	$(GO) run $(SERVER_MAIN)

.PHONY: run-client
run-client:
	SERVER_ADDR="$(SERVER_ADDR)" \
	$(GO) run $(CLIENT_MAIN)

.PHONY: test
test:
	$(GO) test ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_COMMIT := $(shell git rev-parse --short HEAD)
VERSION := docker

.PHONY: docker-up
docker-up:
	BUILD_DATE=$(BUILD_DATE) BUILD_COMMIT=$(BUILD_COMMIT) VERSION=$(VERSION) docker compose up --build

.PHONY: docker-build
docker-build:
	BUILD_DATE=$(BUILD_DATE) BUILD_COMMIT=$(BUILD_COMMIT) VERSION=$(VERSION) docker compose build --no-cache

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: certs
certs: $(CERT_FILE) $(KEY_FILE)
$(CERT_FILE) $(KEY_FILE):
	@mkdir -p $(CERT_DIR)
	@printf "[req]\n" > $(CERT_CONFIG)
	@printf "default_bits = 2048\n" >> $(CERT_CONFIG)
	@printf "prompt = no\n" >> $(CERT_CONFIG)
	@printf "default_md = sha256\n" >> $(CERT_CONFIG)
	@printf "distinguished_name = dn\n" >> $(CERT_CONFIG)
	@printf "x509_extensions = v3_req\n\n" >> $(CERT_CONFIG)
	@printf "[dn]\n" >> $(CERT_CONFIG)
	@printf "CN = $(CERT_HOST)\n\n" >> $(CERT_CONFIG)
	@printf "[v3_req]\n" >> $(CERT_CONFIG)
	@printf "subjectAltName = @alt_names\n\n" >> $(CERT_CONFIG)
	@printf "[alt_names]\n" >> $(CERT_CONFIG)
	@printf "DNS.1 = $(CERT_HOST)\n" >> $(CERT_CONFIG)
	@printf "DNS.2 = localhost\n" >> $(CERT_CONFIG)
	@printf "IP.1 = $(CERT_IP)\n" >> $(CERT_CONFIG)
	@openssl req -x509 -nodes -newkey rsa:2048 \
		-keyout $(KEY_FILE) \
		-out $(CERT_FILE) \
		-days $(CERT_DAYS) \
		-config $(CERT_CONFIG)
	@echo "Generated $(CERT_FILE) and $(KEY_FILE)"