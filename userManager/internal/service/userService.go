package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"userManager/internal/dto"
	"userManager/internal/repository"
	"userManager/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials error = errors.New("invalid credentials")
var ErrInvalidToken error = errors.New("invalid refresh token")

type UserService struct {
	userRepository         *repository.UserRepository
	refreshTokenRepository *repository.RefreshTokenRepository
	jwtSecret              string
	accessTTL              time.Duration
	refreshTTL             time.Duration
}

func NewUserService(
	userRepository *repository.UserRepository,
	refreshTokenRepository *repository.RefreshTokenRepository,
	jwtSecret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *UserService {
	return &UserService{
		userRepository:         userRepository,
		refreshTokenRepository: refreshTokenRepository,
		jwtSecret:              jwtSecret,
		accessTTL:              accessTTL,
		refreshTTL:             refreshTTL,
	}
}

func (us *UserService) Create(ctx context.Context, userDto dto.UserDto) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userDto.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	err = us.userRepository.Create(ctx, userDto.Login, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("unable to create a user: %w", err)
	}

	return nil
}

func (us *UserService) Login(ctx context.Context, userDto dto.UserDto) (*dto.AuthResponse, error) {
	user, err := us.userRepository.GetByLogin(ctx, userDto.Login)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(userDto.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := utils.GenerateAccessToken(user.Id, us.jwtSecret, us.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	refreshToken := utils.GenerateRefreshToken()
	tokenHash := utils.HashToken(refreshToken)

	expiresAt := time.Now().Add(us.refreshTTL)
	err = us.refreshTokenRepository.Create(ctx, user.Id, tokenHash, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (us *UserService) Refresh(ctx context.Context, rawRefreshToken string) (*dto.AuthResponse, error) {
	tokenHash := utils.HashToken(rawRefreshToken)
	token, err := us.refreshTokenRepository.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if token.Revoked {
		return nil, ErrInvalidToken
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	_ = us.refreshTokenRepository.Revoke(ctx, tokenHash)

	newAccessToken, err := utils.GenerateAccessToken(token.UserId, us.jwtSecret, us.accessTTL)
	newRefreshToken := utils.GenerateRefreshToken()
	newHash := utils.HashToken(newRefreshToken)

	if err := us.refreshTokenRepository.Create(ctx, token.UserId, newHash, time.Now().Add(us.refreshTTL)); err != nil {
		return nil, fmt.Errorf("failed to create new token: %w", err)
	}

	return &dto.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
