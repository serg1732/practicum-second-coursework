package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	keepermodels "github.com/serg1732/practicum-second-coursework/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMasterPassword = "master-password"

var testToken = keepermodels.Token{AccessToken: "access-token"}

func newMockGRPCClient(t *testing.T) *MockGophKeeperGRPCClient {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	return NewMockGophKeeperGRPCClient(ctrl)
}

func newAuthorizedAdapter(client GophKeeperGRPCClient, filesDir string) *GRPCAdapter {
	return &GRPCAdapter{
		client:         client,
		token:          testToken,
		masterPassword: testMasterPassword,
		filesDir:       filesDir,
	}
}

func testItemID(kind ItemType, name string) string {
	return string(kind) + ":" + name
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func requireFileContent(t *testing.T, path string, expected string) {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expected, string(data))
}

func buildCardDraft(title string) Draft {
	return Draft{
		Type:  ItemCard,
		Title: title,
		Meta:  `{"description":"card description","payment_system":"visa","number":"4111111111111111","holder":"IVAN IVANOV","cvc":"123","end_date":"12/29"}`,
	}
}

func TestBuildGRPCAdapter(t *testing.T) {
	t.Run("использует директорию по умолчанию", func(t *testing.T) {
		adapter := BuildGRPCAdapter(nil, "")

		assert.Equal(t, defaultFilesDir, adapter.filesDir)
	})

	t.Run("использует переданную директорию", func(t *testing.T) {
		adapter := BuildGRPCAdapter(nil, "/tmp/gophkeeper-files")

		assert.Equal(t, "/tmp/gophkeeper-files", adapter.filesDir)
	})
}

func TestGRPCAdapterRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("успешная регистрация", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := BuildGRPCAdapter(client, t.TempDir())

		client.EXPECT().
			Registration(ctx, "user", "password").
			Return(keepermodels.Token{AccessToken: "registered-token"}, nil)

		err := adapter.Register(ctx, "user", "password")

		require.NoError(t, err)
	})

	t.Run("ошибка клиента", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := BuildGRPCAdapter(client, t.TempDir())
		wantErr := errors.New("registration failed")

		client.EXPECT().
			Registration(ctx, "user", "password").
			Return(keepermodels.Token{}, wantErr)

		err := adapter.Register(ctx, "user", "password")

		require.ErrorIs(t, err, wantErr)
	})
}

func TestGRPCAdapterLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("успешная авторизация сохраняет токен и master password", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := BuildGRPCAdapter(client, t.TempDir())

		client.EXPECT().
			Authentication(ctx, "user", testMasterPassword).
			Return(testToken, nil)

		err := adapter.Login(ctx, "user", testMasterPassword)

		require.NoError(t, err)
		assert.Equal(t, testToken, adapter.token)
		assert.Equal(t, testMasterPassword, adapter.masterPassword)
	})

	t.Run("ошибка клиента не меняет текущую авторизацию", func(t *testing.T) {
		client := newMockGRPCClient(t)
		wantErr := errors.New("authentication failed")
		oldToken := keepermodels.Token{AccessToken: "old-token"}
		adapter := &GRPCAdapter{
			client:         client,
			token:          oldToken,
			masterPassword: "old-password",
			filesDir:       t.TempDir(),
		}

		client.EXPECT().
			Authentication(ctx, "user", testMasterPassword).
			Return(keepermodels.Token{}, wantErr)

		err := adapter.Login(ctx, "user", testMasterPassword)

		require.ErrorIs(t, err, wantErr)
		assert.Equal(t, oldToken, adapter.token)
		assert.Equal(t, "old-password", adapter.masterPassword)
	})
}

func TestGRPCAdapterLogout(t *testing.T) {
	adapter := newAuthorizedAdapter(nil, t.TempDir())

	err := adapter.Logout(context.Background())

	require.NoError(t, err)
	assert.Empty(t, adapter.token.AccessToken)
	assert.Empty(t, adapter.masterPassword)
}

