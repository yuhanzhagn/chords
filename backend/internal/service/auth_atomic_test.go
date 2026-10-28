package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"backend/internal/cache"
	"backend/internal/model"
	"backend/internal/repo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLoginReplacementAtomic(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}))
	repos := repo.NewRepoContainer(db)
	s := NewAuthService(repos, cache.NewTypedCache[BlockEntry](time.Minute, time.Minute)).(*authService)
	require.NoError(t, s.Register(&model.User{Username: "alice", Email: "alice@test.com", Password: "secret"}))
	_, old, _, _, err := s.Login("alice", "secret")
	require.NoError(t, err)
	issuer := s.issueToken
	s.issueToken = func(string, string) (string, string, time.Time, error) {
		return "", "", time.Time{}, errors.New("signing failed")
	}
	_, _, _, _, err = s.Login("alice", "secret")
	require.Error(t, err)
	_, _, err = s.ValidateJWT(old)
	require.NoError(t, err)
	s.issueToken = issuer
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_session BEFORE INSERT ON user_sessions BEGIN SELECT RAISE(ABORT, 'session insert failure'); END`).Error)
	_, _, _, _, err = s.Login("alice", "secret")
	require.Error(t, err)
	_, _, err = s.ValidateJWT(old)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`DROP TRIGGER fail_session`).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_revoke BEFORE UPDATE ON user_sessions BEGIN SELECT RAISE(ABORT, 'session revoke failure'); END`).Error)
	_, _, _, _, err = s.Login("alice", "secret")
	require.Error(t, err)
	_, _, err = s.ValidateJWT(old)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`DROP TRIGGER fail_revoke`).Error)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, _, _, e := s.Login("alice", "secret"); errs <- e }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	var count int64
	require.NoError(t, db.Model(&model.UserSession{}).Where("revoked = ?", false).Count(&count).Error)
	require.EqualValues(t, 1, count)
	_, _, err = s.ValidateJWT(old)
	require.Error(t, err)
	_, newest, _, _, err := s.Login("alice", "secret")
	require.NoError(t, err)
	require.NoError(t, s.Logout(old))
	_, _, err = s.ValidateJWT(newest)
	require.NoError(t, err, "logout from an old session must not revoke the newer login")
	require.NoError(t, db.Model(&model.UserSession{}).Where("revoked = ?", false).Update("expires_at", time.Now().Add(-time.Minute)).Error)
	var expired model.UserSession
	require.NoError(t, db.Where("revoked = ?", false).First(&expired).Error)
	_, err = repos.UserSession.FindValidSession(expired.SessionID, expired.UserID)
	require.Error(t, err)
}
