# GophKeeper

GophKeeper — клиент-серверное приложение для безопасного хранения приватных данных: кредов, банковских карт, текстовых и
бинарных данных.

Проект состоит из двух частей:

- **server** — серверная часть с API, авторизацией, хранением данных и синхронизацией;
- **client** — консольный клиент для работы с хранилищем пользователя.

## Возможности

- регистрация и аутентификация пользователей;
- хранение приватных данных на сервере;
- синхронизация данных между клиентом и сервером;
- поддержка текстовых данных, карт, логинов/паролей и файлов;
- сборка клиента и сервера под Linux, Windows и macOS;

## Требования

- Go 1.26+
- PostgreSQL
- Make
- OpenSSL — нужен для команды `make certs`
- Docker и Docker Compose — нужны для запуска сервера и PostgreSQL через `make docker-up`

## Быстрый старт

### 1. Подготовить зависимости

```bash
go mod tidy
```

Или через Makefile:

```bash
make tidy
```

### 2. Сгенерировать TLS-сертификаты

```bash
make certs
```

По умолчанию сертификаты будут созданы в папке `certs`:

```text
certs/server.crt
certs/server.key
```

Можно переопределить хост и IP:

```bash
make certs CERT_HOST=localhost CERT_IP=127.0.0.1
```

### 3. Запустить сервер

```bash
make run-server
```

По умолчанию используются параметры:

```text
SERVER_ADDR=localhost:8081
DATABASE_DSN=postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable
```

Их можно переопределить при запуске:

```bash
make run-server \
  SERVER_ADDR=localhost:8081 \
  DATABASE_DSN="postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable"
```

### 4. Запустить клиент

```bash
make run-client
```

При необходимости можно указать адрес сервера:

```bash
make run-client SERVER_ADDR=localhost:8081
```

## Конфигурация клиента и сервера

Параметры запуска можно задавать двумя способами:

1. через CLI-флаги;
2. через переменные окружения.

Конфиг сначала заполняется значениями по умолчанию и CLI-флагами, после этого читаются переменные окружения. Если один и
тот же параметр задан и флагом, и переменной окружения, будет использовано значение из переменной окружения.

### Флаги сервера

Пример запуска сервера напрямую:

```bash
./bin/gophkeeper_server \
  -g :8081 \
  -d "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" \
  -tc certs/server.crt \
  -tk certs/server.key \
  -lp /tmp/gophkeeper-server/ \
  -e 600 \
  -ll INFO
```

| Флаг  | Переменная окружения      | Значение по умолчанию | Описание                                       |
|-------|---------------------------|-----------------------|------------------------------------------------|
| `-g`  | `GRPC_ADDRESS`            | `:8081`               | адрес и порт gRPC-сервера                      |
| `-d`  | `DATABASE_DSN`            | пусто                 | строка подключения к PostgreSQL                |
| `-tc` | `TLS_CERT_SERVER_PATH`    | пусто                 | путь до TLS-сертификата сервера                |
| `-tk` | `TLS_KEY_SERVER_PATH`     | пусто                 | путь до TLS-ключа сервера                      |
| `-lp` | `LOCAL_FILE_STORAGE_PATH` | `/tmp/`               | путь до локального хранилища файлов на сервере |
| `-e`  | `TOKEN_EXP`               | `600`                 | время жизни токена в секундах                  |
| `-ll` | `LOG_LEVEL`               | `INFO`                | уровень логирования                            |

Пример через переменные окружения:

```bash
GRPC_ADDRESS=:8081 \
DATABASE_DSN="postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" \
TLS_CERT_SERVER_PATH=certs/server.crt \
TLS_KEY_SERVER_PATH=certs/server.key \
LOCAL_FILE_STORAGE_PATH=/tmp/gophkeeper-server/ \
TOKEN_EXP=600 \
LOG_LEVEL=INFO \
./bin/gophkeeper_server
```

### Флаги клиента

Пример запуска клиента напрямую:

```bash
./bin/gophkeeper_client \
  -g localhost:8081 \
  -tc certs/server.crt \
  -s /tmp/gophkeeper-client/ \
  -ll INFO
```

| Флаг  | Переменная окружения  | Значение по умолчанию | Описание                                                  |
|-------|-----------------------|-----------------------|-----------------------------------------------------------|
| `-g`  | `GRPC_ADDRESS`        | `localhost:8081`      | адрес и порт gRPC-сервера                                 |
| `-tc` | `TLS_AGENT_CERT_PATH` | пусто                 | путь до TLS-сертификата сервера, которому доверяет клиент |
| `-s`  | `LOCAL_STORAGE_PATH`  | `/tmp/client/`        | путь до локального хранилища клиента                      |
| `-ll` | `LOG_LEVEL`           | `INFO`                | уровень логирования                                       |

