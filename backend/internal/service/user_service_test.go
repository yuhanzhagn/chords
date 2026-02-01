package service_test

import (
//    "errors"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
//    "gorm.io/gorm"

    "backend/internal/model"
    "backend/internal/repo"
    "backend/internal/service"
)


func TestUserService_CreateUser(t *testing.T) {
    db := setupTestDB(t)
    repos := repo.NewRepoContainer(db)
    userSvc := service.NewUserService(repos)

    t.Run("nil user", func(t *testing.T) {
        err := userSvc.CreateUser(nil)
        require.Error(t, err)
        require.Equal(t, "user cannot be nil", err.Error())
    })

    t.Run("empty username", func(t *testing.T) {
        user := &model.User{Username: "", Email: "test@example.com"}
        err := userSvc.CreateUser(user)
        require.Error(t, err)
        require.Equal(t, "user name is required", err.Error())
    })

    t.Run("successful creation", func(t *testing.T) {
        user := &model.User{
            Username:  "alice",
            Email:     "alice@example.com",
            Password:  "secret",
            CreatedAt: time.Now(),
        }
        err := userSvc.CreateUser(user)
        require.NoError(t, err)
        require.NotZero(t, user.ID)
    })
}

func TestUserService_GetAllUsers(t *testing.T) {
    db := setupTestDB(t)
    repos := repo.NewRepoContainer(db)
    userSvc := service.NewUserService(repos)

    t.Run("get all users", func(t *testing.T) {
        users, err := userSvc.GetAllUsers()
        require.NoError(t, err)
        require.GreaterOrEqual(t, len(users), 0)
    })
}

func TestUserService_GetUserByUsername(t *testing.T) {
    db := setupTestDB(t)
    repos := repo.NewRepoContainer(db)
    userSvc := service.NewUserService(repos)

    t.Run("empty username", func(t *testing.T) {
        user, err := userSvc.GetUserByUsername("")
        require.Error(t, err)
        require.Nil(t, user)
        require.Equal(t, "username is required", err.Error())
    })

    t.Run("nonexistent user", func(t *testing.T) {
        user, err := userSvc.GetUserByUsername("nonexistent")
        require.NoError(t, err)
        require.Nil(t, user)
    })

    t.Run("existing user", func(t *testing.T) {
        // TODO: create test user in DB
        testUser := &model.User{
            Username:  "bob",
            Email:     "bob@example.com",
            Password:  "secret",
            CreatedAt: time.Now(),
        }
        _ = userSvc.CreateUser(testUser)

        user, err := userSvc.GetUserByUsername("bob")
        require.NoError(t, err)
        require.NotNil(t, user)
        require.Equal(t, "bob", user.Username)
    })
}

