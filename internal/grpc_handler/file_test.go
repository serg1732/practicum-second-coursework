package grpc_handler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/serg1732/practicum-second-coursework/internal/models"
	grpc "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestFileUploadSuccess(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	data := []byte("file-data")

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *models.FileRequest) (bool, error) {
			assert.Equal(t, testUserID, req.UserID)
			assert.Equal(t, "file.bin", req.Name)
			return false, nil
		},
	)
	deps.file.EXPECT().UploadFile(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *models.FileRequest) (*models.File, error) {
			assert.Equal(t, testUserID, req.UserID)
			assert.Equal(t, "file.bin", req.Name)
			return &models.File{Name: req.Name}, nil
		},
	)
	deps.fileManager.EXPECT().UploadFile(testUserID, "file.bin", data).Return(nil)

	resp, err := h.FileUpload(context.Background(), grpc.UploadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
		Data:        data,
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "file.bin", resp.GetName())
}

func TestFileUploadErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.FileUpload(context.Background(), grpc.UploadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
		Data:        []byte("file-data"),
	}.Build())

	assert.Error(t, err)
}

func TestFileUploadTokenNotValid(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.FileUpload(context.Background(), grpc.UploadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
		Data:        []byte("file-data"),
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestFileUploadFileAlreadyExists(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).Return(true, nil)

	_, err := h.FileUpload(context.Background(), grpc.UploadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
		Data:        []byte("file-data"),
	}.Build())

	assertErrorCode(t, codes.AlreadyExists, err)
}

func TestFileUploadErrorCheckExist(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).Return(false, errTest)

	_, err := h.FileUpload(context.Background(), grpc.UploadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
		Data:        []byte("file-data"),
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestFileUploadErrorSave(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).Return(false, nil)
	deps.file.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(nil, errTest)

	_, err := h.FileUpload(context.Background(), grpc.UploadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
		Data:        []byte("file-data"),
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestFileUploadErrorWriteLocalStorage(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	data := []byte("file-data")

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).Return(false, nil)
	deps.file.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(&models.File{Name: "file.bin"}, nil)
	deps.fileManager.EXPECT().UploadFile(testUserID, "file.bin", data).Return(errTest)

	_, err := h.FileUpload(context.Background(), grpc.UploadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
		Data:        data,
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestFileDownloadSuccess(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	data := []byte("file-data")

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *models.FileRequest) (bool, error) {
			assert.Equal(t, testUserID, req.UserID)
			assert.Equal(t, "file.bin", req.Name)
			return true, nil
		},
	)
	deps.fileManager.EXPECT().DownloadFile(testUserID, "file.bin").Return(data, nil)

	resp, err := h.FileDownload(context.Background(), grpc.DownloadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, data, resp.GetData())
}

func TestFileDownloadErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.FileDownload(context.Background(), grpc.DownloadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assert.Error(t, err)
}

func TestFileDownloadNotValidToken(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.FileDownload(context.Background(), grpc.DownloadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestFileDownloadFileNotExist(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).Return(false, nil)

	_, err := h.FileDownload(context.Background(), grpc.DownloadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assertErrorCode(t, codes.AlreadyExists, err)
}

func TestFileDownloadErrorCheckExist(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).Return(false, errTest)

	_, err := h.FileDownload(context.Background(), grpc.DownloadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestFileDownloadErrorReadFile(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().FileExists(gomock.Any(), gomock.Any()).Return(true, nil)
	deps.fileManager.EXPECT().DownloadFile(testUserID, "file.bin").Return(nil, errTest)

	_, err := h.FileDownload(context.Background(), grpc.DownloadBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestFileGetListSuccess(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().GetListFile(gomock.Any(), testUserID).Return(nil, nil)

	resp, err := h.FileGetList(context.Background(), grpc.GetListBinaryRequest_builder{
		AccessToken: newAccessToken(),
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 0, len(resp.GetNode()))
}

func TestFileGetListErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.FileGetList(context.Background(), grpc.GetListBinaryRequest_builder{
		AccessToken: newAccessToken(),
	}.Build())

	assert.Error(t, err)
}

func TestFileGetListTokenNotValid(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.FileGetList(context.Background(), grpc.GetListBinaryRequest_builder{
		AccessToken: newAccessToken(),
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestFileGetListError(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().GetListFile(gomock.Any(), testUserID).Return(nil, errTest)

	_, err := h.FileGetList(context.Background(), grpc.GetListBinaryRequest_builder{
		AccessToken: newAccessToken(),
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestFileRemoveSuccess(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *models.FileRequest) (int64, error) {
			assert.Equal(t, testUserID, req.UserID)
			assert.Equal(t, "file.bin", req.Name)
			return 77, nil
		},
	)
	deps.fileManager.EXPECT().RemoveFile(testUserID, "file.bin").Return(nil)

	resp, err := h.FileRemove(context.Background(), grpc.DeleteBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, int64(77), resp.GetId())
}

func TestFileRemoveErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.FileRemove(context.Background(), grpc.DeleteBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assert.Error(t, err)
}

func TestFileRemoveNotValidToken(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.FileRemove(context.Background(), grpc.DeleteBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestFileRemoveErrorRemoveFromDB(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).Return(int64(0), errTest)

	_, err := h.FileRemove(context.Background(), grpc.DeleteBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestFileRemoveErrorRemoveLocal(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.file.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).Return(int64(77), nil)
	deps.fileManager.EXPECT().RemoveFile(testUserID, "file.bin").Return(errTest)

	_, err := h.FileRemove(context.Background(), grpc.DeleteBinaryRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "file.bin",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}
