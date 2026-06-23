package models

import (
	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

type User struct {
	ID        int64
	Username  string
	Password  string
	CreatedAt timestamp.Timestamp
	UpdatedAt timestamp.Timestamp
}

type GetAllUsers struct {
	ID       int64
	Username string
	Password string
}

type UserRequest struct {
	Username string
	Password string
}

// GetUserData - конвертация данных пользователя в grpc данные.
func GetUserData(data *User) *pb.User {
	return pb.User_builder{
		UserId:    data.ID,
		Username:  data.Username,
		CreatedAt: &data.CreatedAt,
		UpdatedAt: &data.UpdatedAt,
	}.Build()
}
