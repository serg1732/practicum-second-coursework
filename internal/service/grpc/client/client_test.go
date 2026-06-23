package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/serg1732/practicum-second-coursework/internal/config"
	"github.com/serg1732/practicum-second-coursework/internal/models"
	pb "github.com/serg1732/practicum-second-coursework/internal/proto"
	"github.com/serg1732/practicum-second-coursework/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

var errTest = errors.New("test error")

func TestTypesToString(t *testing.T) {
	assert.Equal(t, "Текстовые данные", Text.ToString())
	assert.Equal(t, "Данные банковской карты", Card.ToString())
	assert.Equal(t, "Данные логин пароля", LoginPassword.ToString())
}

func TestBuildGRPCClientSuccess(t *testing.T) {
	logger := newTestLogger()
	cfg := &config.GophKeeperClientConfig{GRPCRemoteAddr: "passthrough:///bufnet"}

	got, err := BuildGRPCClient(logger, cfg, insecure.NewCredentials())

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Same(t, cfg, got.Cfg)
	assert.NotNil(t, got.grpc)
}

func TestClientPingSuccess(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		pingFunc: func(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
			require.NotNil(t, in)
			return pb.PingResponse_builder{Message: "pong"}.Build(), nil
		},
	})

	got, err := c.Ping(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "pong", got)
}

func TestClientPingError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		pingFunc: func(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
			return nil, errTest
		},
	})

	got, err := c.Ping(context.Background())

	assert.Empty(t, got)
	assert.ErrorIs(t, err, errTest)
}

func TestClientRegistrationSuccess(t *testing.T) {
	token := testToken()
	c := newTestClient(&mockGophKeeperClient{
		registrationFunc: func(ctx context.Context, in *pb.RegistrationRequest, opts ...grpc.CallOption) (*pb.RegistrationResponse, error) {
			assert.Equal(t, "user", in.GetUsername())
			assert.NotEmpty(t, in.GetPassword())
			assert.NotEqual(t, "password", in.GetPassword())

			return pb.RegistrationResponse_builder{AccessToken: buildPBToken(token)}.Build(), nil
		},
	})

	got, err := c.Registration(context.Background(), "user", "password")

	require.NoError(t, err)
	assert.Equal(t, token.AccessToken, got.AccessToken)
	assert.Equal(t, token.UserID, got.UserID)
	assert.Equal(t, token.Iat, got.Iat)
	assert.Equal(t, token.Exp, got.Exp)
}

func TestClientRegistrationGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		registrationFunc: func(ctx context.Context, in *pb.RegistrationRequest, opts ...grpc.CallOption) (*pb.RegistrationResponse, error) {
			return nil, errTest
		},
	})

	got, err := c.Registration(context.Background(), "user", "password")

	assert.Equal(t, models.Token{}, got)
	assert.ErrorIs(t, err, errTest)
}

func TestClientRegistrationInvalidTokenTime(t *testing.T) {
	invalidTimestamp := &timestamp.Timestamp{Seconds: 253402300800}
	c := newTestClient(&mockGophKeeperClient{
		registrationFunc: func(ctx context.Context, in *pb.RegistrationRequest, opts ...grpc.CallOption) (*pb.RegistrationResponse, error) {
			return pb.RegistrationResponse_builder{
				AccessToken: pb.Token_builder{
					Token:  "token",
					UserId: 1,
					Iat:    invalidTimestamp,
					Exp:    timestamp.New(time.Now().Add(time.Hour)),
				}.Build(),
			}.Build(), nil
		},
	})

	got, err := c.Registration(context.Background(), "user", "password")

	assert.Equal(t, models.Token{}, got)
	assert.Error(t, err)
}

