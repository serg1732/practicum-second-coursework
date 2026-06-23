package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/models"
	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

type Types string

const (
	LayoutDate string = "02.01.2006"

	LoginPassword Types = "Данные логин пароля"
	Text          Types = "Текстовые данные"
	Card          Types = "Данные банковской карты"
)

func (t Types) ToString() string {
	return string(t)
}

// Client - GRPC клиент.
type Client struct {
	Cfg  *config.GophKeeperClientConfig
	grpc pb.GophKeeperClient
	log  *slog.Logger
	pb.UnimplementedGophKeeperServer
}

// BuildGRPCClient - создание GRPC клиента.
func BuildGRPCClient(logger *slog.Logger, config *config.GophKeeperClientConfig, creds credentials.TransportCredentials) (*Client, error) {
	client := Client{Cfg: config, log: logger}
	conn, err := grpc.NewClient(
		config.GRPCRemoteAddr,
		grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Error("ошибка инициализации GRPC клиента", "error", err)
		return nil, fmt.Errorf("ошибка инициализации GRPC клиента: %w", err)
	}
	client.grpc = pb.NewGophKeeperClient(conn)
	return &client, nil
}

// Ping - проверка соединения в GRPC сервером
func (c Client) Ping(ctx context.Context) (string, error) {
	c.log.Debug("Запрос на проверку соединения с GRPC сервером")
	msg, err := c.grpc.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		c.log.Error("ошибка при попытки проверить соединение с GRPC сервером", "error", err)
		return "", err
	}
	return msg.GetMessage(), nil
}

// Registration - регистрация нового пользователя.
func (c Client) Registration(ctx context.Context, username, password string) (models.Token, error) {
	c.log.Debug("Регистрация пользователя")
	token := models.Token{}
	password, err := utils.HashPassword(password)
	if err != nil {
		c.log.Error("ошибка при регистрации нового пользователя", "error", err)
		return token, err
	}
	registeredUser, err := c.grpc.Registration(ctx,
		pb.RegistrationRequest_builder{Username: username, Password: password}.Build())
	if err != nil {
		c.log.Error("ошибка при регистрации пользователя", "error", err)
		return token, err
	}
	createdToken, err := utils.ConvertTimestampToTime(registeredUser.GetAccessToken().GetIat())
	if err != nil {
		c.log.Error("ошибка при конвертации времени создания токена", "error", err)
		return token, err
	}
	expToken, err := utils.ConvertTimestampToTime(registeredUser.GetAccessToken().GetExp())
	if err != nil {
		c.log.Error("ошибка при конвертации времени истечения срока токена", "error", err)
		return token, err
	}

	token = models.Token{AccessToken: registeredUser.GetAccessToken().GetToken(), UserID: registeredUser.GetAccessToken().GetUserId(),
		Iat: createdToken, Exp: expToken}
	return token, nil
}

// Authentication - авторизация пользователя.
func (c Client) Authentication(ctx context.Context, username, password string) (models.Token, error) {
	c.log.Debug("Авторизация пользователя")
	token := models.Token{}
	password, err := utils.HashPassword(password)
	if err != nil {
		c.log.Error("ошибка получения hash пароля", "error", err)
		return token, err
	}
	authenticatedUser, err := c.grpc.Authentication(ctx, pb.AuthenticationRequest_builder{
		Username: username,
		Password: password,
	}.Build())
	if err != nil {
		c.log.Error("ошибка авторизации пользователя", "error", err)
		return token, err
	}

	createdToken, err := utils.ConvertTimestampToTime(authenticatedUser.GetAccessToken().GetIat())
	if err != nil {
		c.log.Error("ошибка при конвертации времени создания токена", "error", err)
		return token, err
	}
	expToken, err := utils.ConvertTimestampToTime(authenticatedUser.GetAccessToken().GetExp())
	if err != nil {
		c.log.Error("ошибка при конвертации времени истечения срока токена", "error", err)
		return token, err
	}
	token = models.Token{
		AccessToken: authenticatedUser.GetAccessToken().GetToken(),
		UserID:      authenticatedUser.GetAccessToken().GetUserId(),
		Iat:         createdToken, Exp: expToken}

	return token, nil
}

