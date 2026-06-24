package models

import (
	"time"

	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

type File struct {
	ID        int64
	UserID    int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FileRequest struct {
	UserID      int64
	Name        string
	AccessToken string
}

// GetListFile - получение списка файлов в grpc формате.
func GetListFile(binary []File) []*pb.Binary {
	items := make([]*pb.Binary, len(binary))
	for i := range binary {
		created := timestamp.New(binary[i].CreatedAt)
		items[i] = pb.Binary_builder{
			Id:        binary[i].ID,
			Name:      binary[i].Name,
			CreatedAt: created,
		}.Build()
	}
	return items
}
