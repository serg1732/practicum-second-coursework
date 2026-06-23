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

func TestEntityGetListSuccessGetList(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	deps.entity.EXPECT().GetList(gomock.Any(), testUserID, "card").Return(nil, nil)

	resp, err := h.EntityGetList(context.Background(), grpc.GetListEntityRequest_builder{
		AccessToken: newAccessToken(),
		Type:        "card",
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 0, len(resp.GetNode()))
}

func TestEntityGetListNotValidToken(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.EntityGetList(context.Background(), grpc.GetListEntityRequest_builder{
		AccessToken: newAccessToken(),
		Type:        "card",
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestEntityGetListErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.EntityGetList(context.Background(), grpc.GetListEntityRequest_builder{
		AccessToken: newAccessToken(),
		Type:        "card",
	}.Build())

	assert.Error(t, err)
}

func TestEntityGetListError(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	deps.entity.EXPECT().
		GetList(gomock.Any(), testUserID, "card").
		Return(nil, errTest)

	_, err := h.EntityGetList(context.Background(), grpc.GetListEntityRequest_builder{
		AccessToken: newAccessToken(),
		Type:        "card",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestEntityCreateSuccessCreate(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	data := []byte("encrypted-data")

	deps.entity.EXPECT().Exists(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *models.CreateEntityRequest) (bool, error) {
			assert.Equal(t, testUserID, req.UserID)
			assert.Equal(t, data, req.Data)
			assert.Equal(t, "card-1", req.Metadata.Name)
			assert.Equal(t, "card", req.Metadata.Type)
			return false, nil
		},
	)
	deps.entity.EXPECT().CreateEntity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *models.CreateEntityRequest) (int64, error) {
			assert.Equal(t, "card-1", req.Metadata.Name)
			return 100, nil
		},
	)

	resp, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{"Name":"card-1","Type":"card","Description":"desc"}`,
		Data:        data,
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, int64(100), resp.GetId())
}

func TestEntityCreateErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{"Name":"card-1","Type":"card"}`,
		Data:        []byte("data"),
	}.Build())

	assert.Error(t, err)
}

func TestEntityCreateNotValidToken(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{"Name":"card-1","Type":"card"}`,
		Data:        []byte("data"),
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestEntityCreateNotValidJSONMetadata(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	_, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{bad-json}`,
		Data:        []byte("data"),
	}.Build())

	assert.Error(t, err)
}

func TestEntityCreateEmptyName(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	_, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{"Type":"card"}`,
		Data:        []byte("data"),
	}.Build())

	assertErrorCode(t, codes.InvalidArgument, err)
}

func TestEntityCreateAlreadyExistEntity(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.entity.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)

	_, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{"Name":"card-1","Type":"card"}`,
		Data:        []byte("data"),
	}.Build())

	assertErrorCode(t, codes.AlreadyExists, err)
}

func TestEntityCreateErrorCheckExist(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.entity.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, errTest)

	_, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{"Name":"card-1","Type":"card"}`,
		Data:        []byte("data"),
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestEntityCreateError(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)

	deps.entity.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)
	deps.entity.EXPECT().CreateEntity(gomock.Any(), gomock.Any()).Return(int64(0), errTest)

	_, err := h.EntityCreate(context.Background(), grpc.CreateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Metadata:    `{"Name":"card-1","Type":"card"}`,
		Data:        []byte("data"),
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestEntityDeleteSuccessRemove(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	deps.entity.EXPECT().RemoveEntity(gomock.Any(), testUserID, "card-1", "card").Return(int64(101), nil)

	resp, err := h.EntityDelete(context.Background(), grpc.DeleteEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, int64(101), resp.GetId())
}

func TestEntityDeleteErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.EntityDelete(context.Background(), grpc.DeleteEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
	}.Build())

	assert.Error(t, err)
}

func TestEntityDeleteNotValidToken(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.EntityDelete(context.Background(), grpc.DeleteEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestEntityDeleteErrorRemove(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	deps.entity.EXPECT().RemoveEntity(gomock.Any(), testUserID, "card-1", "card").Return(int64(0), errTest)

	_, err := h.EntityDelete(context.Background(), grpc.DeleteEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}

func TestEntityUpdateSuccess(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	data := []byte("new-data")
	deps.entity.EXPECT().UpdateEntity(gomock.Any(), testUserID, "card-1", "card", data).Return(int64(102), nil)

	resp, err := h.EntityUpdate(context.Background(), grpc.UpdateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
		Data:        data,
	}.Build())

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, int64(102), resp.GetId())
}

func TestEntityUpdateErrorGetToken(t *testing.T) {
	h, deps := newTestHandler(t)
	deps.token.EXPECT().GetExpToken(gomock.Any(), testAccessToken).Return(zeroTime(), errTest)

	_, err := h.EntityUpdate(context.Background(), grpc.UpdateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
		Data:        []byte("new-data"),
	}.Build())

	assert.Error(t, err)
}

func TestEntityUpdateNotValidToken(t *testing.T) {
	h, deps := newTestHandler(t)
	expectInvalidToken(deps)

	_, err := h.EntityUpdate(context.Background(), grpc.UpdateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
		Data:        []byte("new-data"),
	}.Build())

	assertErrorCode(t, codes.Unauthenticated, err)
}

func TestEntityUpdateError(t *testing.T) {
	h, deps := newTestHandler(t)
	expectValidToken(deps)
	data := []byte("new-data")
	deps.entity.EXPECT().UpdateEntity(gomock.Any(), testUserID, "card-1", "card", data).Return(int64(0), errTest)

	_, err := h.EntityUpdate(context.Background(), grpc.UpdateEntityRequest_builder{
		AccessToken: newAccessToken(),
		Name:        "card-1",
		Type:        "card",
		Data:        data,
	}.Build())

	assertErrorCode(t, codes.Internal, err)
}
