package handlers

import (
	userdtos "go-boilerplate/internal/dtos/user_dtos"
	"go-boilerplate/internal/services"
	"go-boilerplate/internal/utils/logs"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserHandler handles HTTP requests related to users
type UserHandler struct {
	userSrv services.UserService
	tracer  trace.Tracer
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userSrv services.UserService) *UserHandler {
	return &UserHandler{
		userSrv: userSrv,
		tracer:  otel.Tracer("user-handler"),
	}
}

// CreateUser handles POST /users requests
func (h *UserHandler) CreateUser(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "UserHandler.CreateUser")
	defer span.End()

	var req userdtos.UserCreateDTO
	if err := c.BindJSON(&req); err != nil {
		logs.SpanError(ctx, span, err, "Failed to bind JSON request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add request attributes to span
	span.SetAttributes(
		attribute.String("user.username", req.Username),
		attribute.String("user.email", req.Email),
	)

	user, err := h.userSrv.CreateUser(ctx, req)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to create user",
			attribute.String("username", req.Username),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log successful creation
	span.SetAttributes(attribute.Int64("user.id", user.ID))
	logs.SpanInfo(ctx, span, "User created successfully",
		attribute.Int64("user_id", user.ID),
		attribute.String("username", user.Username),
	)

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) CreateUserWithRole(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "UserHandler.CreateUserWithRole")
	defer span.End()

	var req userdtos.UserCreateDTO
	if err := c.BindJSON(&req); err != nil {
		logs.SpanError(ctx, span, err, "Failed to bind JSON request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userSrv.CreateUserWithRole(ctx, 0, req)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to create user",
			attribute.String("username", req.Username),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)

}

// GetUserList handles GET /users requests
func (h *UserHandler) GetUserList(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "UserHandler.GetUserList")
	defer span.End()

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	span.SetAttributes(
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	users, err := h.userSrv.GetUserList(ctx, page, pageSize)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to get user list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log successful retrieval
	logs.SpanInfo(ctx, span, "Users retrieved successfully",
		attribute.Int("total_items", int(users.Pagination.TotalItems)),
	)

	c.JSON(http.StatusOK, users)
}

// GetUserByID handles GET /users/:id requests
func (h *UserHandler) GetUserByID(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "UserHandler.GetUserByID")
	defer span.End()

	idStr := c.Param("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logs.SpanError(ctx, span, err, "Invalid user ID format",
			attribute.String("user_id_param", idStr),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	span.SetAttributes(attribute.Int64("user.id", id))

	user, err := h.userSrv.GetUserByID(ctx, id)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to get user by ID",
			attribute.Int64("user_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Log successful retrieval
	logs.SpanInfo(ctx, span, "User retrieved successfully",
		attribute.Int64("user_id", user.ID),
		attribute.String("username", user.Username),
	)

	c.JSON(http.StatusOK, user)
}

// GetUserByUsername handles GET /users/username/:username requests
func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "UserHandler.GetUserByUsername")
	defer span.End()

	username := c.Param("username")
	span.SetAttributes(attribute.String("user.username", username))

	user, err := h.userSrv.GetUserByUsername(ctx, username)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to get user by username",
			attribute.String("username", username),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Log successful retrieval
	logs.SpanInfo(ctx, span, "User retrieved by username successfully",
		attribute.Int64("user_id", user.ID),
		attribute.String("username", user.Username),
	)

	c.JSON(http.StatusOK, user)
}

// DeleteUserByID handles DELETE /users/:id requests
func (h *UserHandler) DeleteUserByID(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "UserHandler.DeleteUserByID")
	defer span.End()

	idStr := c.Param("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logs.SpanError(ctx, span, err, "Invalid user ID format",
			attribute.String("user_id_param", idStr),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	span.SetAttributes(attribute.Int64("user.id", id))

	user, err := h.userSrv.DeleteUserByID(ctx, id)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to delete user by ID",
			attribute.Int64("user_id", id),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log successful retrieval
	logs.SpanInfo(ctx, span, "User deleted successfully",
		attribute.Int64("user_id", user.ID),
		attribute.String("username", user.Username),
	)

	c.JSON(http.StatusOK, user)
}
