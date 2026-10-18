package tui

import (
	"context"
	"time"

	keepermodels "github.com/serg1732/practicum-second-coursework/internal/models"
)

const (
	layoutDate = "02.01.2006"

	applicationName = "GophKeeper"
)

type ItemType string

const (
	ItemCreds  ItemType = "creds"
	ItemCard   ItemType = "card"
	ItemText   ItemType = "text"
	ItemBinary ItemType = "binary"
)

var itemTypes = []ItemType{
	ItemText,
	ItemCreds,
	ItemCard,
	ItemBinary,
}

type Item struct {
	ID        string
	Type      ItemType
	Title     string
	Login     string
	Secret    string
	Meta      string
	UpdatedAt time.Time
}

type Draft struct {
	Type   ItemType
	Title  string
	Login  string
	Secret string
	Meta   string
}

type FileSyncStatus string

const (
	FileSyncStatusSynced     FileSyncStatus = "synced"
	FileSyncStatusLocalOnly  FileSyncStatus = "local_only"
	FileSyncStatusRemoteOnly FileSyncStatus = "remote_only"
)

type FileSyncInfo struct {
	Name            string
	LocalPath       string
	Size            int64
	RemoteCreatedAt time.Time
	LocalUpdatedAt  time.Time
	Status          FileSyncStatus
}

type SyncSnapshot struct {
	Items []Item
	Files []FileSyncInfo
}

// Client - адаптер на GRPC клиентом для TUI.
type Client interface {
	Register(ctx context.Context, username, masterPassword string) error
	Login(ctx context.Context, username, masterPassword string) error
	Sync(ctx context.Context) (SyncSnapshot, error)
	SyncFiles(ctx context.Context) (SyncSnapshot, error)
	Create(ctx context.Context, draft Draft) error
	Update(ctx context.Context, id string, draft Draft) error
	Delete(ctx context.Context, id string) error
	UploadFile(ctx context.Context, path string) error
	DownloadFile(ctx context.Context, name string) error
	RemoveFile(ctx context.Context, name string) error
	DeleteLocalFile(ctx context.Context, name string) error
	Logout(ctx context.Context) error
}

// GophKeeperGRPCClient - описывает методы gRPC-клиента, которые нужны TUI.
type GophKeeperGRPCClient interface {
	Registration(ctx context.Context, username, password string) (keepermodels.Token, error)
	Authentication(ctx context.Context, username, password string) (keepermodels.Token, error)

	Synchronize(ctx context.Context, password string, token keepermodels.Token) (keepermodels.SyncResult, error)

	TextCreate(ctx context.Context, name, description, password, plaintext string, token keepermodels.Token) error
	TextUpdate(ctx context.Context, name, passwordSecure, text string, token keepermodels.Token) error
	TextDelete(ctx context.Context, text string, token keepermodels.Token) error

	CredsCreate(ctx context.Context, name, description, passwordSecure, login, password string, token keepermodels.Token) error
	CredsUpdate(ctx context.Context, name, passwordSecure, login, password string, token keepermodels.Token) error
	CredsDelete(ctx context.Context, creds string, token keepermodels.Token) error

	CardCreate(ctx context.Context, name, description, password, paymentSystem, number, holder, cvc, endDate string, token keepermodels.Token) error
	CardUpdate(ctx context.Context, name, passwordSecure, paymentSystem, number, holder, cvc, endDateCard string, token keepermodels.Token) error
	CardDelete(ctx context.Context, card string, token keepermodels.Token) error

	FileUpload(ctx context.Context, name, password string, file []byte, token keepermodels.Token) (string, error)
	FileDownload(ctx context.Context, name, password string, token keepermodels.Token) ([]byte, error)
	FileRemove(ctx context.Context, binary string, token keepermodels.Token) error
}
