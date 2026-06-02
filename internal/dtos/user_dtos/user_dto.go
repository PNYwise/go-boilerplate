package userdtos

import "time"

// UserCreateDTO represents data for creating a user
type UserCreateDTO struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
}

// UserUpdateDTO represents data for updating a user
type UserUpdateDTO struct {
	Username *string `json:"username" validate:"omitempty,min=3,max=50"`
	Email    *string `json:"email" validate:"omitempty,email"`
}

// UserResponseDTO represents user data in response
type UserResponseDTO struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaginationMeta struct {
    Page       int   `json:"page"`
    PageSize   int   `json:"page_size"`
    TotalItems int64 `json:"total_items"`
    TotalPages int   `json:"total_pages"`
}

// Wrapper for the paginated list response
type UserListResponseDTO struct {
    Users      []UserResponseDTO `json:"users"`
    Pagination PaginationMeta    `json:"pagination"`
}