func TestGRPCAdapterCheckAuth(t *testing.T) {
	tests := []struct {
		name          string
		token         keepermodels.Token
		masterPass    string
		expectedError bool
	}{
		{
			name:          "нет токена",
			token:         keepermodels.Token{},
			masterPass:    testMasterPassword,
			expectedError: true,
		},
		{
			name:          "нет master password",
			token:         testToken,
			masterPass:    "",
			expectedError: true,
		},
		{
			name:          "пользователь авторизован",
			token:         testToken,
			masterPass:    testMasterPassword,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &GRPCAdapter{
				token:          tt.token,
				masterPassword: tt.masterPass,
			}

			err := adapter.checkAuth()

			if tt.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "пользователь не авторизован")
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGRPCAdapterSync(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		snapshot, err := adapter.Sync(ctx)

		require.Error(t, err)
		assert.Empty(t, snapshot.Items)
		assert.Empty(t, snapshot.Files)
	})

	t.Run("ошибка клиента", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, filepath.Join(t.TempDir(), "missing"))
		wantErr := errors.New("sync failed")

		client.EXPECT().
			Synchronize(ctx, testMasterPassword, testToken).
			Return(keepermodels.SyncResult{}, wantErr)

		snapshot, err := adapter.Sync(ctx)

		require.ErrorIs(t, err, wantErr)
		assert.Empty(t, snapshot.Items)
		assert.Empty(t, snapshot.Files)
	})

	t.Run("успешная синхронизация без данных", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, filepath.Join(t.TempDir(), "missing"))

		client.EXPECT().
			Synchronize(ctx, testMasterPassword, testToken).
			Return(keepermodels.SyncResult{}, nil)

		snapshot, err := adapter.Sync(ctx)

		require.NoError(t, err)
		assert.Empty(t, snapshot.Items)
		assert.Empty(t, snapshot.Files)
	})
}

func TestGRPCAdapterSyncFiles(t *testing.T) {
	ctx := context.Background()

	t.Run("загружает локальный файл и делает повторную синхронизацию", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		localPath := filepath.Join(filesDir, "local.txt")
		writeTestFile(t, localPath, []byte("local content"))

		adapter := newAuthorizedAdapter(client, filesDir)

		gomock.InOrder(
			client.EXPECT().
				Synchronize(ctx, testMasterPassword, testToken).
				Return(keepermodels.SyncResult{}, nil),
			client.EXPECT().
				FileUpload(ctx, "local.txt", testMasterPassword, []byte("local content"), testToken).
				Return("local.txt", nil),
			client.EXPECT().
				Synchronize(ctx, testMasterPassword, testToken).
				Return(keepermodels.SyncResult{}, nil),
		)

		snapshot, err := adapter.SyncFiles(ctx)

		require.NoError(t, err)
		require.Len(t, snapshot.Files, 1)
		assert.Equal(t, "local.txt", snapshot.Files[0].Name)
	})

	t.Run("возвращает ошибку первой синхронизации", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		wantErr := errors.New("sync failed")

		client.EXPECT().
			Synchronize(ctx, testMasterPassword, testToken).
			Return(keepermodels.SyncResult{}, wantErr)

		snapshot, err := adapter.SyncFiles(ctx)

		require.ErrorIs(t, err, wantErr)
		assert.Empty(t, snapshot.Items)
		assert.Empty(t, snapshot.Files)
	})
}

func TestGRPCAdapterCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		err := adapter.Create(ctx, Draft{Type: ItemText})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "пользователь не авторизован")
	})

	t.Run("создание текста", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := Draft{Type: ItemText, Title: "note", Meta: "description", Secret: "text value"}

		client.EXPECT().
			TextCreate(ctx, "note", "description", testMasterPassword, "text value", testToken).
			Return(nil)

		err := adapter.Create(ctx, draft)

		require.NoError(t, err)
	})

	t.Run("ошибка создания текста", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		wantErr := errors.New("text create failed")
		draft := Draft{Type: ItemText, Title: "note", Meta: "description", Secret: "text value"}

		client.EXPECT().
			TextCreate(ctx, "note", "description", testMasterPassword, "text value", testToken).
			Return(wantErr)

		err := adapter.Create(ctx, draft)

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("создание логина и пароля", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := Draft{Type: ItemCreds, Title: "site", Meta: "description", Login: "login", Secret: "password"}

		client.EXPECT().
			CredsCreate(ctx, "site", "description", testMasterPassword, "login", "password", testToken).
			Return(nil)

		err := adapter.Create(ctx, draft)

		require.NoError(t, err)
	})

	t.Run("создание карты", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := buildCardDraft("card")

		client.EXPECT().
			CardCreate(
				ctx,
				"card",
				"card description",
				testMasterPassword,
				"visa",
				"4111111111111111",
				"IVAN IVANOV",
				"123",
				gomock.Any(),
				testToken,
			).
			Return(nil)

		err := adapter.Create(ctx, draft)

		require.NoError(t, err)
	})

	t.Run("ошибка парсинга карты", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := Draft{Type: ItemCard, Title: "card", Meta: "bad-json"}

		err := adapter.Create(ctx, draft)

		require.Error(t, err)
	})

	t.Run("создание файла", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		sourcePath := filepath.Join(t.TempDir(), "secret.bin")
		writeTestFile(t, sourcePath, []byte("binary data"))
		adapter := newAuthorizedAdapter(client, filesDir)
		draft := Draft{Type: ItemBinary, Secret: sourcePath}

		client.EXPECT().
			FileUpload(ctx, "secret.bin", testMasterPassword, []byte("binary data"), testToken).
			Return("secret.bin", nil)

		err := adapter.Create(ctx, draft)

		require.NoError(t, err)
		requireFileContent(t, filepath.Join(filesDir, "secret.bin"), "binary data")
	})

	t.Run("неизвестный тип", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.Create(ctx, Draft{Type: ItemType("unknown")})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported item type")
	})
}

func TestGRPCAdapterUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		err := adapter.Update(ctx, testItemID(ItemText, "note"), Draft{Type: ItemText})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "пользователь не авторизован")
	})

	t.Run("обновление текста", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := Draft{Type: ItemText, Secret: "new text"}

		client.EXPECT().
			TextUpdate(ctx, "note", testMasterPassword, "new text", testToken).
			Return(nil)

		err := adapter.Update(ctx, testItemID(ItemText, "note"), draft)

		require.NoError(t, err)
	})

	t.Run("обновление текста использует title если имя не пришло из id", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := Draft{Type: ItemText, Title: "fallback-name", Secret: "new text"}

		client.EXPECT().
			TextUpdate(ctx, "fallback-name", testMasterPassword, "new text", testToken).
			Return(nil)

		err := adapter.Update(ctx, testItemID(ItemText, ""), draft)

		require.NoError(t, err)
	})

	t.Run("обновление логина и пароля", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := Draft{Type: ItemCreds, Login: "new-login", Secret: "new-password"}

		client.EXPECT().
			CredsUpdate(ctx, "creds", testMasterPassword, "new-login", "new-password", testToken).
			Return(nil)

		err := adapter.Update(ctx, testItemID(ItemCreds, "creds"), draft)

		require.NoError(t, err)
	})

	t.Run("обновление карты", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := buildCardDraft("ignored-title")

		client.EXPECT().
			CardUpdate(
				ctx,
				"card-name",
				testMasterPassword,
				"visa",
				"4111111111111111",
				"IVAN IVANOV",
				"123",
				gomock.Any(),
				testToken,
			).
			Return(nil)

		err := adapter.Update(ctx, testItemID(ItemCard, "card-name"), draft)

		require.NoError(t, err)
	})

	t.Run("ошибка парсинга карты", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		draft := Draft{Type: ItemCard, Meta: "bad-json"}

		err := adapter.Update(ctx, testItemID(ItemCard, "card-name"), draft)

		require.Error(t, err)
	})

	t.Run("обновление файла", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		sourcePath := filepath.Join(t.TempDir(), "updated.bin")
		writeTestFile(t, sourcePath, []byte("updated binary"))
		adapter := newAuthorizedAdapter(client, filesDir)
		draft := Draft{Type: ItemBinary, Secret: sourcePath}

		client.EXPECT().
			FileUpload(ctx, "updated.bin", testMasterPassword, []byte("updated binary"), testToken).
			Return("updated.bin", nil)

		err := adapter.Update(ctx, "file:old.bin", draft)

		require.NoError(t, err)
		requireFileContent(t, filepath.Join(filesDir, "updated.bin"), "updated binary")
	})

	t.Run("неизвестный id", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.Update(ctx, "unknown:name", Draft{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported item id")
	})
}

func TestGRPCAdapterDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		err := adapter.Delete(ctx, testItemID(ItemText, "note"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "пользователь не авторизован")
	})

	t.Run("ошибка если плохой id", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.Delete(ctx, string(ItemText))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad item id")
	})

	t.Run("удаление текста", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())

		client.EXPECT().TextDelete(ctx, "note", testToken).Return(nil)

		err := adapter.Delete(ctx, testItemID(ItemText, "note"))

		require.NoError(t, err)
	})

	t.Run("удаление логина и пароля", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())

		client.EXPECT().CredsDelete(ctx, "creds", testToken).Return(nil)

		err := adapter.Delete(ctx, testItemID(ItemCreds, "creds"))

		require.NoError(t, err)
	})

	t.Run("удаление карты", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())

		client.EXPECT().CardDelete(ctx, "card", testToken).Return(nil)

		err := adapter.Delete(ctx, testItemID(ItemCard, "card"))

		require.NoError(t, err)
	})

	t.Run("удаление файла на сервере", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())

		client.EXPECT().FileRemove(ctx, "file.bin", testToken).Return(nil)

		err := adapter.Delete(ctx, "file:file.bin")

		require.NoError(t, err)
	})

	t.Run("ошибка клиента", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		wantErr := errors.New("delete failed")

		client.EXPECT().TextDelete(ctx, "note", testToken).Return(wantErr)

		err := adapter.Delete(ctx, testItemID(ItemText, "note"))

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("неизвестный тип", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.Delete(ctx, "unknown:name")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "неизветсный тип данных")
	})
}

func TestGRPCAdapterUploadFile(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		err := adapter.UploadFile(ctx, "file.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "пользователь не авторизован")
	})

	t.Run("ошибка чтения файла", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.UploadFile(ctx, filepath.Join(t.TempDir(), "missing.txt"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при чтении файла")
	})

	t.Run("ошибка клиента", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		sourcePath := filepath.Join(t.TempDir(), "file.txt")
		writeTestFile(t, sourcePath, []byte("content"))
		adapter := newAuthorizedAdapter(client, filesDir)
		wantErr := errors.New("upload failed")

		client.EXPECT().
			FileUpload(ctx, "file.txt", testMasterPassword, []byte("content"), testToken).
			Return("", wantErr)

		err := adapter.UploadFile(ctx, sourcePath)

		require.ErrorIs(t, err, wantErr)
		assert.NoFileExists(t, filepath.Join(filesDir, "file.txt"))
	})

	t.Run("успешная загрузка", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		sourcePath := filepath.Join(t.TempDir(), "file.txt")
		writeTestFile(t, sourcePath, []byte("content"))
		adapter := newAuthorizedAdapter(client, filesDir)

		client.EXPECT().
			FileUpload(ctx, "file.txt", testMasterPassword, []byte("content"), testToken).
			Return("file.txt", nil)

		err := adapter.UploadFile(ctx, sourcePath)

		require.NoError(t, err)
		requireFileContent(t, filepath.Join(filesDir, "file.txt"), "content")
	})
}

