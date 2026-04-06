package services

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	userdtos "go-boilerplate/internal/dtos/user_dtos"
	"go-boilerplate/internal/entities"
	"go-boilerplate/internal/repositories"
	"go-boilerplate/internal/utils/logs"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	tracer   trace.Tracer // OpenTelemetry tracer for this service
	v        *validator.Validate
}

// NewUserService creates a new user service instance with OpenTelemetry instrumentation
func NewUserService(
	userRepo repositories.UserRepository,
	cfg configs.Config,
	v *validator.Validate,
) UserService {
	return &userService{
		userRepo: userRepo,
		cfg:      cfg,
		tracer:   otel.Tracer("user-service"),
		v:        v,
	}
}

func (s *userService) CreateUser(ctx context.Context, dto userdtos.UserCreateDTO) (*userdtos.UserResponseDTO, error) {
	// Create span for this operation
	ctx, span := s.tracer.Start(ctx, "UserService.CreateUser")
	defer span.End()

	// Add user attributes to span
	span.SetAttributes(
		attribute.String("user.username", dto.Username),
		attribute.String("user.email", dto.Email),
	)

	// Log the start of user creation
	logs.LogInfo(ctx, "Starting user creation",
		attribute.String("username", dto.Username),
		attribute.String("email", dto.Email),
	)

	// Validate input
	if err := s.v.Struct(dto); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		span.AddEvent("User creation validation failed")
		logs.LogError(ctx, err, "User validation failed",
			attribute.String("username", dto.Username),
		)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, dto.Username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check existing user")
		span.AddEvent("Error checking existing user")
		logs.LogError(ctx, err, "Failed to check existing user",
			attribute.String("username", dto.Username),
		)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		err := fmt.Errorf("username '%s' already exists", dto.Username)
		span.RecordError(err)
		span.SetStatus(codes.Error, "username already exists")
		span.AddEvent("Username already exists", trace.WithAttributes(
			attribute.String("username", dto.Username),
		))
		logs.LogWarn(ctx, "Username already exists",
			attribute.String("username", dto.Username),
		)
		return nil, err
	}

	// Create user entity
	user := &entities.User{
		Username: dto.Username,
		Email:    dto.Email,
	}

	// Save to database
	id, err := s.userRepo.Create(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create user")
		span.AddEvent("Error creating user")
		logs.LogError(ctx, err, "Failed to create user in database",
			attribute.String("username", dto.Username),
			attribute.String("email", dto.Email),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Get created user to return complete data
	createdUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to retrieve created user")
		span.AddEvent("Error retrieving created user")
		logs.LogError(ctx, err, "Failed to retrieve created user",
			attribute.Int64("user_id", id),
		)
		return nil, fmt.Errorf("failed to retrieve created user: %w", err)
	}

	// Add success attributes to span
	span.SetAttributes(attribute.Int64("user.id", id))
	span.SetStatus(codes.Ok, "user created successfully")
	span.AddEvent("User created successfully", trace.WithAttributes(
		attribute.Int64("user_id", id),
		attribute.String("username", dto.Username),
	))

	// Log successful user creation
	logs.LogInfo(ctx, "User created successfully",
		attribute.Int64("user_id", id),
		attribute.String("username", dto.Username),
		attribute.String("email", dto.Email),
	)

	return &userdtos.UserResponseDTO{
		ID:        createdUser.ID,
		Username:  createdUser.Username,
		Email:     createdUser.Email,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
	}, nil
}

func (s *userService) GetUserByID(ctx context.Context, id int64) (*userdtos.UserResponseDTO, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUserByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", id))

	logs.LogInfo(ctx, "Retrieving user by ID",
		attribute.Int64("user_id", id),
	)

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user")
		span.AddEvent("Error retrieving user by ID")
		logs.LogError(ctx, err, "Failed to retrieve user from database",
			attribute.Int64("user_id", id),
		)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		err := fmt.Errorf("user with ID %d not found", id)
		span.SetStatus(codes.Error, "user not found")
		span.AddEvent("User not found")
		logs.LogWarn(ctx, "User not found",
			attribute.Int64("user_id", id),
		)
		return nil, err
	}

	span.SetStatus(codes.Ok, "user retrieved successfully")
	span.AddEvent("User retrieved successfully")

	logs.LogInfo(ctx, "User retrieved successfully",
		attribute.Int64("user_id", id),
		attribute.String("username", user.Username),
	)

	return &userdtos.UserResponseDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*userdtos.UserResponseDTO, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUserByUsername")
	defer span.End()

	span.SetAttributes(attribute.String("user.username", username))

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user")
		span.AddEvent("Error retrieving user by username")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		err := fmt.Errorf("user '%s' not found", username)
		span.SetStatus(codes.Error, "user not found")
		span.AddEvent("User not found")
		return nil, err
	}

	span.SetStatus(codes.Ok, "user retrieved successfully")
	span.AddEvent("User retrieved successfully")

	return &userdtos.UserResponseDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