// UserExist - проверяет, есть ли такой аккаунт.
func (c Client) UserExist(ctx context.Context, username string) (bool, error) {
	c.log.Debug("Проверка существование пользователя")
	user, err := c.grpc.UserExist(ctx, pb.UserExistRequest_builder{Username: username}.Build())
	if err != nil {
		c.log.Error("ошибка при проверке существования аккаунта", "error", err)
		return user.GetExist(), err
	}
	return user.GetExist(), nil
}

// CardCreate - создание записи карты.
func (c Client) CardCreate(ctx context.Context, name, description, password, paymentSystem, number, holder, cvc, endDate string, token models.Token) error {
	c.log.Debug("Создание записи карты")
	intCvc, err := strconv.Atoi(cvc)
	if err != nil {
		c.log.Error("ошибка при конвертации cvc", "error", err)
		return err
	}
	timeEndDate, err := time.Parse(LayoutDate, endDate)
	if err != nil {
		c.log.Error("ошибка при конвертации даты окончания карты", "error", err)
		return err
	}
	card := models.Card{Name: name, Description: description, PaymentSystem: paymentSystem, Number: number, Holder: holder, EndDate: timeEndDate, CVC: intCvc}
	jsonCard, err := json.Marshal(card)
	if err != nil {
		c.log.Error("ошибка при конвертации данных карты в JSON", "error", err)
		return err
	}

	secretKey := utils.Pbkdf2KeySecureRandom([]byte(password))
	encryptCard, err := utils.Encrypt(string(jsonCard), secretKey)
	if err != nil {
		c.log.Error("ошибка при шифровании данных", "error", err)
		return err
	}
	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	metadata := models.MetadataEntity{Name: name, Description: description, Type: Card.ToString()}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if _, err = c.grpc.EntityCreate(ctx,
		pb.CreateEntityRequest_builder{
			Data:     []byte(encryptCard),
			Metadata: string(jsonMetadata),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при записи данных в БД")
		return err
	}
	return nil
}

// CardDelete - удаление данных карты.
func (c Client) CardDelete(ctx context.Context, card string, token models.Token) error {
	c.log.Debug("Удаление данных карты")
	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	if _, err := c.grpc.EntityDelete(ctx,
		pb.DeleteEntityRequest_builder{
			Name: card,
			Type: Card.ToString(),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при удалении данных карт", "error", err)
		return err
	}
	return nil
}

// CardUpdate - обновление данных карты.
func (c Client) CardUpdate(ctx context.Context, name, passwordSecure, paymentSystem, number, holder, cvc, endDateCard string, token models.Token) error {
	c.log.Debug("Обновление данных карты")
	parsedCvc, err := strconv.Atoi(cvc)
	if err != nil {
		c.log.Error("ошибка конвертации cvc", "error", err)
		return err
	}
	timeEndDate, err := time.Parse(LayoutDate, endDateCard)
	if err != nil {
		c.log.Error("ошибка конвертации даты срока действия карты", "error", err)
		return err
	}
	card := models.Card{PaymentSystem: paymentSystem, Number: number, Holder: holder, CVC: parsedCvc, EndDate: timeEndDate}
	jsonCard, err := json.Marshal(card)
	if err != nil {
		c.log.Error("ошибка преобразования данных карты в JSON", "error", err)
		return err
	}

	secretKey := utils.Pbkdf2KeySecureRandom([]byte(passwordSecure))
	encryptCard, err := utils.Encrypt(string(jsonCard), secretKey)
	if err != nil {
		c.log.Error("ошибка шифрования данных", "error", err)

		return err
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	if _, err = c.grpc.EntityUpdate(ctx,
		pb.UpdateEntityRequest_builder{
			Name: name,
			Data: []byte(encryptCard),
			Type: Card.ToString(),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("Ошибка обновления данных", "error", err)
		return err
	}

	return nil
}

// TextCreate - добавление текста.
func (c Client) TextCreate(ctx context.Context, name, description, password, plaintext string, token models.Token) error {
	c.log.Debug("Добавление текста")
	secretKey := utils.Pbkdf2KeySecureRandom([]byte(password))
	encryptText, err := utils.Encrypt(plaintext, secretKey)
	if err != nil {
		c.log.Error("ошибка при шифровании данных", "error", err)
		return err
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)
	metadata := models.MetadataEntity{
		Name:        name,
		Description: description,
		Type:        Text.ToString(),
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	if _, err = c.grpc.EntityCreate(ctx,
		pb.CreateEntityRequest_builder{
			Data:     []byte(encryptText),
			Metadata: string(jsonMetadata),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при создании текста", "error", err)
		return err
	}
	return nil
}

// TextDelete - удаление текста.
func (c Client) TextDelete(ctx context.Context, text string, token models.Token) error {
	c.log.Debug("Удаление текста")
	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	if _, err := c.grpc.EntityDelete(ctx,
		pb.DeleteEntityRequest_builder{
			Name: text,
			Type: Text.ToString(),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при удалении данных", "error", err)
		return err
	}
	return nil
}

// TextUpdate - обновление текстовой информации.
func (c Client) TextUpdate(ctx context.Context, name, passwordSecure, text string, token models.Token) error {
	c.log.Debug("Обновление текстовой информации")
	secretKey := utils.Pbkdf2KeySecureRandom([]byte(passwordSecure))
	encryptText, err := utils.Encrypt(text, secretKey)
	if err != nil {
		c.log.Error("ошибка при шифровании данных", "error", err)
		return err
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	if _, err = c.grpc.EntityUpdate(ctx,
		pb.UpdateEntityRequest_builder{
			Name: name,
			Data: []byte(encryptText),
			Type: Text.ToString(),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при обновлении данных", "error", err)
		return err
	}
	return nil
}

// CredsCreate - сохранение данных логина и пароля.
func (c Client) CredsCreate(ctx context.Context, name, description, passwordSecure, login, password string, token models.Token) error {
	c.log.Debug("создание записи с логином и паролем")
	creds := models.Creds{
		Login:    login,
		Password: password,
	}
	jsonCreds, err := json.Marshal(creds)
	if err != nil {
		c.log.Error("ошибка при получении JSON логин и пароля", "error", err)
		return err
	}

	secretKey := utils.Pbkdf2KeySecureRandom([]byte(passwordSecure))
	encryptLoginPassword, err := utils.Encrypt(string(jsonCreds), secretKey)
	if err != nil {
		c.log.Error("ошибка при шифровании данных", "error", err)
		return err
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	metadata := models.MetadataEntity{
		Name:        name,
		Description: description,
		Type:        LoginPassword.ToString(),
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	if _, err = c.grpc.EntityCreate(ctx,
		pb.CreateEntityRequest_builder{
			Data:     []byte(encryptLoginPassword),
			Metadata: string(jsonMetadata),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при создании записи с логином и паролем", "error", err)
		return err
	}
	return nil
}

// CredsDelete - удаление данных логин и пароля.
func (c Client) CredsDelete(ctx context.Context, creds string, token models.Token) error {
	c.log.Debug("Удаление записи с логином и паролем")
	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	if _, err := c.grpc.EntityDelete(ctx,
		pb.DeleteEntityRequest_builder{
			Name: creds,
			Type: LoginPassword.ToString(),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при удалении записи с логином и паролем", "error", err)
		return err
	}
	return nil
}

// CredsUpdate - обновление сохраненных данных логин и пароля.
func (c Client) CredsUpdate(ctx context.Context, name, passwordSecure, login, password string, token models.Token) error {
	c.log.Debug("Обновление данных логина и пароля")
	creds := models.Creds{
		Login:    login,
		Password: password,
	}
	jsonLoginPassword, err := json.Marshal(creds)
	if err != nil {
		c.log.Error("ошибка при конвертации логин и пароля", "error", err)
		return err
	}

	secretKey := utils.Pbkdf2KeySecureRandom([]byte(passwordSecure))
	encryptLoginPassword, err := utils.Encrypt(string(jsonLoginPassword), secretKey)
	if err != nil {
		c.log.Error("ошибка при шифровании данных", "error", err)
		return err
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	if _, err = c.grpc.EntityUpdate(ctx,
		pb.UpdateEntityRequest_builder{
			Name: name,
			Data: []byte(encryptLoginPassword),
			Type: LoginPassword.ToString(),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("Ошибка при обновлении данных логина и пароля", "error", err)
		return err
	}
	return nil
}

// FileUpload - загрузка файла на сервер.
func (c Client) FileUpload(ctx context.Context, name, password string, file []byte, token models.Token) (string, error) {
	c.log.Debug("Загрузка файла с сервера")
	secretKey := utils.Pbkdf2KeySecureRandom([]byte(password))
	encryptFile, err := utils.Encrypt(string(file), secretKey)
	if err != nil {
		c.log.Error("ошибка при шифровании данных")
		return "", err
	}

	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	uploadFile, err := c.grpc.FileUpload(ctx,
		pb.UploadBinaryRequest_builder{
			Name: name,
			Data: []byte(encryptFile),
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build())
	if err != nil {
		c.log.Error("ошибка при загрузке файла")
		return "", err
	}
	return uploadFile.GetName(), nil
}

// FileDownload - скачивание файла
func (c Client) FileDownload(ctx context.Context, name, password string, token models.Token) ([]byte, error) {
	c.log.Debug("Скачивание файла с сервера")
	secretKey := utils.Pbkdf2KeySecureRandom([]byte(password))
	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	downloadedFile, err := c.grpc.FileDownload(ctx,
		pb.DownloadBinaryRequest_builder{
			Name: name,
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build())
	if err != nil {
		c.log.Error("ошибка при скачивании файла с сервера", "error", err)
		return nil, err
	}

	file, err := utils.Decrypt(string(downloadedFile.GetData()), secretKey)
	if err != nil {
		c.log.Error("ошибка при расшифровке файла с сервера", "error", err)
		return nil, err
	}
	return []byte(file), nil
}

// FileRemove - удаление файла
func (c Client) FileRemove(ctx context.Context, binary string, token models.Token) error {
	c.log.Debug("Удаление файла на сервере")
	createdToken := timestamp.New(token.Iat)
	expToken := timestamp.New(token.Exp)

	if _, err := c.grpc.FileRemove(ctx,
		pb.DeleteBinaryRequest_builder{
			Name: binary,
			AccessToken: pb.Token_builder{
				Token:  token.AccessToken,
				UserId: token.UserID,
				Iat:    createdToken,
				Exp:    expToken,
			}.Build(),
		}.Build()); err != nil {
		c.log.Error("ошибка при удалении файла с сервера", "error", err)
		return err
	}
	return nil
}

// Synchronize - синхронизация данных с сервером.
func (c Client) Synchronize(
	ctx context.Context,
	password string,
	token models.Token,
) (models.SyncResult, error) {
	var result models.SyncResult

	secretKey := utils.Pbkdf2KeySecureRandom([]byte(password))
	accessToken := buildAccessToken(token)

	entitySpecs := []entitySyncSpec{
		{
			entityType: Text,
			sync: func(ctx context.Context, entityType Types, accessToken *pb.Token, secretKey []byte) error {
				items, err := loadEntities(ctx, c.grpc, entityType, accessToken, secretKey, buildSyncedText)
				result.Text = items
				return err
			},
		},
		{
			entityType: Card,
			sync: func(ctx context.Context, entityType Types, accessToken *pb.Token, secretKey []byte) error {
				items, err := loadEntities(ctx, c.grpc, entityType, accessToken, secretKey, buildSyncedCard)
				result.Card = items
				return err
			},
		},
		{
			entityType: LoginPassword,
			sync: func(ctx context.Context, entityType Types, accessToken *pb.Token, secretKey []byte) error {
				items, err := loadEntities(ctx, c.grpc, entityType, accessToken, secretKey, buildSyncedCreds)
				result.Creds = items
				return err
			},
		},
	}

	for _, spec := range entitySpecs {
		if err := spec.sync(ctx, spec.entityType, accessToken, secretKey); err != nil {
			return result, err
		}
	}

	response, err := c.grpc.FileGetList(ctx, pb.GetListBinaryRequest_builder{
		AccessToken: accessToken,
	}.Build())

	if err != nil {
		return result, err
	}

	binary, err := buildSyncedBinaryList(response.GetNode())
	if err != nil {
		return result, err
	}
	result.Binary = binary

	return result, nil
}