func TestGRPCAdapterDownloadFile(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		err := adapter.DownloadFile(ctx, "file.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "пользователь не авторизован")
	})

	t.Run("ошибка если имя пустое", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.DownloadFile(ctx, " ")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty file name")
	})

	t.Run("ошибка клиента", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		wantErr := errors.New("download failed")

		client.EXPECT().
			FileDownload(ctx, "file.txt", testMasterPassword, testToken).
			Return(nil, wantErr)

		err := adapter.DownloadFile(ctx, "file.txt")

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("ошибка создания директории", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := filepath.Join(t.TempDir(), "not-dir")
		writeTestFile(t, filesDir, []byte("regular file"))
		adapter := newAuthorizedAdapter(client, filesDir)

		client.EXPECT().
			FileDownload(ctx, "file.txt", testMasterPassword, testToken).
			Return([]byte("content"), nil)

		err := adapter.DownloadFile(ctx, "file.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при создании директории")
	})

	t.Run("успешное скачивание", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		adapter := newAuthorizedAdapter(client, filesDir)

		client.EXPECT().
			FileDownload(ctx, "file.txt", testMasterPassword, testToken).
			Return([]byte("downloaded content"), nil)

		err := adapter.DownloadFile(ctx, "../file.txt")

		require.NoError(t, err)
		requireFileContent(t, filepath.Join(filesDir, "file.txt"), "downloaded content")
	})
}

func TestGRPCAdapterRemoveFile(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		err := adapter.RemoveFile(ctx, "file.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "пользователь не авторизован")
	})

	t.Run("ошибка если имя пустое", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.RemoveFile(ctx, " ")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty file name")
	})

	t.Run("ошибка клиента", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		wantErr := errors.New("remove failed")

		client.EXPECT().FileRemove(ctx, "file.txt", testToken).Return(wantErr)

		err := adapter.RemoveFile(ctx, "file.txt")

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("успешное удаление на сервере", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())

		client.EXPECT().FileRemove(ctx, "file.txt", testToken).Return(nil)

		err := adapter.RemoveFile(ctx, "../file.txt")

		require.NoError(t, err)
	})
}

func TestGRPCAdapterDeleteLocalFile(t *testing.T) {
	ctx := context.Background()

	t.Run("ошибка если пользователь не авторизован", func(t *testing.T) {
		adapter := &GRPCAdapter{}

		err := adapter.DeleteLocalFile(ctx, "file.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "пользователь не авторизован")
	})

	t.Run("ошибка если имя пустое", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.DeleteLocalFile(ctx, " ")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty file name")
	})

	t.Run("ошибка если файла нет", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.DeleteLocalFile(ctx, "missing.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "локальный файл")
		assert.Contains(t, err.Error(), "не найден")
	})

	t.Run("успешное удаление", func(t *testing.T) {
		filesDir := t.TempDir()
		path := filepath.Join(filesDir, "file.txt")
		writeTestFile(t, path, []byte("content"))
		adapter := newAuthorizedAdapter(nil, filesDir)

		err := adapter.DeleteLocalFile(ctx, "../file.txt")

		require.NoError(t, err)
		assert.NoFileExists(t, path)
	})
}

