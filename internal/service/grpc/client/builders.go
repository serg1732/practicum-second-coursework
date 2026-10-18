package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/serg1732/practicum-second-coursework/internal/models"
	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/utils"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

type binaryNode interface {
	GetName() string
	GetCreatedAt() *timestamp.Timestamp
}

func buildSyncedText(entity decodedEntity) (models.SyncedText, error) {
	return models.SyncedText{
		Name:        entity.Metadata.Name,
		Description: entity.Metadata.Description,
		Data:        entity.Plaintext,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}, nil
}

func buildSyncedCard(entity decodedEntity) (models.SyncedCard, error) {
	var card models.Card
	if err := json.Unmarshal([]byte(entity.Plaintext), &card); err != nil {
		return models.SyncedCard{}, fmt.Errorf("ошибка при конвертации данных карты: %w", err)
	}

	return models.SyncedCard{
		Name:          entity.Metadata.Name,
		Description:   entity.Metadata.Description,
		PaymentSystem: card.PaymentSystem,
		Number:        card.Number,
		Holder:        card.Holder,
		CVC:           card.CVC,
		EndDate:       card.EndDate,
		CreatedAt:     entity.CreatedAt,
		UpdatedAt:     entity.UpdatedAt,
	}, nil
}

func buildSyncedCreds(entity decodedEntity) (models.SyncedCreds, error) {
	var creds models.Creds
	if err := json.Unmarshal([]byte(entity.Plaintext), &creds); err != nil {
		return models.SyncedCreds{}, fmt.Errorf("ошибка при конвертации логина и пароля: %w", err)
	}

	return models.SyncedCreds{
		Name:        entity.Metadata.Name,
		Description: entity.Metadata.Description,
		Login:       creds.Login,
		Password:    creds.Password,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}, nil
}

func buildSyncedBinaryList[N binaryNode](nodes []N) ([]models.SyncedBinary, error) {
	items := make([]models.SyncedBinary, 0, len(nodes))

	for _, node := range nodes {
		createdAt, err := utils.ConvertTimestampToTime(node.GetCreatedAt())
		if err != nil {
			return items, fmt.Errorf("ошибка при конвертации времени создания файла: %w", err)
		}

		items = append(items, models.SyncedBinary{
			Name:      node.GetName(),
			CreatedAt: createdAt,
		})
	}
	return items, nil
}

type entitySyncSpec struct {
	entityType Types
	sync       func(ctx context.Context, entityType Types, accessToken *pb.Token, secretKey []byte) error
}

// buildAccessToken - создание токена для gRPC-запросов.
func buildAccessToken(token models.Token) *pb.Token {
	return pb.Token_builder{
		Token:  token.AccessToken,
		UserId: token.UserID,
		Iat:    timestamp.New(token.Iat),
		Exp:    timestamp.New(token.Exp),
	}.Build()
}
