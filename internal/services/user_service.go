package services

import (
	"context"
	"database/sql"
	"fmt"
	"go-boilerplate/internal/configs"
	userdtos "go-boilerplate/internal/dtos/user_dtos"
	"go-boilerplate/internal/entities"
	"go-boilerplate/internal/repositories"
	dbtransaction "go-boilerplate/internal/utils/db-transaction"
	"go-boilerplate/internal/utils/logs"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserService defines user-related business operations
type UserService interface {
	CreateUser(ctx context.Context, dto userdtos.UserCreateDTO) (*userdtos.UserResponseDTO, error)
	GetUserList(ctx context.Context, page, pageSize int) (*userdtos.UserListResponseDTO, error)
	GetUserByID(ctx context.Context, id int64) (*userdtos.UserResponseDTO, error)
	UpdateUserByID(ctx context.Context, id int64, dto userdtos.UserUpdateDTO) (*userdtos.UserResponseDTO, error)
	DeleteUserByID(ctx context.Context, id int64) (*userdtos.UserResponseDTO, error)
	GetUserByUsername(ctx context.Context, username string) (*userdtos.UserResponseDTO, error)
	CreateUserWithRole(ctx context.Context, id int64, dto userdtos.UserCreateDTO) (*userdtos.UserResponseDTO, error)
}

type userService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
	dbtx     dbtransaction.DbTransactionUtil
	cfg      configs.Config
	tracer   trace.Tracer // OpenTelemetry tracer for this service
	v        *validator.Validate
}

// NewUserService creates a new user service instance with OpenTelemetry instrumentation
func NewUserService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	dbtx dbtransaction.DbTransactionUtil,
	cfg configs.Config,
	v *validator.Validate,
) UserService {
	return &userService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		dbtx:     dbtx,
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

	// Validate input
	if err := s.v.Struct(dto); err != nil {
		logs.SpanError(ctx, span, err, "User validation failed",
			attribute.String("username", dto.Username),
		)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, dto.Username)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to check existing user",
			attribute.String("username", dto.Username),
		)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		err := fmt.Errorf("username '%s' already exists", dto.Username)
		span.RecordError(err)
		logs.SpanWarn(ctx, span, "Username already exists",
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

	tx, err := s.dbtx.InitTx(ctx, nil)

	id, err := s.userRepo.Create(ctx, tx, user)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to create user in database",
			attribute.String("username", dto.Username),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	s.dbtx.CommitTx(tx)

	// Get created user to return complete data
	createdUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to retrieve created user",
			attribute.Int64("user_id", id),
		)
		return nil, fmt.Errorf("failed to retrieve created user: %w", err)
	}

	// Add success attributes to span
	span.SetAttributes(attribute.Int64("user.id", id))

	// Log successful user creation
	logs.SpanInfo(ctx, span, "User created successfully",
		attribute.Int64("user_id", id),
		attribute.String("username", dto.Username),
	)

	return &userdtos.UserResponseDTO{
		ID:        createdUser.ID,
		Username:  createdUser.Username,
		Email:     createdUser.Email,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
	}, nil
}

