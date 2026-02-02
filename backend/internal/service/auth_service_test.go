package service_test

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
	"backend/utils"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err, "failed to connect database")

	// Migrate schema
	err = db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Message{}, &model.ChatRoom{}, &model.UserChatRoom{})
	assert.NoError(t, err, "failed to migrate database")

	return db
}

func setupTestCache(t *testing.T) *cache.TypedCache[service.BlockEntry] {
	c := cache.NewTypedCache[service.BlockEntry](
		10*time.Minute,
		15*time.Minute,
	)
	return c
}

func TestAuthService_Register(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	auth := service.NewAuthService(repos, nil)

	t.Run("nil user", func(t *testing.T) {
		err := auth.Register(nil)
		assert.EqualError(t, err, "user cannot be nil")
	})

	t.Run("empty username", func(t *testing.T) {
		u := &model.User{Username: "", Email: "test@example.com"}
		err := auth.Register(u)
		assert.EqualError(t, err, "user name is required")
	})

	t.Run("successful registration", func(t *testing.T) {
		u := &model.User{
			Username:  "alice",
			Email:     "alice@example.com",
			Password:  "secret",
			CreatedAt: time.Now(),
		}

		err := auth.Register(u)
		assert.NoError(t, err)

		// Verify user exists in DB
		var fetched model.User
		err = db.First(&fetched, "username = ?", "alice").Error
		assert.NoError(t, err)
		assert.Equal(t, "alice@example.com", fetched.Email)
	})
}

func TestLogin(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	auth := service.NewAuthService(repos, nil)

	password := "password123"
	hashed, _ := utils.HashPassword(password)
	user := &model.User{
		Username: "loginuser",
		Email:    "login@example.com",
		Password: hashed,
	}
	require.NoError(t, repos.User.Create(user))

	// Case 1: invalid username
	_, _, _, _, err := auth.Login("wronguser", password)
	require.Error(t, err)
	require.Equal(t, "invalid username or password", err.Error())

	// Case 2: invalid password
	_, _, _, _, err = auth.Login("loginuser", "wrongpassword")
	require.Error(t, err)
	require.Equal(t, "invalid username or password", err.Error())

	// Case 3: successful login
	id, token, jti, sessionID, err := auth.Login("loginuser", password)
	require.NoError(t, err)
	require.Equal(t, user.ID, id)
	require.NotEmpty(t, token)
	require.NotEmpty(t, jti)
	require.NotEmpty(t, sessionID)

	// Verify session in DB
	var session model.UserSession
	err = db.Where("session_id = ?", sessionID).First(&session).Error
	require.NoError(t, err)
	require.Equal(t, user.ID, session.UserID)
}

func TestValidateJWT(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	s := service.NewAuthService(repos, nil)

	user := model.User{
		Username: "tester",
		Email:    "test@test.com",
		Password: "pass",
	}
	require.NoError(t, repos.User.Create(&user))

	session := model.UserSession{
		UserID:     user.ID,
		SessionID:  "session123",
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		Revoked:    false,
	}
	require.NoError(t, repos.UserSession.Create(&session))

	// Generate valid JWT
	token, _, _, err := utils.GenerateJWT(user.Username, session.SessionID)
	require.NoError(t, err)

	// -------------------------------
	// Case 1: Valid token
	// -------------------------------
	gotUser, gotSession, err := s.ValidateJWT(token)
	require.NoError(t, err)
	require.Equal(t, user.ID, gotUser.ID)
	require.Equal(t, session.SessionID, gotSession.SessionID)

	// -------------------------------
	// Case 2: Invalid token string
	// -------------------------------
	_, _, err = s.ValidateJWT("BADTOKEN.INVALID.DATA")
	require.Error(t, err)
	require.Equal(t, "invalid token", err.Error())

	// -------------------------------
	// Case 3: Username not found
	// -------------------------------
	tokenBadUser, _, _, err := utils.GenerateJWT("does_not_exist", session.SessionID)
	require.NoError(t, err)

	_, _, err = s.ValidateJWT(tokenBadUser)
	require.Error(t, err)
	require.Equal(t, "username is invalid", err.Error())

	// Case 4: Session revoked
	require.NoError(t, repos.UserSession.RevokeOne(user.ID, session.SessionID))

	tokenRevoked, _, _, err := utils.GenerateJWT(user.Username, session.SessionID)
	require.NoError(t, err)

	_, _, err = s.ValidateJWT(tokenRevoked)
	require.Error(t, err)
	require.Equal(t, "session invalid or revoked", err.Error())

	// Case 5: User deleted — new DB, session repo only (no user for "tester")
	db2 := setupTestDB(t)
	repos2 := repo.NewRepoContainer(db2)
	s2 := service.NewAuthService(repos2, nil)
	require.NoError(t, repos2.UserSession.Create(&session))

	tokenMissingUser, _, _, err := utils.GenerateJWT("tester", session.SessionID)
	require.NoError(t, err)

	_, _, err = s2.ValidateJWT(tokenMissingUser)
	require.Error(t, err)
	require.Equal(t, "username is invalid", err.Error())
}

func TestForceLogoutAll(t *testing.T) {
	db := setupTestDB(t)
	repos := repo.NewRepoContainer(db)
	auth := service.NewAuthService(repos, nil)

	password := "password123"
	hashed, _ := utils.HashPassword(password)
	user := &model.User{
		Username: "loginuser",
		Email:    "login@example.com",
		Password: hashed,
	}
	require.NoError(t, repos.User.Create(user))

	userID := user.ID
	sessionID := "12345"
	session := &model.UserSession{
		UserID:     userID,
		SessionID:  sessionID,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repos.UserSession.Create(session))

	auth.ForceLogoutAll(userID)
	var sess model.UserSession
	err := db.Where("session_id = ? AND user_id = ? AND revoked = ?", sessionID, userID, true).First(&sess).Error
	require.NoError(t, err)
}
