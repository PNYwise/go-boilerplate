package handlers

import (
	exampledtos "go-boilerplate/internal/dtos/example_dtos"
	"go-boilerplate/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ExampleHandler handles HTTP requests related to examples.
// It uses the ExampleService to perform operations on example entities.
// The handler methods are responsible for binding request data, validating it, and calling the service methods.
// This approach promotes separation of concerns and makes the code more maintainable.
type ExampleHandler struct {
	exampleSrv services.ExampleService // This should be the interface type for the service
}

// NewExampleHandler creates a new ExampleHandler with the provided ExampleService.
// It initializes the handler with the service, allowing it to handle HTTP requests related to examples.
// The handler methods will use this service to perform business logic operations.
// This approach promotes separation of concerns and makes the code more maintainable.
func NewExampleHandler(exampleSrv services.ExampleService) *ExampleHandler {
	return &ExampleHandler{
		exampleSrv: exampleSrv,
	}
}

// CreateExample handles the HTTP POST request to create a new example entity.
// It binds the incoming JSON request to an ExampleDTO, validates it, and calls the service to create the entity.
// If successful, it returns a 201 Created response with the new entity's ID.
func (h *ExampleHandler) CreateExample(c *gin.Context) {
	var in exampledtos.ExampleDTO
	if err := c.BindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.exampleSrv.CreateExample(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// Add methods to handle HTTP requests, such as CreateExample, UpdateExample, etc.
// Each method should correspond to a specific route and handle the request logic.
