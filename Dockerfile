# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG BUILD_COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.buildVersion=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.buildCommit=${BUILD_COMMIT}" \
    -o /app/bin/server \
    ./cmd/gophkeeper_server


FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8081

ENV SERVER_ADDR=:8081

CMD ["/app/server"]