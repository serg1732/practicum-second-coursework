package repository

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type FileManager struct {
	pathDir string
}

// BuildFileManager - создание менеджера по работе с локальным файлами.
func BuildFileManager(dir string) *FileManager {
	return &FileManager{pathDir: dir}
}

// CreateStorageUser - создание локального хранилища (папок).
func (f *FileManager) CreateStorageUser(id int64) error {
	userID := strconv.Itoa(int(id))
	path := filepath.Join(f.pathDir, userID)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// UploadFile - создание загруженного с сервера файла.
func (f *FileManager) UploadFile(id int64, name string, data []byte) error {
	userID := strconv.Itoa(int(id))
	path := filepath.Join(f.pathDir, userID, "/", name)
	err := os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// DownloadFile - чтение файла для дальнейшей загрузке на сервер.
func (f *FileManager) DownloadFile(id int64, name string) ([]byte, error) {
	userID := strconv.Itoa(int(id))
	path := filepath.Join(f.pathDir, userID, "/", name)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// RemoveFile - удаление файла на локальном хранилище.
func (f *FileManager) RemoveFile(id int64, name string) error {
	userID := strconv.Itoa(int(id))
	path := filepath.Join(f.pathDir, userID, "/", name)
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}