func (s *userService) GetUserList(ctx context.Context, page, pageSize int) (*userdtos.UserListResponseDTO, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUserList")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	users, totalItems, err := s.userRepo.GetList(ctx, page, pageSize)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to retrieve user from database")
		return nil, fmt.Errorf("failed to get user list: %w", err)
	}

	userResponses := []userdtos.UserResponseDTO{}
	for _, user := range users {
		userResponses = append(userResponses, userdtos.UserResponseDTO{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	}

	logs.SpanInfo(ctx, span, "User list retrieved successfully",
		attribute.Int("count", len(userResponses)),
	)

	return &userdtos.UserListResponseDTO{
		Users: userResponses,
		Pagination: userdtos.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *userService) GetUserByID(ctx context.Context, id int64) (*userdtos.UserResponseDTO, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUserByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", id))

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to retrieve user from database",
			attribute.Int64("user_id", id),
		)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		err := fmt.Errorf("user with ID %d not found", id)
		logs.SpanWarn(ctx, span, "User not found",
			attribute.Int64("user_id", id),
		)
		return nil, err
	}

	logs.SpanInfo(ctx, span, "User retrieved successfully",
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
		logs.SpanError(ctx, span, err, "Failed to get user by username",
			attribute.String("username", username),
		)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		err := fmt.Errorf("user '%s' not found", username)
		logs.SpanWarn(ctx, span, "User not found by username",
			attribute.String("username", username),
		)
		return nil, err
	}

	logs.SpanInfo(ctx, span, "User retrieved by username successfully",
		attribute.String("username", user.Username),
		attribute.Int64("user_id", user.ID),
	)

	return &userdtos.UserResponseDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (s *userService) CreateUserWithRole(ctx context.Context, id int64, dto userdtos.UserCreateDTO) (*userdtos.UserResponseDTO, error) {
	// update user and role

	// init tx
	tx, err := s.dbtx.InitTx(ctx, &sql.TxOptions{
		Isolation: sql.IsolationLevel(sql.LevelLinearizable),
		ReadOnly:  false,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize transaction: %w", err)
	}

	_, err = s.userRepo.Create(ctx, tx, &entities.User{
		Username: dto.Username,
		Email:    dto.Email,
	})
	if err != nil {
		s.dbtx.RollbackTx(tx)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	_, err = s.roleRepo.Create(ctx, tx, &entities.Role{
		Name: "admin",
	})
	if err != nil {
		s.dbtx.RollbackTx(tx)
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	s.dbtx.CommitTx(tx)

	return nil, nil
}

func (s *userService) DeleteUserByID(ctx context.Context, id int64) (*userdtos.UserResponseDTO, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.DeleteUserByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", id))

	user, err := s.userRepo.DeleteByID(ctx, id)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to retrieve user from database",
			attribute.Int64("user_id", id),
		)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	logs.SpanInfo(ctx, span, "User deleted successfully",
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

func (s *userService) UpdateUserByID(ctx context.Context, id int64, dto userdtos.UserUpdateDTO) (*userdtos.UserResponseDTO, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.UpdateUserByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", id))

	// 1. Validate DTO
	if err := s.v.Struct(dto); err != nil {
		logs.SpanError(ctx, span, err, "User update validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Fetch existing user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to retrieve user from database", attribute.Int64("user_id", id))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		err := fmt.Errorf("user with ID %d not found", id)
		logs.SpanWarn(ctx, span, "User not found", attribute.Int64("user_id", id))
		return nil, err
	}

	logs.SpanInfo(ctx, span, "User retrieved successfully", attribute.Int64("user_id", id), attribute.String("username", user.Username))

	// 3. Uniqueness checks and value setting
	hasChanges := false

	if dto.Username != nil && *dto.Username != "" && *dto.Username != user.Username {
		existing, err := s.userRepo.GetByUsername(ctx, *dto.Username)
		if err != nil {
			logs.SpanError(ctx, span, err, "Failed to check existing username")
			return nil, fmt.Errorf("failed to check username: %w", err)
		}
		if existing != nil && existing.ID != user.ID {
			err := fmt.Errorf("username '%s' already exists", *dto.Username)
			logs.SpanWarn(ctx, span, "Username already exists", attribute.String("username", *dto.Username))
			return nil, err
		}
		user.Username = *dto.Username
		hasChanges = true
	}

	if dto.Email != nil && *dto.Email != "" && *dto.Email != user.Email {
		existing, err := s.userRepo.GetByEmail(ctx, *dto.Email)
		if err != nil {
			logs.SpanError(ctx, span, err, "Failed to check existing email")
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if existing != nil && existing.ID != user.ID {
			err := fmt.Errorf("email '%s' already exists", *dto.Email)
			logs.SpanWarn(ctx, span, "Email already exists", attribute.String("email", *dto.Email))
			return nil, err
		}
		user.Email = *dto.Email
		hasChanges = true
	}

	// 4. Perform update within transaction if there are changes
	if hasChanges {
		tx, err := s.dbtx.InitTx(ctx, nil)
		if err != nil {
			logs.SpanError(ctx, span, err, "Failed to initialize transaction")
			return nil, fmt.Errorf("failed to initialize transaction: %w", err)
		}
		defer func() {
			if err != nil {
				s.dbtx.RollbackTx(tx)
			}
		}()

		err = s.userRepo.Update(ctx, tx, user)
		if err != nil {
			logs.SpanError(ctx, span, err, "Failed to update user in database")
			return nil, fmt.Errorf("failed to update user: %w", err)
		}

		s.dbtx.CommitTx(tx)

		// Get updated user to return complete data
		updatedUser, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			logs.SpanError(ctx, span, err, "Failed to retrieve updated user",
				attribute.Int64("user_id", id),
			)
			return nil, fmt.Errorf("failed to retrieve updated user: %w", err)
		}
		user = updatedUser
	}

	logs.SpanInfo(ctx, span, "User updated successfully",
		attribute.Int64("user_id", user.ID),
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
