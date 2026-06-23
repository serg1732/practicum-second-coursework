package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/serg1732/practicum-second-coursework/internal/models"
	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/utils"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

type decodedEntity struct {
	Metadata  models.MetadataEntity
	Plaintext string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type entityNode interface {
	GetData() []byte
	GetMetadata() string
	GetCreatedAt() *timestamp.Timestamp
	GetUpdatedAt() *timestamp.Timestamp
}

type decodedEntityMapper[T any] func(entity decodedEntity) (T, error)

// loadEntities - получает сущности одного типа, расшифровывает их и приводит к нужной модели.
func loadEntities[T any](
	ctx context.Context,
	grpcClient pb.GophKeeperClient,
	entityType Types,
	accessToken *pb.Token,
	secretKey []byte,
	mapper decodedEntityMapper[T],
) ([]T, error) {
	response, err := grpcClient.EntityGetList(ctx, pb.GetListEntityRequest_builder{
		Type:        entityType.ToString(),
		AccessToken: accessToken,
	}.Build())
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка секретов: %w", err)
	}

	return decodeEntities(response.GetNode(), secretKey, mapper)
}

// decodeEntities - преобразует список proto-сущностей в клиентские модели.
func decodeEntities[N entityNode, T any](
	nodes []N,
	secretKey []byte,
	mapper decodedEntityMapper[T],
) ([]T, error) {
	items := make([]T, 0, len(nodes))

	for _, node := range nodes {
		entity, err := decodeEntity(node, secretKey)
		if err != nil {
			return items, err
		}

		item, err := mapper(entity)
		if err != nil {
			return items, err
		}

		items = append(items, item)
	}
	return items, nil
}

// decodeEntity - расшифровывает данные сущности, metadata и даты.
func decodeEntity[N entityNode](
	node N,
	secretKey []byte,
) (decodedEntity, error) {
	plaintext, err := utils.Decrypt(string(node.GetData()), secretKey)
	if err != nil {
		return decodedEntity{}, fmt.Errorf("ошибка при расшифровке данных: %w", err)
	}

	var metadata models.MetadataEntity
	if err = json.Unmarshal([]byte(node.GetMetadata()), &metadata); err != nil {
		return decodedEntity{}, fmt.Errorf("ошибка при получении metadata: %w", err)
	}

	createdAt, err := utils.ConvertTimestampToTime(node.GetCreatedAt())
	if err != nil {
		return decodedEntity{}, fmt.Errorf("ошибка при конвертации времени создания записи: %w", err)
	}

	updatedAt, err := utils.ConvertTimestampToTime(node.GetUpdatedAt())
	if err != nil {
		return decodedEntity{}, fmt.Errorf("ошибка при конвертации времени обновления записи: %w", err)
	}

	return decodedEntity{
		Metadata:  metadata,
		Plaintext: plaintext,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
