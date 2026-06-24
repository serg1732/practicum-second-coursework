package repository

import "errors"

var (
	ErrFileNotExists           = errors.New("файл не существует")
	ErrRecordNotFound          = errors.New("запись не найдена")
	ErrNameAlreadyExists       = errors.New("имя уже существует")
	ErrNotValidateToken        = errors.New("невалидный токен")
	ErrNoMetadataSet           = errors.New("не удалось задать metadata")
	ErrUsernameAlreadyExists   = errors.New("пользователь уже существует")
	ErrWrongUsernameOrPassword = errors.New("неверный логин или пароль")
)
