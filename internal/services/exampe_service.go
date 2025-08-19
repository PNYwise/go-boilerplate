package services

import (
	"context"
	"go-boilerplate/internal/configs"
	exampledtos "go-boilerplate/internal/dtos/example_dtos"
	"go-boilerplate/internal/entities"
	"go-boilerplate/internal/repositories"

	"github.com/go-playground/validator/v10"
)

// ExampleService defines the interface for example-related business logic.
type ExampleService interface {
	CreateExample(ctx context.Context, dto exampledtos.ExampleDTO) (int64, error)
}

type exampleService struct {
	exampleRepo repositories.ExampleRepository
	validator   *validator.Validate
	cfg         configs.Config
}

// NewExampleService creates a new instance of ExampleService.
// It initializes the service with the provided repository and configuration.
// The service is responsible for handling business logic related to ExampleEntity.
// It uses the repository to interact with the database and the validator for input validation.
// The ExampleService interface defines the methods that the service should implement.
// This allows for easier testing and flexibility in implementation.
func NewExampleService(r repositories.ExampleRepository, cfg configs.Config) ExampleService {
	return &exampleService{
		exampleRepo: r,
		validator:   validator.New(),
		cfg:         cfg,
	}
}

func (s *exampleService) CreateExample(ctx context.Context, o exampledtos.ExampleDTO) (int64, error) {
	if err := s.validator.Struct(o); err != nil {
		return 0, err
	}
	// Convert DTO to entity
	entity := &entities.ExampleEntity{
		UserID: o.UserID,
		Amount: o.Amount,
	}

	// Validate the entity
	if err := s.validator.Struct(entity); err != nil {
		return 0, err
	}
	return s.exampleRepo.Create(ctx, entity)
}