func TestClientAuthenticationSuccess(t *testing.T) {
	token := testToken()
	c := newTestClient(&mockGophKeeperClient{
		authenticationFunc: func(ctx context.Context, in *pb.AuthenticationRequest, opts ...grpc.CallOption) (*pb.AuthenticationResponse, error) {
			assert.Equal(t, "user", in.GetUsername())
			assert.NotEmpty(t, in.GetPassword())
			assert.NotEqual(t, "password", in.GetPassword())

			return pb.AuthenticationResponse_builder{AccessToken: buildPBToken(token)}.Build(), nil
		},
	})

	got, err := c.Authentication(context.Background(), "user", "password")

	require.NoError(t, err)
	assert.Equal(t, token.AccessToken, got.AccessToken)
	assert.Equal(t, token.UserID, got.UserID)
	assert.Equal(t, token.Iat, got.Iat)
	assert.Equal(t, token.Exp, got.Exp)
}

func TestClientAuthenticationGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		authenticationFunc: func(ctx context.Context, in *pb.AuthenticationRequest, opts ...grpc.CallOption) (*pb.AuthenticationResponse, error) {
			return nil, errTest
		},
	})

	got, err := c.Authentication(context.Background(), "user", "password")

	assert.Equal(t, models.Token{}, got)
	assert.ErrorIs(t, err, errTest)
}

func TestClientUserExistSuccess(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		userExistFunc: func(ctx context.Context, in *pb.UserExistRequest, opts ...grpc.CallOption) (*pb.UserExistResponse, error) {
			assert.Equal(t, "user", in.GetUsername())
			return pb.UserExistResponse_builder{Exist: true}.Build(), nil
		},
	})

	got, err := c.UserExist(context.Background(), "user")

	require.NoError(t, err)
	assert.True(t, got)
}

func TestClientUserExistError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		userExistFunc: func(ctx context.Context, in *pb.UserExistRequest, opts ...grpc.CallOption) (*pb.UserExistResponse, error) {
			return pb.UserExistResponse_builder{Exist: false}.Build(), errTest
		},
	})

	got, err := c.UserExist(context.Background(), "user")

	assert.False(t, got)
	assert.ErrorIs(t, err, errTest)
}

func TestClientCardCreateSuccess(t *testing.T) {
	token := testToken()
	password := "master-password"
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			var metadata models.MetadataEntity
			require.NoError(t, json.Unmarshal([]byte(in.GetMetadata()), &metadata))
			assert.Equal(t, "card-name", metadata.Name)
			assert.Equal(t, "card-description", metadata.Description)
			assert.Equal(t, Card.ToString(), metadata.Type)

			plain, err := decryptTestData(in.GetData(), password)
			require.NoError(t, err)

			var card models.Card
			require.NoError(t, json.Unmarshal([]byte(plain), &card))
			assert.Equal(t, "card-name", card.Name)
			assert.Equal(t, "card-description", card.Description)
			assert.Equal(t, "MASTERCARD", card.PaymentSystem)
			assert.Equal(t, "4111111111111111", card.Number)
			assert.Equal(t, "IVAN IVANOV", card.Holder)
			assert.Equal(t, 321, card.CVC)
			assert.Equal(t, "12.12.2030", card.EndDate.Format(LayoutDate))

			require.NotNil(t, in.GetAccessToken())
			assert.Equal(t, token.AccessToken, in.GetAccessToken().GetToken())
			assert.Equal(t, token.UserID, in.GetAccessToken().GetUserId())

			return pb.CreateEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.CardCreate(
		context.Background(),
		"card-name",
		"card-description",
		password,
		"MASTERCARD",
		"4111111111111111",
		"IVAN IVANOV",
		"321",
		"12.12.2030",
		token,
	)

	require.NoError(t, err)
}

func TestClientCardCreateInvalidCVC(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			t.Fatal("EntityCreate не должен вызываться при невалидном cvc")
			return nil, nil
		},
	})

	err := c.CardCreate(context.Background(), "card", "desc", "password", "MASTERCARD", "number", "holder", "abc", "12.12.2030", testToken())

	assert.Error(t, err)
}

