package controller_service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/cache"
	"backend/internal/middleware/jwtauth"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/routes"
	"backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatRoomDeletionRequiresCreator(t *testing.T) {
	db := setupTestDB()
	repos := repo.NewRepoContainer(db)
	auth := service.NewAuthService(repos, cache.NewTypedCache[service.BlockEntry](time.Minute, time.Minute))
	users := make([]model.User, 3)
	tokens := make([]string, 3)
	for i, name := range []string{"creator", "member", "outsider"} {
		users[i] = model.User{Username: name, Email: name + "@test.com", Password: "secret"}
		require.NoError(t, auth.Register(&users[i]))
		_, token, _, _, err := auth.Login(name, "secret")
		require.NoError(t, err)
		tokens[i] = token
	}
	r := gin.New()
	routes.SetupChatroomRouter(r.Group("/api"), service.NewChatRoomService(repos), jwtauth.NewAuthMiddleware(auth).Auth(), func(c *gin.Context) { c.Next() })
	request := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api"+path, bytes.NewBufferString(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	require.Equal(t, 401, request("POST", "/chatrooms", "", `{"name":"anonymous"}`).Code)
	created := request("POST", "/chatrooms", tokens[0], fmt.Sprintf(`{"name":"owned","CreatorID":%d,"creator_id":%d,"ID":999}`, users[1].ID, users[2].ID))
	require.Equal(t, 201, created.Code, created.Body.String())
	var result struct {
		Data model.ChatRoom `json:"data"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &result))
	path := fmt.Sprintf("/chatrooms/%d", result.Data.ID)
	var stored model.ChatRoom
	require.NoError(t, db.First(&stored, result.Data.ID).Error)
	// Test the externally visible field without assuming its Go representation.
	var dto map[string]any
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &dto))
	require.EqualValues(t, users[0].ID, dto["data"].(map[string]any)["CreatorID"])
	require.NotEqualValues(t, 999, stored.ID)
	require.NotNil(t, stored.CreatorID)
	require.Equal(t, users[0].ID, *stored.CreatorID)
	require.NoError(t, db.Create(&model.UserChatRoom{UserID: users[1].ID, ChatRoomID: stored.ID}).Error)
	for _, token := range []string{tokens[1], tokens[2]} {
		denied := request("DELETE", path, token, fmt.Sprintf(`{"CreatorID":%d}`, users[0].ID))
		require.Equal(t, 403, denied.Code, denied.Body.String())
		require.NoError(t, db.First(&model.ChatRoom{}, stored.ID).Error)
	}
	require.Equal(t, 401, request("DELETE", path, "", "").Code)
	// Creator authority is independent of membership; creation does not auto-join.
	require.Equal(t, 200, request("DELETE", path, tokens[0], "").Code)
	var count int64
	require.NoError(t, db.Model(&model.ChatRoom{}).Where("id = ?", stored.ID).Count(&count).Error)
	require.Zero(t, count)
	require.Equal(t, 404, request("DELETE", path, tokens[0], "").Code)
	for _, id := range []string{"0", "-1", "18446744073709551616", "abc"} {
		require.Equal(t, 400, request("DELETE", "/chatrooms/"+id, tokens[0], "").Code)
	}
	legacy := model.ChatRoom{Name: "unknown-owner"}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&model.UserChatRoom{UserID: users[0].ID, ChatRoomID: legacy.ID}).Error)
	for _, token := range tokens {
		require.Equal(t, 403, request("DELETE", fmt.Sprintf("/chatrooms/%d", legacy.ID), token, "").Code)
	}
	require.NoError(t, db.First(&model.ChatRoom{}, legacy.ID).Error)
	require.NoError(t, auth.ForceLogoutAll(users[0].ID))
	require.Equal(t, 401, request("POST", "/chatrooms", tokens[0], `{"name":"revoked"}`).Code)
}