func TestGRPCAdapterSyncFile(t *testing.T) {
	ctx := context.Background()

	t.Run("локальный файл загружается на сервер", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		path := filepath.Join(filesDir, "local.txt")
		writeTestFile(t, path, []byte("local"))
		adapter := newAuthorizedAdapter(client, filesDir)

		client.EXPECT().
			FileUpload(ctx, "local.txt", testMasterPassword, []byte("local"), testToken).
			Return("local.txt", nil)

		err := adapter.syncFile(ctx, FileSyncInfo{
			Name:      "local.txt",
			LocalPath: path,
			Status:    FileSyncStatusLocalOnly,
		})

		require.NoError(t, err)
	})

	t.Run("ошибка загрузки локального файла оборачивается", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		path := filepath.Join(filesDir, "local.txt")
		writeTestFile(t, path, []byte("local"))
		adapter := newAuthorizedAdapter(client, filesDir)
		wantErr := errors.New("upload failed")

		client.EXPECT().
			FileUpload(ctx, "local.txt", testMasterPassword, []byte("local"), testToken).
			Return("", wantErr)

		err := adapter.syncFile(ctx, FileSyncInfo{
			Name:      "local.txt",
			LocalPath: path,
			Status:    FileSyncStatusLocalOnly,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "загрузка локального файла")
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("удаленный файл скачивается локально", func(t *testing.T) {
		client := newMockGRPCClient(t)
		filesDir := t.TempDir()
		adapter := newAuthorizedAdapter(client, filesDir)

		client.EXPECT().
			FileDownload(ctx, "remote.txt", testMasterPassword, testToken).
			Return([]byte("remote"), nil)

		err := adapter.syncFile(ctx, FileSyncInfo{
			Name:   "remote.txt",
			Status: FileSyncStatusRemoteOnly,
		})

		require.NoError(t, err)
		requireFileContent(t, filepath.Join(filesDir, "remote.txt"), "remote")
	})

	t.Run("ошибка скачивания удаленного файла оборачивается", func(t *testing.T) {
		client := newMockGRPCClient(t)
		adapter := newAuthorizedAdapter(client, t.TempDir())
		wantErr := errors.New("download failed")

		client.EXPECT().
			FileDownload(ctx, "remote.txt", testMasterPassword, testToken).
			Return(nil, wantErr)

		err := adapter.syncFile(ctx, FileSyncInfo{
			Name:   "remote.txt",
			Status: FileSyncStatusRemoteOnly,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "скачивание файла с сервера")
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("синхронизированный файл пропускается", func(t *testing.T) {
		adapter := newAuthorizedAdapter(nil, t.TempDir())

		err := adapter.syncFile(ctx, FileSyncInfo{
			Name:   "synced.txt",
			Status: FileSyncStatusSynced,
		})

		require.NoError(t, err)
	})
}

func TestGRPCAdapterSaveLocalFileCopy(t *testing.T) {
	t.Run("копирует файл в локальное хранилище", func(t *testing.T) {
		filesDir := t.TempDir()
		sourcePath := filepath.Join(t.TempDir(), "source.txt")
		writeTestFile(t, sourcePath, []byte("content"))
		adapter := &GRPCAdapter{filesDir: filesDir}

		err := adapter.saveLocalFileCopy(sourcePath, "copy.txt")

		require.NoError(t, err)
		requireFileContent(t, filepath.Join(filesDir, "copy.txt"), "content")
	})

	t.Run("ничего не делает если source и destination совпадают", func(t *testing.T) {
		filesDir := t.TempDir()
		path := filepath.Join(filesDir, "file.txt")
		writeTestFile(t, path, []byte("content"))
		adapter := &GRPCAdapter{filesDir: filesDir}

		err := adapter.saveLocalFileCopy(path, "file.txt")

		require.NoError(t, err)
		requireFileContent(t, path, "content")
	})

	t.Run("ошибка создания директории", func(t *testing.T) {
		filesDir := filepath.Join(t.TempDir(), "not-dir")
		writeTestFile(t, filesDir, []byte("regular file"))
		adapter := &GRPCAdapter{filesDir: filesDir}

		err := adapter.saveLocalFileCopy(filepath.Join(t.TempDir(), "source.txt"), "copy.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при создании директории")
	})
}

func TestGRPCAdapterFileStatuses(t *testing.T) {
	filesDir := t.TempDir()
	writeTestFile(t, filepath.Join(filesDir, "local-only.txt"), []byte("local"))
	writeTestFile(t, filepath.Join(filesDir, "synced.txt"), []byte("synced"))
	remoteCreatedAt := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	adapter := &GRPCAdapter{filesDir: filesDir}

	files, err := adapter.fileStatuses(map[string]time.Time{
		"remote-only.txt": remoteCreatedAt,
		"synced.txt":      remoteCreatedAt,
	})

	require.NoError(t, err)
	require.Len(t, files, 3)

	byName := make(map[string]FileSyncInfo, len(files))
	for _, file := range files {
		byName[file.Name] = file
	}

	assert.Equal(t, FileSyncStatusLocalOnly, byName["local-only.txt"].Status)
	assert.Equal(t, FileSyncStatusRemoteOnly, byName["remote-only.txt"].Status)
	assert.Equal(t, FileSyncStatusSynced, byName["synced.txt"].Status)
	assert.Equal(t, remoteCreatedAt, byName["remote-only.txt"].RemoteCreatedAt)
	assert.Equal(t, remoteCreatedAt, byName["synced.txt"].RemoteCreatedAt)
	assert.NotEmpty(t, byName["local-only.txt"].LocalPath)
}

func TestSortFileStatuses(t *testing.T) {
	files := []FileSyncInfo{
		{Name: "b.txt", Status: FileSyncStatusLocalOnly},
		{Name: "a.txt", Status: FileSyncStatusLocalOnly},
	}

	sortFileStatuses(files)

	assert.Equal(t, "a.txt", files[0].Name)
	assert.Equal(t, "b.txt", files[1].Name)
}

func TestScanLocalFiles(t *testing.T) {
	t.Run("возвращает пустоту если директории нет", func(t *testing.T) {
		files, err := scanLocalFiles(filepath.Join(t.TempDir(), "missing"))

		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("читает только файлы верхнего уровня", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "file.txt"), []byte("content"))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0o700))
		writeTestFile(t, filepath.Join(dir, "nested", "ignored.txt"), []byte("ignored"))

		files, err := scanLocalFiles(dir)

		require.NoError(t, err)
		require.Len(t, files, 1)

		file := files["file.txt"]
		assert.Equal(t, "file.txt", file.Name)
		assert.Equal(t, filepath.Join(dir, "file.txt"), file.LocalPath)
		assert.Equal(t, int64(len("content")), file.Size)
		assert.Equal(t, FileSyncStatusLocalOnly, file.Status)
		assert.False(t, file.LocalUpdatedAt.IsZero())
	})

	t.Run("ошибка чтения директории", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "not-dir")
		writeTestFile(t, path, []byte("regular file"))

		files, err := scanLocalFiles(path)

		require.Error(t, err)
		assert.Nil(t, files)
		assert.Contains(t, err.Error(), "read files dir")
	})
}

