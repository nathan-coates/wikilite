package api

import (
	"context"
	"net/http"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/danielgtaylor/huma/v2"
)

// UserEmailInput represents the input for getting a user by email.
type UserEmailInput struct {
	Email string `doc:"The email of the user" format:"email" path:"email"`
}

// UserIDInput represents the input for getting a user by ID.
type UserIDInput struct {
	ID int `doc:"The numeric ID of the user" path:"id"`
}

// CreateUserInput represents the input for creating a new user.
type CreateUserInput struct {
	Body struct {
		Password   *string `doc:"Required for local users. Omit for external IDP users." json:"password,omitempty"`
		Name       string  `json:"name"                                                  required:"true"`
		Email      string  `format:"email"                                               json:"email"                                                 required:"true"`
		Role       int     `default:"1"                                                  doc:"1=Read, 2=Write, 3=Admin"                               json:"role"`
		IsExternal bool    `default:"false"                                              doc:"Set to true if this user is managed by an external IDP" json:"isExternal"`
	}
}

// UpdateUserInput represents the input for updating a user.
type UpdateUserInput struct {
	Body struct {
		Name     *string `json:"name,omitempty"`
		Email    *string `format:"email"            json:"email,omitempty"`
		Password *string `json:"password,omitempty"`
		Role     *int    `json:"role,omitempty"`
		Disabled *bool   `json:"disabled,omitempty"`
	}
	Email string `doc:"The email of the user" format:"email" path:"email"`
}

// SafeUser hides the password hash.
type SafeUser struct {
	Name       string          `json:"name"`
	Email      string          `json:"email"`
	Id         int             `json:"id"`
	Role       models.UserRole `json:"role"`
	IsExternal bool            `json:"isExternal"`
	Disabled   bool            `json:"disabled"`
}

// UserOutput represents the output for a single user.
type UserOutput struct {
	Body struct {
		User *SafeUser `json:"user"`
	}
}

// UserListOutput represents the output for a list of users.
type UserListOutput struct {
	Body struct {
		Users []*SafeUser `json:"users"`
	}
}

// registerUserRoutes registers the user routes with the API.
func (s *Server) registerUserRoutes() {
	huma.Register(s.api, huma.Operation{
		OperationID: "create-user",
		Method:      http.MethodPost,
		Path:        "/api/users",
		Summary:     "Create User",
		Description: "Create a local user (requires password) or pre-provision an external IDP user.",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleCreateUser)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        "/api/users/{email}",
		Summary:     "Get User",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetUser)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-user-by-id",
		Method:      http.MethodGet,
		Path:        "/api/users/id/{id}",
		Summary:     "Get User By ID",
		Description: "Retrieve user details by numeric ID (Admin only).",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetUserByID)

	huma.Register(s.api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPatch,
		Path:        "/api/users/{email}",
		Summary:     "Update User",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleUpdateUser)

	huma.Register(s.api, huma.Operation{
		OperationID: "delete-user",
		Method:      http.MethodDelete,
		Path:        "/api/users/{email}",
		Summary:     "Delete User",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleDeleteUser)
}

// toSafeUser converts a user model to a safe user model.
func toSafeUser(u *models.User) *SafeUser {
	return &SafeUser{
		Id:         u.Id,
		Name:       u.Name,
		Email:      u.Email,
		Role:       u.Role,
		IsExternal: u.IsExternal,
		Disabled:   u.Disabled,
	}
}

// handleCreateUser handles the creation of a new user.
func (s *Server) handleCreateUser(
	ctx context.Context,
	input *CreateUserInput,
) (*UserOutput, error) {
	admin := getAdminUserFromContext(ctx)
	if admin == nil {
		return nil, huma.Error403Forbidden("Only admins can create users")
	}

	var hash string

	if input.Body.IsExternal {
		hash = ""
	} else {
		if input.Body.Password == nil || *input.Body.Password == "" {
			return nil, huma.Error400BadRequest("Password is required for local users")
		}

		hashed, err := utils.HashPassword(*input.Body.Password)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to process password", err)
		}

		hash = hashed
	}

	newUser := &models.User{
		Name:       input.Body.Name,
		Email:      input.Body.Email,
		Hash:       hash,
		IsExternal: input.Body.IsExternal,
		Role:       models.UserRole(input.Body.Role),
	}

	err := s.db.CreateUser(ctx, newUser)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create user", err)
	}

	resp := &UserOutput{}
	resp.Body.User = toSafeUser(newUser)

	return resp, nil
}