func TestClientCardCreateInvalidEndDate(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			t.Fatal("EntityCreate не должен вызываться при невалидной дате")
			return nil, nil
		},
	})

	err := c.CardCreate(context.Background(), "card", "desc", "password", "MASTERCARD", "number", "holder", "123", "12/2030", testToken())

	assert.Error(t, err)
}

func TestClientCardCreateGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.CardCreate(context.Background(), "card", "desc", "password", "MASTERCARD", "number", "holder", "123", "12.12.2030", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientCardDeleteSuccess(t *testing.T) {
	token := testToken()
	c := newTestClient(&mockGophKeeperClient{
		entityDeleteFunc: func(ctx context.Context, in *pb.DeleteEntityRequest, opts ...grpc.CallOption) (*pb.DeleteEntityResponse, error) {
			assert.Equal(t, "card", in.GetName())
			assert.Equal(t, Card.ToString(), in.GetType())
			assert.Equal(t, token.AccessToken, in.GetAccessToken().GetToken())
			return pb.DeleteEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.CardDelete(context.Background(), "card", token)

	require.NoError(t, err)
}

func TestClientCardDeleteError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityDeleteFunc: func(ctx context.Context, in *pb.DeleteEntityRequest, opts ...grpc.CallOption) (*pb.DeleteEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.CardDelete(context.Background(), "card", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientCardUpdateSuccess(t *testing.T) {
	token := testToken()
	password := "master-password"
	c := newTestClient(&mockGophKeeperClient{
		entityUpdateFunc: func(ctx context.Context, in *pb.UpdateEntityRequest, opts ...grpc.CallOption) (*pb.UpdateEntityResponse, error) {
			assert.Equal(t, "card", in.GetName())
			assert.Equal(t, Card.ToString(), in.GetType())
			assert.Equal(t, token.AccessToken, in.GetAccessToken().GetToken())

			plain, err := decryptTestData(in.GetData(), password)
			require.NoError(t, err)

			var card models.Card
			require.NoError(t, json.Unmarshal([]byte(plain), &card))
			assert.Equal(t, "mir", card.PaymentSystem)
			assert.Equal(t, "2200000000000000", card.Number)
			assert.Equal(t, "PETR PETROV", card.Holder)
			assert.Equal(t, 777, card.CVC)
			assert.Equal(t, "01.01.2031", card.EndDate.Format(LayoutDate))

			return pb.UpdateEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.CardUpdate(context.Background(), "card", password, "mir", "2200000000000000", "PETR PETROV", "777", "01.01.2031", token)

	require.NoError(t, err)
}

func TestClientCardUpdateInvalidCVC(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{})

	err := c.CardUpdate(context.Background(), "card", "password", "mir", "number", "holder", "wrong", "01.01.2031", testToken())

	assert.Error(t, err)
}

func TestClientCardUpdateInvalidEndDate(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{})

	err := c.CardUpdate(context.Background(), "card", "password", "mir", "number", "holder", "777", "01/2031", testToken())

	assert.Error(t, err)
}

func TestClientCardUpdateGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityUpdateFunc: func(ctx context.Context, in *pb.UpdateEntityRequest, opts ...grpc.CallOption) (*pb.UpdateEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.CardUpdate(context.Background(), "card", "password", "mir", "number", "holder", "777", "01.01.2031", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientTextCreateSuccess(t *testing.T) {
	token := testToken()
	password := "master-password"
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			var metadata models.MetadataEntity
			require.NoError(t, json.Unmarshal([]byte(in.GetMetadata()), &metadata))
			assert.Equal(t, "note", metadata.Name)
			assert.Equal(t, "note description", metadata.Description)
			assert.Equal(t, Text.ToString(), metadata.Type)

			plain, err := decryptTestData(in.GetData(), password)
			require.NoError(t, err)
			assert.Equal(t, "secret text", plain)
			assert.Equal(t, token.AccessToken, in.GetAccessToken().GetToken())

			return pb.CreateEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.TextCreate(context.Background(), "note", "note description", password, "secret text", token)

	require.NoError(t, err)
}

func TestClientTextCreateGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.TextCreate(context.Background(), "note", "description", "password", "text", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientTextDeleteSuccess(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityDeleteFunc: func(ctx context.Context, in *pb.DeleteEntityRequest, opts ...grpc.CallOption) (*pb.DeleteEntityResponse, error) {
			assert.Equal(t, "note", in.GetName())
			assert.Equal(t, Text.ToString(), in.GetType())
			return pb.DeleteEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.TextDelete(context.Background(), "note", testToken())

	require.NoError(t, err)
}

func TestClientTextDeleteError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityDeleteFunc: func(ctx context.Context, in *pb.DeleteEntityRequest, opts ...grpc.CallOption) (*pb.DeleteEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.TextDelete(context.Background(), "note", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientTextUpdateSuccess(t *testing.T) {
	password := "master-password"
	c := newTestClient(&mockGophKeeperClient{
		entityUpdateFunc: func(ctx context.Context, in *pb.UpdateEntityRequest, opts ...grpc.CallOption) (*pb.UpdateEntityResponse, error) {
			assert.Equal(t, "note", in.GetName())
			assert.Equal(t, Text.ToString(), in.GetType())

			plain, err := decryptTestData(in.GetData(), password)
			require.NoError(t, err)
			assert.Equal(t, "updated text", plain)

			return pb.UpdateEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.TextUpdate(context.Background(), "note", password, "updated text", testToken())

	require.NoError(t, err)
}

func TestClientTextUpdateGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityUpdateFunc: func(ctx context.Context, in *pb.UpdateEntityRequest, opts ...grpc.CallOption) (*pb.UpdateEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.TextUpdate(context.Background(), "note", "password", "text", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientCredsCreateSuccess(t *testing.T) {
	passwordSecure := "master-password"
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			var metadata models.MetadataEntity
			require.NoError(t, json.Unmarshal([]byte(in.GetMetadata()), &metadata))
			assert.Equal(t, "github", metadata.Name)
			assert.Equal(t, "github account", metadata.Description)
			assert.Equal(t, LoginPassword.ToString(), metadata.Type)

			plain, err := decryptTestData(in.GetData(), passwordSecure)
			require.NoError(t, err)

			var creds models.Creds
			require.NoError(t, json.Unmarshal([]byte(plain), &creds))
			assert.Equal(t, "zed", creds.Login)
			assert.Equal(t, "secret", creds.Password)

			return pb.CreateEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.CredsCreate(context.Background(), "github", "github account", passwordSecure, "zed", "secret", testToken())

	require.NoError(t, err)
}

func TestClientCredsCreateGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityCreateFunc: func(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.CredsCreate(context.Background(), "github", "github account", "password", "login", "secret", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientCredsDeleteSuccess(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityDeleteFunc: func(ctx context.Context, in *pb.DeleteEntityRequest, opts ...grpc.CallOption) (*pb.DeleteEntityResponse, error) {
			assert.Equal(t, "github", in.GetName())
			assert.Equal(t, LoginPassword.ToString(), in.GetType())
			return pb.DeleteEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.CredsDelete(context.Background(), "github", testToken())

	require.NoError(t, err)
}

func TestClientCredsDeleteError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityDeleteFunc: func(ctx context.Context, in *pb.DeleteEntityRequest, opts ...grpc.CallOption) (*pb.DeleteEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.CredsDelete(context.Background(), "github", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientCredsUpdateSuccess(t *testing.T) {
	passwordSecure := "master-password"
	c := newTestClient(&mockGophKeeperClient{
		entityUpdateFunc: func(ctx context.Context, in *pb.UpdateEntityRequest, opts ...grpc.CallOption) (*pb.UpdateEntityResponse, error) {
			assert.Equal(t, "github", in.GetName())
			assert.Equal(t, LoginPassword.ToString(), in.GetType())

			plain, err := decryptTestData(in.GetData(), passwordSecure)
			require.NoError(t, err)

			var creds models.Creds
			require.NoError(t, json.Unmarshal([]byte(plain), &creds))
			assert.Equal(t, "new-login", creds.Login)
			assert.Equal(t, "new-password", creds.Password)

			return pb.UpdateEntityResponse_builder{}.Build(), nil
		},
	})

	err := c.CredsUpdate(context.Background(), "github", passwordSecure, "new-login", "new-password", testToken())

	require.NoError(t, err)
}

func TestClientCredsUpdateGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityUpdateFunc: func(ctx context.Context, in *pb.UpdateEntityRequest, opts ...grpc.CallOption) (*pb.UpdateEntityResponse, error) {
			return nil, errTest
		},
	})

	err := c.CredsUpdate(context.Background(), "github", "password", "login", "secret", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientFileUploadSuccess(t *testing.T) {
	password := "master-password"
	c := newTestClient(&mockGophKeeperClient{
		fileUploadFunc: func(ctx context.Context, in *pb.UploadBinaryRequest, opts ...grpc.CallOption) (*pb.UploadBinaryResponse, error) {
			assert.Equal(t, "secret.txt", in.GetName())

			plain, err := decryptTestData(in.GetData(), password)
			require.NoError(t, err)
			assert.Equal(t, "file content", plain)

			return pb.UploadBinaryResponse_builder{Name: "secret.txt"}.Build(), nil
		},
	})

	got, err := c.FileUpload(context.Background(), "secret.txt", password, []byte("file content"), testToken())

	require.NoError(t, err)
	assert.Equal(t, "secret.txt", got)
}

func TestClientFileUploadGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		fileUploadFunc: func(ctx context.Context, in *pb.UploadBinaryRequest, opts ...grpc.CallOption) (*pb.UploadBinaryResponse, error) {
			return nil, errTest
		},
	})

	got, err := c.FileUpload(context.Background(), "secret.txt", "password", []byte("file content"), testToken())

	assert.Empty(t, got)
	assert.ErrorIs(t, err, errTest)
}

func TestClientFileDownloadSuccess(t *testing.T) {
	password := "master-password"
	secretKey := utils.Pbkdf2KeySecureRandom([]byte(password))
	encryptedFile, err := utils.Encrypt("file content", secretKey)
	require.NoError(t, err)

	c := newTestClient(&mockGophKeeperClient{
		fileDownloadFunc: func(ctx context.Context, in *pb.DownloadBinaryRequest, opts ...grpc.CallOption) (*pb.DownloadBinaryResponse, error) {
			assert.Equal(t, "secret.txt", in.GetName())
			return pb.DownloadBinaryResponse_builder{Data: []byte(encryptedFile)}.Build(), nil
		},
	})

	got, err := c.FileDownload(context.Background(), "secret.txt", password, testToken())

	require.NoError(t, err)
	assert.Equal(t, []byte("file content"), got)
}

func TestClientFileDownloadGRPCError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		fileDownloadFunc: func(ctx context.Context, in *pb.DownloadBinaryRequest, opts ...grpc.CallOption) (*pb.DownloadBinaryResponse, error) {
			return nil, errTest
		},
	})

	got, err := c.FileDownload(context.Background(), "secret.txt", "password", testToken())

	assert.Nil(t, got)
	assert.ErrorIs(t, err, errTest)
}

func TestClientFileDownloadDecryptError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		fileDownloadFunc: func(ctx context.Context, in *pb.DownloadBinaryRequest, opts ...grpc.CallOption) (*pb.DownloadBinaryResponse, error) {
			return pb.DownloadBinaryResponse_builder{Data: []byte("broken encrypted data")}.Build(), nil
		},
	})

	got, err := c.FileDownload(context.Background(), "secret.txt", "password", testToken())

	assert.Nil(t, got)
	assert.Error(t, err)
}

func TestClientFileRemoveSuccess(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		fileRemoveFunc: func(ctx context.Context, in *pb.DeleteBinaryRequest, opts ...grpc.CallOption) (*pb.DeleteBinaryResponse, error) {
			assert.Equal(t, "secret.txt", in.GetName())
			return pb.DeleteBinaryResponse_builder{}.Build(), nil
		},
	})

	err := c.FileRemove(context.Background(), "secret.txt", testToken())

	require.NoError(t, err)
}

func TestClientFileRemoveError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		fileRemoveFunc: func(ctx context.Context, in *pb.DeleteBinaryRequest, opts ...grpc.CallOption) (*pb.DeleteBinaryResponse, error) {
			return nil, errTest
		},
	})

	err := c.FileRemove(context.Background(), "secret.txt", testToken())

	assert.ErrorIs(t, err, errTest)
}

func TestClientSynchronizeEntityGetListError(t *testing.T) {
	c := newTestClient(&mockGophKeeperClient{
		entityGetListFunc: func(ctx context.Context, in *pb.GetListEntityRequest, opts ...grpc.CallOption) (*pb.GetListEntityResponse, error) {
			assert.Equal(t, Text.ToString(), in.GetType())
			return nil, errTest
		},
	})

	got, err := c.Synchronize(context.Background(), "password", testToken())

	assert.Equal(t, models.SyncResult{}, got)
	assert.ErrorIs(t, err, errTest)
}

func newTestClient(grpcClient pb.GophKeeperClient) Client {
	return Client{
		grpc: grpcClient,
		log:  newTestLogger(),
	}
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testToken() models.Token {
	return models.Token{
		AccessToken: "access-token",
		UserID:      42,
		Iat:         time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		Exp:         time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
	}
}

func buildPBToken(token models.Token) *pb.Token {
	return pb.Token_builder{
		Token:  token.AccessToken,
		UserId: token.UserID,
		Iat:    timestamp.New(token.Iat),
		Exp:    timestamp.New(token.Exp),
	}.Build()
}

func decryptTestData(data []byte, password string) (string, error) {
	secretKey := utils.Pbkdf2KeySecureRandom([]byte(password))
	return utils.Decrypt(string(data), secretKey)
}

type mockGophKeeperClient struct {
	pingFunc           func(context.Context, *pb.PingRequest, ...grpc.CallOption) (*pb.PingResponse, error)
	registrationFunc   func(context.Context, *pb.RegistrationRequest, ...grpc.CallOption) (*pb.RegistrationResponse, error)
	authenticationFunc func(context.Context, *pb.AuthenticationRequest, ...grpc.CallOption) (*pb.AuthenticationResponse, error)
	userExistFunc      func(context.Context, *pb.UserExistRequest, ...grpc.CallOption) (*pb.UserExistResponse, error)
	entityCreateFunc   func(context.Context, *pb.CreateEntityRequest, ...grpc.CallOption) (*pb.CreateEntityResponse, error)
	entityDeleteFunc   func(context.Context, *pb.DeleteEntityRequest, ...grpc.CallOption) (*pb.DeleteEntityResponse, error)
	entityUpdateFunc   func(context.Context, *pb.UpdateEntityRequest, ...grpc.CallOption) (*pb.UpdateEntityResponse, error)
	entityGetListFunc  func(context.Context, *pb.GetListEntityRequest, ...grpc.CallOption) (*pb.GetListEntityResponse, error)
	fileUploadFunc     func(context.Context, *pb.UploadBinaryRequest, ...grpc.CallOption) (*pb.UploadBinaryResponse, error)
	fileDownloadFunc   func(context.Context, *pb.DownloadBinaryRequest, ...grpc.CallOption) (*pb.DownloadBinaryResponse, error)
	fileRemoveFunc     func(context.Context, *pb.DeleteBinaryRequest, ...grpc.CallOption) (*pb.DeleteBinaryResponse, error)
	fileGetListFunc    func(context.Context, *pb.GetListBinaryRequest, ...grpc.CallOption) (*pb.GetListBinaryResponse, error)
}

func (f *mockGophKeeperClient) Ping(ctx context.Context, in *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	if f.pingFunc != nil {
		return f.pingFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected Ping call")
}

func (f *mockGophKeeperClient) Registration(ctx context.Context, in *pb.RegistrationRequest, opts ...grpc.CallOption) (*pb.RegistrationResponse, error) {
	if f.registrationFunc != nil {
		return f.registrationFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected Registration call")
}

func (f *mockGophKeeperClient) Authentication(ctx context.Context, in *pb.AuthenticationRequest, opts ...grpc.CallOption) (*pb.AuthenticationResponse, error) {
	if f.authenticationFunc != nil {
		return f.authenticationFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected Authentication call")
}

func (f *mockGophKeeperClient) UserExist(ctx context.Context, in *pb.UserExistRequest, opts ...grpc.CallOption) (*pb.UserExistResponse, error) {
	if f.userExistFunc != nil {
		return f.userExistFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected UserExist call")
}

func (f *mockGophKeeperClient) EntityCreate(ctx context.Context, in *pb.CreateEntityRequest, opts ...grpc.CallOption) (*pb.CreateEntityResponse, error) {
	if f.entityCreateFunc != nil {
		return f.entityCreateFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected EntityCreate call")
}

func (f *mockGophKeeperClient) EntityDelete(ctx context.Context, in *pb.DeleteEntityRequest, opts ...grpc.CallOption) (*pb.DeleteEntityResponse, error) {
	if f.entityDeleteFunc != nil {
		return f.entityDeleteFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected EntityDelete call")
}

func (f *mockGophKeeperClient) EntityUpdate(ctx context.Context, in *pb.UpdateEntityRequest, opts ...grpc.CallOption) (*pb.UpdateEntityResponse, error) {
	if f.entityUpdateFunc != nil {
		return f.entityUpdateFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected EntityUpdate call")
}

func (f *mockGophKeeperClient) EntityGetList(ctx context.Context, in *pb.GetListEntityRequest, opts ...grpc.CallOption) (*pb.GetListEntityResponse, error) {
	if f.entityGetListFunc != nil {
		return f.entityGetListFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected EntityGetList call")
}

func (f *mockGophKeeperClient) FileUpload(ctx context.Context, in *pb.UploadBinaryRequest, opts ...grpc.CallOption) (*pb.UploadBinaryResponse, error) {
	if f.fileUploadFunc != nil {
		return f.fileUploadFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected FileUpload call")
}

func (f *mockGophKeeperClient) FileDownload(ctx context.Context, in *pb.DownloadBinaryRequest, opts ...grpc.CallOption) (*pb.DownloadBinaryResponse, error) {
	if f.fileDownloadFunc != nil {
		return f.fileDownloadFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected FileDownload call")
}

func (f *mockGophKeeperClient) FileRemove(ctx context.Context, in *pb.DeleteBinaryRequest, opts ...grpc.CallOption) (*pb.DeleteBinaryResponse, error) {
	if f.fileRemoveFunc != nil {
		return f.fileRemoveFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected FileRemove call")
}

func (f *mockGophKeeperClient) FileGetList(ctx context.Context, in *pb.GetListBinaryRequest, opts ...grpc.CallOption) (*pb.GetListBinaryResponse, error) {
	if f.fileGetListFunc != nil {
		return f.fileGetListFunc(ctx, in, opts...)
	}
	return nil, errors.New("unexpected FileGetList call")
}
