package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	keepermodels "github.com/serg1732/practicum-second-coursework/internal/models"
)

const defaultFilesDir = ".gophkeeper/files"

// GRPCAdapter - адаптер над GRPC клиентом для TUI.
type GRPCAdapter struct {
	client         GophKeeperGRPCClient
	token          keepermodels.Token
	masterPassword string
	filesDir       string
}

// BuildGRPCAdapter - создание адаптера над GRPC клиентом для TUI.
func BuildGRPCAdapter(client GophKeeperGRPCClient, filesDir string) *GRPCAdapter {
	if filesDir == "" {
		filesDir = defaultFilesDir
	}

	return &GRPCAdapter{
		client:   client,
		filesDir: filesDir,
	}
}

// Register - регистрация.
func (a *GRPCAdapter) Register(ctx context.Context, username, masterPassword string) error {
	_, err := a.client.Registration(ctx, username, masterPassword)
	return err
}

// Login - авторизация.
func (a *GRPCAdapter) Login(ctx context.Context, username, masterPassword string) error {
	token, err := a.client.Authentication(ctx, username, masterPassword)
	if err != nil {
		return err
	}

	a.token = token
	a.masterPassword = masterPassword
	return nil
}

// Logout - выход из учетной записи.
func (a *GRPCAdapter) Logout(_ context.Context) error {
	a.token = keepermodels.Token{}
	a.masterPassword = ""
	return nil
}

// Sync - синхронизация данных на клиенте.
func (a *GRPCAdapter) Sync(ctx context.Context) (SyncSnapshot, error) {
	if err := a.checkAuth(); err != nil {
		return SyncSnapshot{}, err
	}

	result, err := a.client.Synchronize(ctx, a.masterPassword, a.token)
	if err != nil {
		return SyncSnapshot{}, err
	}

	files, err := a.fileStatuses(remoteFilesFromBinary(result.Binary))
	if err != nil {
		return SyncSnapshot{}, err
	}

	return SyncSnapshot{
		Items: syncResultToItems(result),
		Files: files,
	}, nil
}

// SyncFiles - синхронизация файлов.
func (a *GRPCAdapter) SyncFiles(ctx context.Context) (SyncSnapshot, error) {
	snapshot, err := a.Sync(ctx)
	if err != nil {
		return SyncSnapshot{}, err
	}

	if err = os.MkdirAll(a.filesDir, 0o700); err != nil {
		return SyncSnapshot{}, fmt.Errorf("create files dir: %w", err)
	}

	for _, file := range snapshot.Files {
		if err = a.syncFile(ctx, file); err != nil {
			return SyncSnapshot{}, err
		}
	}

	return a.Sync(ctx)
}

// Create - добавление данных / файла.
func (a *GRPCAdapter) Create(ctx context.Context, draft Draft) error {
	if err := a.checkAuth(); err != nil {
		return err
	}

	switch draft.Type {
	case ItemText:
		return a.client.TextCreate(ctx, draft.Title, draft.Meta, a.masterPassword, draft.Secret, a.token)
	case ItemCreds:
		return a.client.CredsCreate(ctx, draft.Title, draft.Meta, a.masterPassword, draft.Login, draft.Secret, a.token)
	case ItemCard:
		return a.createCard(ctx, draft)
	case ItemBinary:
		return a.UploadFile(ctx, draft.Secret)
	default:
		return fmt.Errorf("unsupported item type %q", draft.Type)
	}
}

// Update - обновление сущности.
func (a *GRPCAdapter) Update(ctx context.Context, id string, draft Draft) error {
	if err := a.checkAuth(); err != nil {
		return err
	}

	kind, name := splitItemID(id)
	if name == "" {
		name = draft.Title
	}

	switch kind {
	case string(ItemText):
		return a.client.TextUpdate(ctx, name, a.masterPassword, draft.Secret, a.token)
	case string(ItemCreds):
		return a.client.CredsUpdate(ctx, name, a.masterPassword, draft.Login, draft.Secret, a.token)
	case string(ItemCard):
		return a.updateCard(ctx, name, draft)
	case "file":
		return a.UploadFile(ctx, draft.Secret)
	default:
		return fmt.Errorf("unsupported item id %q", id)
	}
}

// Delete - удаление сущности.
func (a *GRPCAdapter) Delete(ctx context.Context, id string) error {
	if err := a.checkAuth(); err != nil {
		return err
	}

	kind, name := splitItemID(id)
	if name == "" {
		return fmt.Errorf("bad item id %q", id)
	}

	switch kind {
	case string(ItemText):
		return a.client.TextDelete(ctx, name, a.token)
	case string(ItemCreds):
		return a.client.CredsDelete(ctx, name, a.token)
	case string(ItemCard):
		return a.client.CardDelete(ctx, name, a.token)
	case "file":
		return a.RemoveFile(ctx, name)
	default:
		return fmt.Errorf("неизветсный тип данных id %q", id)
	}
}

// UploadFile - загрузка файла.
func (a *GRPCAdapter) UploadFile(ctx context.Context, path string) error {
	if err := a.checkAuth(); err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла: %w", err)
	}

	name, err := normalizedFileName(filepath.Base(path))
	if err != nil {
		return err
	}

	if _, err = a.client.FileUpload(ctx, name, a.masterPassword, data, a.token); err != nil {
		return err
	}

	return a.saveLocalFileCopy(path, name)
}