// handleGetUser handles getting a user by email.
func (s *Server) handleGetUser(ctx context.Context, input *UserEmailInput) (*UserOutput, error) {
	reqUser := getUserFromContext(ctx)
	if reqUser == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	if reqUser.Email != input.Email && reqUser.Role != models.ADMIN {
		return nil, huma.Error403Forbidden("You can only view your own profile")
	}

	user, err := s.db.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if user == nil {
		return nil, huma.Error404NotFound("User not found")
	}

	resp := &UserOutput{}
	resp.Body.User = toSafeUser(user)

	return resp, nil
}

// handleGetUserByID handles getting a user by ID.
func (s *Server) handleGetUserByID(ctx context.Context, input *UserIDInput) (*UserOutput, error) {
	user := getAdminUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error403Forbidden("Only admins can view users by ID")
	}

	user, err := s.db.GetUserByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if user == nil {
		return nil, huma.Error404NotFound("User not found")
	}

	resp := &UserOutput{}
	resp.Body.User = toSafeUser(user)

	return resp, nil
}

// handleUpdateUser handles updating a user.
func (s *Server) handleUpdateUser(
	ctx context.Context,
	input *UpdateUserInput,
) (*UserOutput, error) {
	reqUser := getUserFromContext(ctx)
	if reqUser == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	targetUser, err := s.db.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if targetUser == nil {
		return nil, huma.Error404NotFound("User not found")
	}

	isAdmin := reqUser.Role == models.ADMIN
	isSelf := reqUser.Id == targetUser.Id

	if !isAdmin && !isSelf {
		return nil, huma.Error403Forbidden("You cannot update this user")
	}

	var cols []string

	if input.Body.Name != nil {
		targetUser.Name = *input.Body.Name

		cols = append(cols, "name")
	}

	if input.Body.Email != nil {
		targetUser.Email = *input.Body.Email

		cols = append(cols, "email")
	}

	if input.Body.Password != nil {
		hashed, err := utils.HashPassword(*input.Body.Password)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to process password", err)
		}

		targetUser.Hash = hashed

		cols = append(cols, "hash")
	}

	if isAdmin {
		if input.Body.Role != nil {
			targetUser.Role = models.UserRole(*input.Body.Role)

			cols = append(cols, "role")
		}

		if input.Body.Disabled != nil {
			targetUser.Disabled = *input.Body.Disabled

			cols = append(cols, "disabled")
		}
	}

	if len(cols) > 0 {
		err := s.db.UpdateUser(ctx, targetUser, cols...)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to update user", err)
		}
	}

	resp := &UserOutput{}
	resp.Body.User = toSafeUser(targetUser)

	return resp, nil
}

// handleDeleteUser handles deleting a user.
func (s *Server) handleDeleteUser(
	ctx context.Context,
	input *UserEmailInput,
) (*struct{ Status int }, error) {
	reqUser := getAdminUserFromContext(ctx)
	if reqUser == nil {
		return nil, huma.Error403Forbidden("Only admins can delete users")
	}

	targetUser, err := s.db.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if targetUser == nil {
		return nil, huma.Error404NotFound("User not found")
	}

	if reqUser.Id == targetUser.Id {
		return nil, huma.Error400BadRequest("You cannot delete yourself")
	}

	err = s.db.DeleteUser(ctx, targetUser.Id)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete user", err)
	}

	return &struct{ Status int }{Status: http.StatusNoContent}, nil
}
