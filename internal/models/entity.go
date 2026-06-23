package models

import (
	"encoding/json"
	"time"

	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

type Entity struct {
	ID        int64
	UserID    int64
	Data      []byte
	Metadata  MetadataEntity
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MetadataEntity struct {
	Name        string
	Description string
	Type        string
}

type CreateEntityRequest struct {
	UserID      int64
	Data        []byte
	Metadata    MetadataEntity
	AccessToken string
}

// GetListEntity - получение списка сущностей в grpc формате.
func GetListEntity(data []Entity) ([]*pb.Entity, error) {
	items := make([]*pb.Entity, len(data))
	for i := range data {
		jsonMetadata, err := json.Marshal(data[i].Metadata)
		if err != nil {
			return items, err
		}
		created := timestamp.New(data[i].CreatedAt)
		updated := timestamp.New(data[i].UpdatedAt)
		items[i] = pb.Entity_builder{
			Id:       data[i].ID,
			UserId:   data[i].UserID,
			Data:     data[i].Data,
			Metadata: string(jsonMetadata), CreatedAt: created, UpdatedAt: updated,
		}.Build()
	}
	return items, nil
}