// DownloadFile - скачивание файла.
func (a *GRPCAdapter) DownloadFile(ctx context.Context, name string) error {
	if err := a.checkAuth(); err != nil {
		return err
	}

	name, err := normalizedFileName(name)
	if err != nil {
		return err
	}

	data, err := a.client.FileDownload(ctx, name, a.masterPassword, a.token)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(a.filesDir, 0o700); err != nil {
		return fmt.Errorf("ошибка при создании директории: %w", err)
	}

	path := filepath.Join(a.filesDir, name)
	if err = os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("ошибка при записи файла: %w", err)
	}

	return nil
}

// RemoveFile - удаление файла на сервере.
func (a *GRPCAdapter) RemoveFile(ctx context.Context, name string) error {
	if err := a.checkAuth(); err != nil {
		return err
	}

	name, err := normalizedFileName(name)
	if err != nil {
		return err
	}

	return a.client.FileRemove(ctx, name, a.token)
}

// DeleteLocalFile - удаление файла в локальном хранилище.
func (a *GRPCAdapter) DeleteLocalFile(_ context.Context, name string) error {
	if err := a.checkAuth(); err != nil {
		return err
	}

	name, err := normalizedFileName(name)
	if err != nil {
		return err
	}

	path := filepath.Join(a.filesDir, name)
	if err = os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("локальный файл %q не найден", name)
		}

		return fmt.Errorf("ошибка при удалении локального файла: %w", err)
	}

	return nil
}

// checkAuth - проверка авторизации.
func (a *GRPCAdapter) checkAuth() error {
	if a.token.AccessToken == "" || a.masterPassword == "" {
		return errors.New("пользователь не авторизован")
	}

	return nil
}

// createCard - создание секрета с банковской картой.
func (a *GRPCAdapter) createCard(ctx context.Context, draft Draft) error {
	card, err := parseCardDraft(draft)
	if err != nil {
		return err
	}

	return a.client.CardCreate(
		ctx,
		draft.Title,
		card.Description,
		a.masterPassword,
		card.PaymentSystem,
		card.Number,
		card.Holder,
		card.CVC,
		card.EndDate,
		a.token,
	)
}

// updateCard - обновление секрета с банковской картой.
func (a *GRPCAdapter) updateCard(ctx context.Context, name string, draft Draft) error {
	card, err := parseCardDraft(draft)
	if err != nil {
		return err
	}

	return a.client.CardUpdate(
		ctx,
		name,
		a.masterPassword,
		card.PaymentSystem,
		card.Number,
		card.Holder,
		card.CVC,
		card.EndDate,
		a.token,
	)
}

// syncFile - синхронизация всех файлов (клиент / сервер).
func (a *GRPCAdapter) syncFile(ctx context.Context, file FileSyncInfo) error {
	switch file.Status {
	case FileSyncStatusLocalOnly:
		if err := a.UploadFile(ctx, file.LocalPath); err != nil {
			return fmt.Errorf("загрузка локального файла %q: %w", file.Name, err)
		}
	case FileSyncStatusRemoteOnly:
		if err := a.DownloadFile(ctx, file.Name); err != nil {
			return fmt.Errorf("скачивание файла с сервера %q: %w", file.Name, err)
		}
	}

	return nil
}

// saveLocalFileCopy - валидация папок и сохранение файла в локале.
func (a *GRPCAdapter) saveLocalFileCopy(srcPath, name string) error {
	if err := os.MkdirAll(a.filesDir, 0o700); err != nil {
		return fmt.Errorf("ошибка при создании директории: %w", err)
	}

	dstPath := filepath.Join(a.filesDir, name)
	if filepath.Clean(srcPath) == filepath.Clean(dstPath) {
		return nil
	}
	return copyFile(srcPath, dstPath)
}

// fileStatuses - обновление статусов у файлов.
func (a *GRPCAdapter) fileStatuses(remote map[string]time.Time) ([]FileSyncInfo, error) {
	local, err := scanLocalFiles(a.filesDir)
	if err != nil {
		return nil, err
	}

	result := make([]FileSyncInfo, 0, len(remote)+len(local))

	for name, remoteCreatedAt := range remote {
		if localInfo, ok := local[name]; ok {
			delete(local, name)
			localInfo.RemoteCreatedAt = remoteCreatedAt
			localInfo.Status = FileSyncStatusSynced
			result = append(result, localInfo)
			continue
		}

		result = append(result, FileSyncInfo{
			Name:            name,
			RemoteCreatedAt: remoteCreatedAt,
			Status:          FileSyncStatusRemoteOnly,
		})
	}

	for _, localInfo := range local {
		localInfo.Status = FileSyncStatusLocalOnly
		result = append(result, localInfo)
	}

	sortFileStatuses(result)
	return result, nil
}

func sortFileStatuses(files []FileSyncInfo) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Status != files[j].Status {
			return files[i].Status < files[j].Status
		}

		return files[i].Name < files[j].Name
	})
}

func scanLocalFiles(dir string) (map[string]FileSyncInfo, error) {
	result := make(map[string]FileSyncInfo)

	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read files dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("read file info: %w", err)
		}

		name := safeFileName(entry.Name())
		result[name] = FileSyncInfo{
			Name:           name,
			LocalPath:      filepath.Join(dir, name),
			Size:           info.Size(),
			LocalUpdatedAt: info.ModTime(),
			Status:         FileSyncStatusLocalOnly,
		}
	}

	return result, nil
}

func normalizedFileName(name string) (string, error) {
	name = safeFileName(name)
	if name == "" {
		return "", errors.New("empty file name")
	}

	return name, nil
}

func safeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) {
		return ""
	}

	return name
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open destination file: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return out.Sync()
}
