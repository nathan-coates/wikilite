package db

import (
	"context"
	"testing"
	"wikilite/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBackupCode(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Hash:  "hashedpassword",
		Role:  models.READ,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	backupCode := &models.BackupCode{
		UserId: user.Id,
		Code:   "12345678",
		Used:   false,
	}

	err = db.CreateBackupCode(ctx, backupCode)
	require.NoError(t, err)
	assert.NotZero(t, backupCode.Id)
}

func TestCreateBackupCodes(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Hash:  "hashedpassword",
		Role:  models.READ,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	backupCodes := []*models.BackupCode{
		{UserId: user.Id, Code: "12345678", Used: false},
		{UserId: user.Id, Code: "87654321", Used: false},
		{UserId: user.Id, Code: "11111111", Used: false},
	}

	err = db.CreateBackupCodes(ctx, backupCodes)
	require.NoError(t, err)

	for _, bc := range backupCodes {
		assert.NotZero(t, bc.Id)
	}
}

func TestGetBackupCodeByCode(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Hash:  "hashedpassword",
		Role:  models.READ,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	backupCode := &models.BackupCode{
		UserId: user.Id,
		Code:   "12345678",
		Used:   false,
	}
	err = db.CreateBackupCode(ctx, backupCode)
	require.NoError(t, err)

	found, err := db.GetBackupCodeByCode(ctx, "12345678")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, backupCode.Id, found.Id)
	assert.Equal(t, backupCode.UserId, found.UserId)
	assert.Equal(t, backupCode.Code, found.Code)
	assert.False(t, found.Used)

	found, err = db.GetBackupCodeByCode(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestGetBackupCodesByUserId(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Hash:  "hashedpassword",
		Role:  models.READ,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	backupCodes := []*models.BackupCode{
		{UserId: user.Id, Code: "12345678", Used: false},
		{UserId: user.Id, Code: "87654321", Used: true},
		{UserId: user.Id, Code: "11111111", Used: false},
	}

	err = db.CreateBackupCodes(ctx, backupCodes)
	require.NoError(t, err)

	allCodes, err := db.GetBackupCodesByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, allCodes, 3)

	unusedCodes, err := db.GetUnusedBackupCodesByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, unusedCodes, 2)
}

func TestUseBackupCode(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Hash:  "hashedpassword",
		Role:  models.READ,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	backupCode := &models.BackupCode{
		UserId: user.Id,
		Code:   "12345678",
		Used:   false,
	}
	err = db.CreateBackupCode(ctx, backupCode)
	require.NoError(t, err)

	found, err := db.GetBackupCodeByCode(ctx, "12345678")
	require.NoError(t, err)
	assert.False(t, found.Used)

	err = db.UseBackupCode(ctx, found)
	require.NoError(t, err)

	found, err = db.GetBackupCodeByCode(ctx, "12345678")
	require.NoError(t, err)
	assert.True(t, found.Used)
}

func TestDeleteBackupCodesByUserId(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Hash:  "hashedpassword",
		Role:  models.READ,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	backupCodes := []*models.BackupCode{
		{UserId: user.Id, Code: "12345678", Used: false},
		{UserId: user.Id, Code: "87654321", Used: false},
	}

	err = db.CreateBackupCodes(ctx, backupCodes)
	require.NoError(t, err)

	allCodes, err := db.GetBackupCodesByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, allCodes, 2)

	err = db.DeleteBackupCodesByUserId(ctx, user.Id)
	require.NoError(t, err)

	allCodes, err = db.GetBackupCodesByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, allCodes, 0)
}

func TestCountUnusedBackupCodesByUserId(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Hash:  "hashedpassword",
		Role:  models.READ,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	backupCodes := []*models.BackupCode{
		{UserId: user.Id, Code: "12345678", Used: false},
		{UserId: user.Id, Code: "87654321", Used: true},
		{UserId: user.Id, Code: "11111111", Used: false},
	}

	err = db.CreateBackupCodes(ctx, backupCodes)
	require.NoError(t, err)

	count, err := db.CountUnusedBackupCodesByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
