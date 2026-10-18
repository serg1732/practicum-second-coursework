package grpc_handler

import (
	"context"
	"encoding/json"

	"github.com/serg1732/practicum-second-coursework/internal/models"
	grpc "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EntityGetList - обработчик получения списка данных на сервере (кроме файлов).
func (h *Handler) EntityGetList(ctx context.Context, req *grpc.GetListEntityRequest) (*grpc.GetListEntityResponse, error) {
	h.log.Debug("Запрос на получение списка данных")
	expToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении данных о токене", "error", err)
		return grpc.GetListEntityResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(expToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.GetListEntityResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	ListEntity, err := h.entity.GetList(ctx, req.GetAccessToken().GetUserId(), req.GetType())
	if err != nil {
		h.log.Error("ошибка при получении данных из БД", "error", err)
		return grpc.GetListEntityResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	list, err := models.GetListEntity(ListEntity)
	if err != nil {
		h.log.Error("ошибка при парсинге данных из БД", "error", err)
		return grpc.GetListEntityResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.GetListEntityResponse_builder{Node: list}.Build(), nil
}

// EntityCreate - обработчик сохранения записи.
func (h *Handler) EntityCreate(ctx context.Context, req *grpc.CreateEntityRequest) (*grpc.CreateEntityResponse, error) {
	h.log.Debug("Запрос на сохранение данных")
	expToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении токена", "error", err)
		return grpc.CreateEntityResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(expToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.CreateEntityResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	var metadata models.MetadataEntity
	if err = json.Unmarshal([]byte(req.GetMetadata()), &metadata); err != nil {
		h.log.Error("ошибка при парсинге metadata", "error", err)
		return grpc.CreateEntityResponse_builder{}.Build(), err
	}

	EntityData := &models.CreateEntityRequest{}
	EntityData.UserID = req.GetAccessToken().GetUserId()
	EntityData.Data = req.GetData()
	EntityData.Metadata = metadata
	if metadata.Name == "" {
		h.log.Error("ошибка при инициализации metadata", "error", "пустое значение name")
		return grpc.CreateEntityResponse_builder{}.Build(), status.Error(
			codes.InvalidArgument, repository.ErrNoMetadataSet.Error(),
		)
	}

	exists, err := h.entity.Exists(ctx, EntityData)
	if err != nil {
		h.log.Error("ошибка при проверке наличия данных", "error", err)
		return grpc.CreateEntityResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	if exists {
		h.log.Error("данные уже существуют")
		return grpc.CreateEntityResponse_builder{}.Build(), status.Error(
			codes.AlreadyExists, repository.ErrNameAlreadyExists.Error(),
		)
	}

	CreatedEntityID, err := h.entity.CreateEntity(ctx, EntityData)
	if err != nil {
		h.log.Error("ошибка при добавлении данных в БД", "error", err)
		return grpc.CreateEntityResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.CreateEntityResponse_builder{Id: CreatedEntityID}.Build(), nil
}

// EntityDelete - обработчик удаления данных из БД.
func (h *Handler) EntityDelete(ctx context.Context, req *grpc.DeleteEntityRequest) (*grpc.DeleteEntityResponse, error) {
	h.log.Debug("Удаление данных из БД")
	expToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении времени истечения срока токена", "error", err)
		return grpc.DeleteEntityResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(expToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.DeleteEntityResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	DeletedEntityID, err := h.entity.RemoveEntity(ctx, req.GetAccessToken().GetUserId(), req.GetName(), req.GetType())
	if err != nil {
		h.log.Error("ошибка при удалении данных", "error", err)
		return grpc.DeleteEntityResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.DeleteEntityResponse_builder{Id: DeletedEntityID}.Build(), nil
}

// EntityUpdate - обработчик обновления данных сущности.
func (h *Handler) EntityUpdate(ctx context.Context, req *grpc.UpdateEntityRequest) (*grpc.UpdateEntityResponse, error) {
	h.log.Debug("Запрос на обновление данных в БД")
	endDateToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении времени истечения срока токена", "error", err)
		return grpc.UpdateEntityResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(endDateToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.UpdateEntityResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	UpdatedEntityID, err := h.entity.UpdateEntity(ctx, req.GetAccessToken().GetUserId(), req.GetName(), req.GetType(), req.GetData())
	if err != nil {
		h.log.Error("ошибка при обновлении данных", "error", err)
		return grpc.UpdateEntityResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.UpdateEntityResponse_builder{Id: UpdatedEntityID}.Build(), nil
}
