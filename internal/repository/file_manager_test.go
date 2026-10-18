package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileManagerCreateStorageUserCreateUserFolder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := BuildFileManager(dir)

	err := manager.CreateStorageUser(42)
	assert.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, "42"))
	assert.NoError(t, err)
	assert.Equal(t, true, info.IsDir())
}

func TestFileManagerUploadFileCreateFileWithData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := BuildFileManager(dir)
	userID := int64(42)
	fileName := "data.txt"
	wantData := []byte("test data")

	assert.NoError(t, manager.CreateStorageUser(userID))

	err := manager.UploadFile(userID, fileName, wantData)
	assert.NoError(t, err)

	gotData, err := os.ReadFile(filepath.Join(dir, "42", fileName))
	assert.NoError(t, err)
	assert.Equal(t, string(wantData), string(gotData))
}

func TestFileManagerUploadFileErrorUserFolderNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := BuildFileManager(dir)

	err := manager.UploadFile(42, "data.txt", []byte("test data"))
	assert.Error(t, err)
}

func TestFileManagerDownloadFileReadFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := BuildFileManager(dir)
	userID := int64(42)
	fileName := "data.txt"
	wantData := []byte("test data")

	assert.NoError(t, manager.CreateStorageUser(userID))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "42", fileName), wantData, 0644))

	gotData, err := manager.DownloadFile(userID, fileName)
	assert.NoError(t, err)
	assert.Equal(t, string(wantData), string(gotData))
}

func TestFileManagerDownloadFileErrorFileNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := BuildFileManager(dir)

	gotData, err := manager.DownloadFile(42, "missing.txt")
	assert.Error(t, err)
	assert.Equal(t, true, gotData == nil)
}

func TestFileManagerRemoveFileRemoveFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := BuildFileManager(dir)
	userID := int64(42)
	fileName := "data.txt"
	filePath := filepath.Join(dir, "42", fileName)

	assert.NoError(t, manager.CreateStorageUser(userID))
	assert.NoError(t, os.WriteFile(filePath, []byte("test data"), 0644))

	err := manager.RemoveFile(userID, fileName)
	assert.NoError(t, err)

	_, err = os.Stat(filePath)
	assert.Equal(t, true, errors.Is(err, os.ErrNotExist))
}

func TestFileManagerRemoveFileErrorFileNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := BuildFileManager(dir)

	err := manager.RemoveFile(42, "missing.txt")
	assert.Error(t, err)
}
