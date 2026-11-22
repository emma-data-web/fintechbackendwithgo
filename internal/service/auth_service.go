package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"fintech-system/internal/config"
	"fintech-system/internal/domain"
	"fintech-system/internal/repository"
	"fintech-system/pkg/hasher"
	"fintech-system/pkg/jwt"
)

type AuthService interface {
	Register(ctx context.Context, name, email, password, address string, role domain.Role) (*domain.User, error)
	Login(ctx context.Context, email, password string) (string, *domain.User, error)
}

type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *authService) Register(ctx context.Context, name, email, password, address string, role domain.Role) (*domain.User, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	hash, err := hasher.HashPassword(password)
	if err != nil {
		return nil, err
	}

	if role == "" {
		role = domain.RoleCustomer
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New().String(),
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		Address:      address,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	if !hasher.CheckPassword(password, user.PasswordHash) {
		return "", nil, domain.ErrInvalidCredentials
	}

	duration := time.Duration(s.cfg.TokenExpiryHours) * time.Hour
	token, err := jwt.GenerateToken(user.ID, user.Email, string(user.Role), s.cfg.JWTSecret, duration)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}
