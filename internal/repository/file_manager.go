package repository

import (
	"os"
	"strconv"
)

const (
	dirPerm  os.FileMode = 0700
	filePerm os.FileMode = 0644
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
	root, err := f.openStorageRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	return root.MkdirAll(strconv.FormatInt(id, 10), dirPerm)
}

// UploadFile - создание загруженного с сервера файла.
func (f *FileManager) UploadFile(id int64, name string, data []byte) error {
	root, err := f.openUserRoot(id)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.WriteFile(name, data, filePerm)
}

// DownloadFile - чтение файла для дальнейшей загрузке на сервер.
func (f *FileManager) DownloadFile(id int64, name string) ([]byte, error) {
	root, err := f.openUserRoot(id)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(name)
}

// RemoveFile - удаление файла на локальном хранилище.
func (f *FileManager) RemoveFile(id int64, name string) error {
	root, err := f.openUserRoot(id)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.Remove(name)
}

func (f *FileManager) openStorageRoot() (*os.Root, error) {
	if err := os.MkdirAll(f.pathDir, dirPerm); err != nil {
		return nil, err
	}
	return os.OpenRoot(f.pathDir)
}

func (f *FileManager) openUserRoot(id int64) (*os.Root, error) {
	root, err := f.openStorageRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.OpenRoot(strconv.FormatInt(id, 10))
}