func TestNormalizedFileName(t *testing.T) {
	t.Run("валидное имя", func(t *testing.T) {
		name, err := normalizedFileName("../file.txt")

		require.NoError(t, err)
		assert.Equal(t, "file.txt", name)
	})

	t.Run("пустое имя", func(t *testing.T) {
		name, err := normalizedFileName(" ")

		require.Error(t, err)
		assert.Empty(t, name)
		assert.Contains(t, err.Error(), "empty file name")
	})
}

func TestSafeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "обычное имя", input: "file.txt", expected: "file.txt"},
		{name: "пробелы по краям", input: "  file.txt  ", expected: "file.txt"},
		{name: "путь до файла", input: filepath.Join("some", "dir", "file.txt"), expected: "file.txt"},
		{name: "выход из директории", input: "../secret.txt", expected: "secret.txt"},
		{name: "пустая строка", input: "", expected: ""},
		{name: "строка из пробелов", input: "   ", expected: ""},
		{name: "текущая директория", input: ".", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := safeFileName(tt.input)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Run("успешное копирование", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeTestFile(t, src, []byte("content"))

		err := copyFile(src, dst)

		require.NoError(t, err)
		requireFileContent(t, dst, "content")
	})

	t.Run("ошибка открытия source", func(t *testing.T) {
		err := copyFile(filepath.Join(t.TempDir(), "missing.txt"), filepath.Join(t.TempDir(), "dst.txt"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "open source file")
	})

	t.Run("ошибка открытия destination", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		writeTestFile(t, src, []byte("content"))

		err := copyFile(src, dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "open destination file")
	})
}
