package user

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	userpb "one_adventure_rpc/proto/user"
	servicetoken "one_adventure_servicekit/token"
	appconfig "user/internal/config"
	"user/internal/logic"
	"user/internal/model/entity"
	"user/internal/service"
)

type userService struct {
	userpb.UnimplementedUserServiceServer
	userService        service.UserService
	tokenGenerator     servicetoken.Generator
	tokenVerifier      servicetoken.Verifier
	refreshTokenStore  refreshTokenStore
	refreshTokenExpire time.Duration
	now                func() time.Time
}

func newUserService(businessService service.UserService, generator servicetoken.Generator, verifier servicetoken.Verifier, store refreshTokenStore, refreshTokenExpire time.Duration) userpb.UserServiceServer {
	return &userService{
		userService: businessService, tokenGenerator: generator, tokenVerifier: verifier,
		refreshTokenStore: store, refreshTokenExpire: refreshTokenExpire, now: time.Now,
	}
}

func newDefaultUserService(config appconfig.JWTConfig) (userpb.UserServiceServer, error) {
	generator, err := servicetoken.NewRS256GeneratorFromFile(config.PrivateKeyPath, config.Issuer, config.AccessTokenExpire)
	if err != nil {
		return nil, err
	}
	verifier, err := servicetoken.NewRS256VerifierFromFile(config.PublicKeyPath, config.Issuer)
	if err != nil {
		return nil, err
	}
	return newUserService(
		logic.NewUserService(), generator, verifier,
		redisRefreshTokenStore{keyPrefix: config.RefreshTokenKeyPrefix},
		config.RefreshTokenExpire,
	), nil
}

func (s *userService) Login(ctx context.Context, request *userpb.LoginReq) (*userpb.LoginResp, error) {
	if request == nil || strings.TrimSpace(request.GetUsername()) == "" || request.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}
	user, err := s.userService.FindByUsername(ctx, strings.TrimSpace(request.GetUsername()))
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid username or password")
		}
		return nil, status.Error(codes.Internal, "query user failed")
	}
	if !passwordMatches(request.GetPassword(), user.Password) {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}
	if user.Status != 0 {
		return nil, status.Error(codes.PermissionDenied, "user is disabled")
	}
	userInfo := userInfoFromEntity(user)
	accessToken, err := s.tokenGenerator.Generate(userInfo)
	if err != nil {
		return nil, status.Error(codes.Internal, "create access token failed")
	}
	refreshToken := uuid.NewString()
	record := RefreshTokenRecord{User: userInfo, ExpiresAt: s.now().Add(s.refreshTokenExpire)}
	if err = s.refreshTokenStore.Save(ctx, refreshToken, record); err != nil {
		return nil, status.Error(codes.Internal, "store refresh token failed")
	}
	return &userpb.LoginResp{Token: accessToken, RefreshToken: refreshToken}, nil
}

func passwordMatches(password, storedHash string) bool {
	calculated := []byte(passwordMD5(password))
	stored := []byte(strings.TrimSpace(storedHash))
	return len(calculated) == len(stored) && subtle.ConstantTimeCompare(calculated, stored) == 1
}

func passwordMD5(password string) string {
	return fmt.Sprintf("%032x", md5.Sum([]byte(password)))
}

func userInfoFromEntity(user *entity.User) servicetoken.UserInfo {
	return servicetoken.UserInfo{ID: user.Id, Username: user.Username, Status: user.Status}
}

func (s *userService) UserInfoGet(_ context.Context, request *userpb.UserInfoGetReq) (*userpb.UserInfoGetResp, error) {
	if request == nil || strings.TrimSpace(request.GetToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	claims, err := s.tokenVerifier.VerifyAndParse(strings.TrimSpace(request.GetToken()))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "token is invalid or expired")
	}
	return &userpb.UserInfoGetResp{UserId: claims.UserInfo.ID}, nil
}

func (s *userService) RefreshToken(ctx context.Context, request *userpb.RefreshTokenReq) (*userpb.RefreshTokenResp, error) {
	if request == nil || strings.TrimSpace(request.GetRefreshToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}
	refreshToken := strings.TrimSpace(request.GetRefreshToken())
	if _, err := uuid.Parse(refreshToken); err != nil {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is invalid")
	}
	record, err := s.refreshTokenStore.Get(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, errRefreshTokenNotFound) {
			return nil, status.Error(codes.Unauthenticated, "refresh token is invalid or expired")
		}
		return nil, status.Error(codes.Internal, "read refresh token failed")
	}
	if !record.ExpiresAt.After(s.now()) {
		return nil, status.Error(codes.Unauthenticated, "refresh token is invalid or expired")
	}
	if record.User.Status != 0 {
		return nil, status.Error(codes.PermissionDenied, "user is disabled")
	}
	accessToken, err := s.tokenGenerator.Generate(record.User)
	if err != nil {
		return nil, status.Error(codes.Internal, "create access token failed")
	}
	return &userpb.RefreshTokenResp{Token: accessToken}, nil
}
