package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"wikilite/pkg/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateUser_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	password := "password123"
	input := &CreateUserInput{}
	input.Body.Name = "New User"
	input.Body.Email = "new@user.com"
	input.Body.Password = &password
	input.Body.Role = int(models.WRITE)

	resp, err := server.handleCreateUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "New User", resp.Body.User.Name)
	assert.Equal(t, "new@user.com", resp.Body.User.Email)
	assert.Equal(t, models.WRITE, resp.Body.User.Role)

	user, err := db.GetUserByEmail(context.Background(), "new@user.com")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.NotEmpty(t, user.Hash)
}

func TestHandleCreateUser_External_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &CreateUserInput{}
	input.Body.Name = "External User"
	input.Body.Email = "external@user.com"
	input.Body.IsExternal = true
	input.Body.Role = int(models.READ)

	resp, err := server.handleCreateUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "External User", resp.Body.User.Name)
	assert.True(t, resp.Body.User.IsExternal)

	user, err := db.GetUserByEmail(context.Background(), "external@user.com")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Empty(t, user.Hash)
}

func TestHandleCreateUser_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	ctx := contextWithUser(user)

	input := &CreateUserInput{}
	input.Body.Name = "New User"
	input.Body.Email = "new@user.com"

	resp, err := server.handleCreateUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleCreateUser_MissingPassword(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &CreateUserInput{}
	input.Body.Name = "New User"
	input.Body.Email = "new@user.com"
	input.Body.IsExternal = false

	resp, err := server.handleCreateUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 400, humaErr.Status)
}

func TestHandleGetUser_Success_Self(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := contextWithUser(user)

	input := &UserEmailInput{Email: user.Email}
	resp, err := server.handleGetUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, user.Name, resp.Body.User.Name)
}

func TestHandleGetUser_Success_Admin(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &UserEmailInput{Email: user.Email}
	resp, err := server.handleGetUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, user.Name, resp.Body.User.Name)
}

func TestHandleGetUser_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()

	input := &UserEmailInput{Email: "test@example.com"}
	resp, err := server.handleGetUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleGetUser_Forbidden(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user1 := &models.User{Name: "User One", Email: "user1@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user1)
	require.NoError(t, err)

	user2 := &models.User{Name: "User Two", Email: "user2@example.com", Role: models.WRITE}
	err = db.CreateUser(context.Background(), user2)
	require.NoError(t, err)

	ctx := contextWithUser(user1)

	input := &UserEmailInput{Email: user2.Email}
	resp, err := server.handleGetUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleGetUser_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := contextWithUser(user)

	input := &UserEmailInput{Email: "non-existent@user.com"}
	resp, err := server.handleGetUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleGetUser_NotFound_AsAdmin(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &UserEmailInput{Email: "non-existent@user.com"}
	resp, err := server.handleGetUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleGetUserByID_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &UserIDInput{ID: user.Id}
	resp, err := server.handleGetUserByID(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, user.Name, resp.Body.User.Name)
}

func TestHandleGetUserByID_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := contextWithUser(user)

	input := &UserIDInput{ID: user.Id}
	resp, err := server.handleGetUserByID(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleGetUserByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &UserIDInput{ID: 999}
	resp, err := server.handleGetUserByID(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleUpdateUser_Success_Self(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := contextWithUser(user)
	newName := "Updated Name"
	input := &UpdateUserInput{
		Email: user.Email,
		Body: struct {
			Name     *string `json:"name,omitempty"`
			Email    *string `format:"email"            json:"email,omitempty"`
			Password *string `json:"password,omitempty"`
			Role     *int    `json:"role,omitempty"`
			Disabled *bool   `json:"disabled,omitempty"`
		}{Name: &newName},
	}

	resp, err := server.handleUpdateUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, newName, resp.Body.User.Name)

	updatedUser, err := db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	assert.Equal(t, newName, updatedUser.Name)
}

func TestHandleUpdateUser_Success_Admin(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	newRole := int(models.ADMIN)
	input := &UpdateUserInput{
		Email: user.Email,
		Body: struct {
			Name     *string `json:"name,omitempty"`
			Email    *string `format:"email"            json:"email,omitempty"`
			Password *string `json:"password,omitempty"`
			Role     *int    `json:"role,omitempty"`
			Disabled *bool   `json:"disabled,omitempty"`
		}{Role: &newRole},
	}

	resp, err := server.handleUpdateUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, models.ADMIN, resp.Body.User.Role)

	updatedUser, err := db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	assert.Equal(t, models.ADMIN, updatedUser.Role)
}

func TestHandleUpdateUser_Forbidden(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user1 := &models.User{Name: "User One", Email: "user1@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user1)
	require.NoError(t, err)

	user2 := &models.User{Name: "User Two", Email: "user2@example.com", Role: models.WRITE}
	err = db.CreateUser(context.Background(), user2)
	require.NoError(t, err)

	ctx := contextWithUser(user1)
	newName := "Updated Name"
	input := &UpdateUserInput{
		Email: user2.Email,
		Body: struct {
			Name     *string `json:"name,omitempty"`
			Email    *string `format:"email"            json:"email,omitempty"`
			Password *string `json:"password,omitempty"`
			Role     *int    `json:"role,omitempty"`
			Disabled *bool   `json:"disabled,omitempty"`
		}{Name: &newName},
	}

	resp, err := server.handleUpdateUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleUpdateUser_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()
	newName := "Updated Name"
	input := &UpdateUserInput{
		Email: "test@example.com",
		Body: struct {
			Name     *string `json:"name,omitempty"`
			Email    *string `format:"email"            json:"email,omitempty"`
			Password *string `json:"password,omitempty"`
			Role     *int    `json:"role,omitempty"`
			Disabled *bool   `json:"disabled,omitempty"`
		}{Name: &newName},
	}

	resp, err := server.handleUpdateUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleUpdateUser_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	newName := "Updated Name"
	input := &UpdateUserInput{
		Email: "non-existent@user.com",
		Body: struct {
			Name     *string `json:"name,omitempty"`
			Email    *string `format:"email"            json:"email,omitempty"`
			Password *string `json:"password,omitempty"`
			Role     *int    `json:"role,omitempty"`
			Disabled *bool   `json:"disabled,omitempty"`
		}{Name: &newName},
	}

	resp, err := server.handleUpdateUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleDeleteUser_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "To Be Deleted", Email: "delete@me.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &UserEmailInput{Email: user.Email}
	resp, err := server.handleDeleteUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusNoContent, resp.Status)

	deletedUser, err := db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	assert.Nil(t, deletedUser)
}

func TestHandleDeleteUser_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := contextWithUser(user)

	input := &UserEmailInput{Email: user.Email}
	resp, err := server.handleDeleteUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleDeleteUser_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &UserEmailInput{Email: "non-existent@user.com"}
	resp, err := server.handleDeleteUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleDeleteUser_Self(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin, err := db.GetUserByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)
	ctx := contextWithUser(admin)

	input := &UserEmailInput{Email: admin.Email}
	resp, err := server.handleDeleteUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 400, humaErr.Status)
}
