package controller_test

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/spriigan/RPApp/interface/controller"
	"github.com/spriigan/RPApp/interface/repository"
	"github.com/spriigan/RPApp/user-proto/grpc/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type interactorMock struct {
	mock.Mock
}

func (in *interactorMock) Create(ctx context.Context, payload *models.UserPayload) (*models.UserBio, error) {
	args := in.Called(payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserBio), args.Error(1)
}

func (in *interactorMock) FindUsers(ctx context.Context) (*models.Users, error) {
	args := in.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Users), args.Error(1)
}

func (in *interactorMock) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	args := in.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (in *interactorMock) DeleteByUsername(ctx context.Context, username string) error {
	args := in.Called(username)
	return args.Error(0)
}

func (in *interactorMock) Update(ctx context.Context, user *models.UserPayload) error {
	args := in.Called(user)
	return args.Error(0)
}

var mockInteractor *interactorMock
var client models.UserServiceClient
var lis *bufconn.Listener

func bufDialer(ctx context.Context, s string) (net.Conn, error) {
	return lis.Dial()
}

func TestMain(m *testing.M) {
	lis = bufconn.Listen(1024 * 1024)
	defer lis.Close()
	s := grpc.NewServer()
	defer s.Stop()
	mockInteractor = new(interactorMock)
	models.RegisterUserServiceServer(s, controller.NewUserServer(mockInteractor))
	conn, err := grpc.DialContext(context.Background(), "buffnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client = models.NewUserServiceClient(conn)
	go func() {
		if err = s.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()
	os.Exit(m.Run())
}

func TestRegisterUser(t *testing.T) {
	testTable := map[string]struct {
		arrange func(t *testing.T)
		assert  func(t *testing.T, actual *models.UserBio, err error)
	}{
		"succes call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("Create", mock.Anything).Return(&models.UserBio{}, nil).Once()
			},
			assert: func(t *testing.T, actual *models.UserBio, err error) {
				require.NoError(t, err)
				require.NotZero(t, actual)
			},
		},
		"fail call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("Create", mock.Anything).Return(nil, errors.New("got an error")).Once()
			},
			assert: func(t *testing.T, actual *models.UserBio, err error) {
				require.Error(t, err)
				require.Zero(t, actual)
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for k, v := range testTable {
		t.Run(k, func(t *testing.T) {
			v.arrange(t)

			result, err := client.RegisterUser(ctx, &models.UserPayload{Bio: &models.UserBio{}})

			v.assert(t, result, err)
		})
	}
}

func TestFindByUsername(t *testing.T) {
	user := &models.User{
		Fname: "dabi",
	}
	testTable := map[string]struct {
		arrange func(t *testing.T)
		assert  func(t *testing.T, actual *models.UserBio, err error)
	}{
		"succes call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("FindByUsername", mock.Anything).Return(user, nil).Once()
			},
			assert: func(t *testing.T, actual *models.UserBio, err error) {
				require.NoError(t, err)
				require.NotNil(t, actual)
				require.Equal(t, user.Fname, actual.GetFname())
			},
		},
		"fail call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("FindByUsername", mock.Anything).Return(nil, errors.New("got an error")).Once()
			},
			assert: func(t *testing.T, actual *models.UserBio, err error) {
				require.Error(t, err)
				require.Nil(t, actual)
			},
		},
		"user not found": {
			arrange: func(t *testing.T) {
				mockInteractor.On("FindByUsername", mock.Anything).Return(nil, repository.ErrNoUserFound).Once()
			},
			assert: func(t *testing.T, actual *models.UserBio, err error) {
				require.Error(t, err)
				require.Nil(t, actual)
				require.ErrorIs(t, err, status.Error(codes.NotFound, repository.ErrNoUserFound.Error()))
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for k, v := range testTable {
		t.Run(k, func(t *testing.T) {
			v.arrange(t)

			result, err := client.FindByUsername(ctx, &models.Username{Username: ""})

			v.assert(t, result, err)
		})
	}
}

func TestFindUsers(t *testing.T) {
	bio := []*models.UserBio{
		{},
		{},
		{},
	}
	users := &models.Users{User: bio}
	testTable := map[string]struct {
		arrange func(t *testing.T)
		assert  func(t *testing.T, actual *models.Users, err error)
	}{
		"succes call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("FindUsers").Return(users, nil).Once()
			},
			assert: func(t *testing.T, actual *models.Users, err error) {
				require.NoError(t, err)
				require.Equal(t, users.User, actual.User)
				require.Equal(t, len(users.User), len(actual.User))
			},
		},
		"fail call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("FindUsers", mock.Anything).Return(nil, errors.New("got an error")).Once()
			},
			assert: func(t *testing.T, actual *models.Users, err error) {
				require.Error(t, err)
				require.Zero(t, actual)
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for k, v := range testTable {
		t.Run(k, func(t *testing.T) {
			v.arrange(t)

			result, err := client.FindUsers(ctx, &emptypb.Empty{})

			v.assert(t, result, err)
		})
	}
}

func TestDeleteByUsername(t *testing.T) {
	testTable := map[string]struct {
		arrange func(t *testing.T)
		assert  func(t *testing.T, err error)
	}{
		"succes call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("DeleteByUsername", mock.Anything).Return(nil).Once()
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"fail call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("DeleteByUsername", mock.Anything).Return(errors.New("got an error")).Once()
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for k, v := range testTable {
		t.Run(k, func(t *testing.T) {
			v.arrange(t)

			_, err := client.DeleteByUsername(ctx, &models.Username{Username: ""})

			v.assert(t, err)
		})
	}
}

func TestUpdate(t *testing.T) {
	testTable := map[string]struct {
		arrange func(t *testing.T)
		assert  func(t *testing.T, err error)
	}{
		"succes call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("Update", mock.Anything).Return(nil).Once()
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"fail call": {
			arrange: func(t *testing.T) {
				mockInteractor.On("Update", mock.Anything).Return(errors.New("got an error")).Once()
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for k, v := range testTable {
		t.Run(k, func(t *testing.T) {
			v.arrange(t)

			_, err := client.Update(ctx, &models.UserPayload{})

			v.assert(t, err)
		})
	}
}
