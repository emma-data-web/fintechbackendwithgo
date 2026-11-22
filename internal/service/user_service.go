package service

import (
	"context"

	"fintech-system/internal/domain"
	"fintech-system/internal/repository"
)

type UserService interface {
	GetProfile(ctx context.Context, id string) (*domain.User, error)
	ListUsers(ctx context.Context, page, limit int) ([]*domain.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *userService) ListUsers(ctx context.Context, page, limit int) ([]*domain.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	return s.userRepo.List(ctx, limit, offset)
}