Пример через переменные окружения:

```bash
GRPC_ADDRESS=localhost:8081 \
TLS_AGENT_CERT_PATH=certs/server.crt \
LOCAL_STORAGE_PATH=/tmp/gophkeeper-client/ \
LOG_LEVEL=INFO \
./bin/gophkeeper_client
```

## Запуск сервера и БД в Docker

Сервер и PostgreSQL можно поднять одной командой через Docker Compose:

```bash
make docker-up
```

Команда выполнит сборку образа сервера и запустит сервисы из `docker-compose.yaml`.

Для полной пересборки образов без использования кеша:

```bash
make docker-build
```

Остановить и удалить контейнеры, созданные Docker Compose:

```bash
make docker-down
```

При запуске через `docker-up` в сборку также передаются build info переменные:

| Переменная     | Описание          |
|----------------|-------------------|
| `VERSION`      | версия приложения |
| `BUILD_DATE`   | дата сборки       |
| `BUILD_COMMIT` | git commit сборки |

Пример запуска с явными значениями:

```bash
make docker-up \
  VERSION=v1.0.0 \
  BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  BUILD_COMMIT=$(git rev-parse --short HEAD)
```

## Сборка

Собрать сервер и клиент для текущей ОС:

```bash
make build
```

Отдельно сервер:

```bash
make build-server
```

Отдельно клиент:

```bash
make build-client
```

Собрать под Linux, Windows и macOS:

```bash
make build-all
```

Готовые бинарные файлы появятся в папке `bin`.

## Тесты

```bash
make test
```

## Полезные команды

```bash
make help          # список доступных команд
make tidy          # обновить go.mod/go.sum
make clean         # удалить собранные бинарные файлы
make certs         # сгенерировать TLS-сертификаты
make docker-up     # собрать и запустить сервер + PostgreSQL в Docker
make docker-build  # пересобрать Docker-образы без кеша
make docker-down   # остановить Docker Compose
```

## Переменные окружения и Makefile-переменные

Для запуска напрямую через бинарники используются переменные окружения из конфигов клиента и сервера:

| Переменная                | Где используется | Значение по умолчанию                         | Описание                                       |
|---------------------------|------------------|-----------------------------------------------|------------------------------------------------|
| `GRPC_ADDRESS`            | client/server    | `localhost:8081` у клиента, `:8081` у сервера | адрес gRPC-сервера                             |
| `DATABASE_DSN`            | server           | пусто                                         | строка подключения к PostgreSQL                |
| `TLS_CERT_SERVER_PATH`    | server           | пусто                                         | путь до TLS-сертификата сервера                |
| `TLS_KEY_SERVER_PATH`     | server           | пусто                                         | путь до TLS-ключа сервера                      |
| `LOCAL_FILE_STORAGE_PATH` | server           | `/tmp/`                                       | путь до локального файлового хранилища сервера |
| `TOKEN_EXP`               | server           | `600`                                         | время жизни токена в секундах                  |
| `TLS_AGENT_CERT_PATH`     | client           | пусто                                         | путь до TLS-сертификата сервера для клиента    |
| `LOCAL_STORAGE_PATH`      | client           | `/tmp/client/`                                | путь до локального хранилища клиента           |
| `LOG_LEVEL`               | client/server    | `INFO`                                        | уровень логирования                            |

Также в Makefile используются переменные для сборки, запуска и генерации сертификатов:

| Переменная     | Значение по умолчанию | Описание                                 |
|----------------|-----------------------|------------------------------------------|
| `SERVER_ADDR`  | `localhost:8081`      | адрес сервера при запуске через Makefile |
| `VERSION`      | `dev`                 | версия приложения для build info         |
| `BUILD_DATE`   | `unknown`             | дата сборки для build info               |
| `BUILD_COMMIT` | `unknown`             | git commit для build info                |
| `CERT_HOST`    | `localhost`           | DNS-имя в TLS-сертификате                |
| `CERT_IP`      | `127.0.0.1`           | IP-адрес в TLS-сертификате               |
| `CERT_DAYS`    | `365`                 | срок действия сертификата                |

## Очистка

```bash
make clean
```
