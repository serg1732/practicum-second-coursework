package grpc_handler

import (
	"context"

	"github.com/serg1732/practicum-second-coursework/internal/models"
	grpc "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FileDownload - обработчик загрузки файла с сервера.
func (h *Handler) FileDownload(ctx context.Context, req *grpc.DownloadBinaryRequest) (*grpc.DownloadBinaryResponse, error) {
	h.log.Debug("Запрос на загрузку файла с сервера", "filename", req.GetName())
	expToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении времени истечения срока токена", "error", err)
		return grpc.DownloadBinaryResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(expToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.DownloadBinaryResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	fileData := &models.FileRequest{}
	fileData.UserID = req.GetAccessToken().GetUserId()
	fileData.Name = req.GetName()

	exists, err := h.file.FileExists(ctx, fileData)
	if err != nil {
		h.log.Error("ошибка при проверке существования файла", "error", err)
		return grpc.DownloadBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	if !exists {
		err = repository.ErrFileNotExists
		h.log.Error("файл не существует", "filename", fileData.Name)
		return grpc.DownloadBinaryResponse_builder{}.Build(), status.Error(
			codes.AlreadyExists, err.Error(),
		)
	}

	data, err := h.fileManager.DownloadFile(req.GetAccessToken().GetUserId(), req.GetName())
	if err != nil {
		h.log.Error("ошибка при загрузке файла на сервер", "error", err)
		return grpc.DownloadBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.DownloadBinaryResponse_builder{Data: data}.Build(), nil
}

// FileGetList - обработчик получения списка загруженных файлов.
func (h *Handler) FileGetList(ctx context.Context, req *grpc.GetListBinaryRequest) (*grpc.GetListBinaryResponse, error) {
	h.log.Debug("Запрос на получени списка загруженных файлов")
	expToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении времени истечения срока токена", "error", err)
		return grpc.GetListBinaryResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(expToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.GetListBinaryResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	ListFile, err := h.file.GetListFile(ctx, req.GetAccessToken().GetUserId())
	if err != nil {
		h.log.Error("ошибка при получении списка загруженных файлов", "error", err)
		return grpc.GetListBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	list := models.GetListFile(ListFile)
	return grpc.GetListBinaryResponse_builder{Node: list}.Build(), nil
}

// FileRemove - обработчик удаления файла на сервере.
func (h *Handler) FileRemove(ctx context.Context, req *grpc.DeleteBinaryRequest) (*grpc.DeleteBinaryResponse, error) {
	h.log.Debug("Запрос на удаление файла из БД", "filename", req.GetName())
	expToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении времени истечения срока токена", "error", err)
		return grpc.DeleteBinaryResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(expToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.DeleteBinaryResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	fileData := &models.FileRequest{}
	fileData.UserID = req.GetAccessToken().GetUserId()
	fileData.Name = req.GetName()

	BinaryID, err := h.file.DeleteFile(ctx, fileData)
	if err != nil {
		h.log.Error("ошибка при удалении файла", "error", err)
		return grpc.DeleteBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}

	if err = h.fileManager.RemoveFile(req.GetAccessToken().GetUserId(), req.GetName()); err != nil {
		h.log.Error("ошибка при удалении файла", "error", err)
		return grpc.DeleteBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.DeleteBinaryResponse_builder{Id: BinaryID}.Build(), nil
}

// FileUpload - обработчик загрузки файла на сервер.
func (h *Handler) FileUpload(ctx context.Context, req *grpc.UploadBinaryRequest) (*grpc.UploadBinaryResponse, error) {
	h.log.Debug("Запрос на загрузку файла на сервер", "filename", req.GetName())
	expToken, err := h.token.GetExpToken(ctx, req.GetAccessToken().GetToken())
	if err != nil {
		h.log.Error("ошибка при получении времени истечения срока токена", "error", err)
		return grpc.UploadBinaryResponse_builder{}.Build(), err
	}
	if isValid := h.token.Validate(expToken); !isValid {
		h.log.Error("вышел срок действия токена")
		return grpc.UploadBinaryResponse_builder{}.Build(), status.Error(
			codes.Unauthenticated, repository.ErrNotValidateToken.Error(),
		)
	}

	fileData := &models.FileRequest{}
	fileData.UserID = req.GetAccessToken().GetUserId()
	fileData.Name = req.GetName()

	exists, err := h.file.FileExists(ctx, fileData)
	if err != nil {
		h.log.Error("ошибка при проверке файла на существование", "error", err)
		return grpc.UploadBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	if exists {
		err = repository.ErrNameAlreadyExists
		h.log.Error("имя файла уже существует", "filename", fileData.Name)
		return grpc.UploadBinaryResponse_builder{}.Build(), status.Error(
			codes.AlreadyExists, err.Error(),
		)
	}

	UploadFile, err := h.file.UploadFile(ctx, fileData)
	if err != nil {
		h.log.Error("ошибка при загрузке файла на сервер", "error", err)
		return grpc.UploadBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}

	if err = h.fileManager.UploadFile(req.GetAccessToken().GetUserId(), req.GetName(), req.GetData()); err != nil {
		h.log.Error("ошибка при загрузке с сервера файла", "error", err)
		return grpc.UploadBinaryResponse_builder{}.Build(), status.Error(
			codes.Internal, err.Error(),
		)
	}
	return grpc.UploadBinaryResponse_builder{Name: UploadFile.Name}.Build(), nil
}
