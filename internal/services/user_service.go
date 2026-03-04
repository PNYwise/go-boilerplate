package services

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	userdtos "go-boilerplate/internal/dtos/user_dtos"
	"go-boilerplate/internal/entities"
	"go-boilerplate/internal/repositories"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// UserService defines user-related business operations
type UserService interface {
	CreateUser(ctx context.Context, dto userdtos.UserCreateDTO) (*userdtos.UserResponseDTO, error)
	GetUserByID(ctx context.Context, id int64) (*userdtos.UserResponseDTO, error)
	GetUserByUsername(ctx context.Context, username string) (*userdtos.UserResponseDTO, error)
}

type userService struct {
	userRepo repositories.UserRepository
	cfg      configs.Config
	logger   *zap.Logger
	v        *validator.Validate
}

// NewUserService creates a new user service instance
func NewUserService(
	userRepo repositories.UserRepository,
	cfg configs.Config,
	logger *zap.Logger,
	v *validator.Validate,
) UserService {
	return &userService{
		userRepo: userRepo,
		cfg:      cfg,
		logger:   logger,
		v:        v,
	}
}

func (s *userService) CreateUser(ctx context.Context, dto userdtos.UserCreateDTO) (*userdtos.UserResponseDTO, error) {
	// Validate input
	if err := s.v.Struct(dto); err != nil {
		s.logger.Error("User creation validation failed", zap.Error(err))
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, dto.Username)
	if err != nil {
		s.logger.Error("Error checking existing user", zap.Error(err))
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("username '%s' already exists", dto.Username)
	}

	// Create user entity
	user := &entities.User{
		Username: dto.Username,
		Email:    dto.Email,
	}

	// Save to database
	id, err := s.userRepo.Create(ctx, user)
	if err != nil {
		s.logger.Error("Error creating user", zap.Error(err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Get created user to return complete data
	createdUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Error retrieving created user", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve created user: %w", err)
	}

	s.logger.Info("User created successfully", 
		zap.Int64("user_id", id), 
		zap.String("username", dto.Username))

	return &userdtos.UserResponseDTO{
		ID:        createdUser.ID,
		Username:  createdUser.Username,
		Email:     createdUser.Email,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
	}, nil
}

func (s *userService) GetUserByID(ctx context.Context, id int64) (*userdtos.UserResponseDTO, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Error retrieving user by ID", zap.Int64("user_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user with ID %d not found", id)
	}

	return &userdtos.UserResponseDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*userdtos.UserResponseDTO, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		s.logger.Error("Error retrieving user by username", zap.String("username", username), zap.Error(err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user '%s' not found", username)
	}

	return &userdtos.UserResponseDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}