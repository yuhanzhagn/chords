package controller_service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/cache"
	"backend/internal/controller"
	"backend/internal/identity"
	"backend/internal/middleware/jwtauth"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/routes"
	"backend/internal/service"
	kafkapb "backend/proto/kafka"
	"backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type recordingProducer struct{ events []*kafkapb.KafkaEvent }

func (p *recordingProducer) Publish(_ string, _ []byte, raw []byte) error {
	e := &kafkapb.KafkaEvent{}
	if err := proto.Unmarshal(raw, e); err != nil {
		return err
	}
	p.events = append(p.events, e)
	return nil
}
func (p *recordingProducer) Close() error { return nil }

func TestAuthenticatedResourceBoundaries(t *testing.T) {
	db := setupTestDB()
	repos := repo.NewRepoContainer(db)
	auth := service.NewAuthService(repos, cache.NewTypedCache[service.BlockEntry](time.Minute, time.Minute))
	alice := &model.User{Username: "alice", Email: "a@test.com", Password: "secret"}
	bob := &model.User{Username: "bob", Email: "b@test.com", Password: "secret"}
	require.NoError(t, auth.Register(alice))
	require.NoError(t, auth.Register(bob))
	_, token, _, sid, err := auth.Login("alice", "secret")
	require.NoError(t, err)
	p := identity.New(alice.ID, alice.Username, sid)
	room := model.ChatRoom{Name: "General"}
	require.NoError(t, db.Create(&room).Error)
	bobMsg := model.Message{UserID: bob.ID, RoomID: room.ID, Content: "Bob's message"}
	require.NoError(t, db.Create(&bobMsg).Error)
	ms := service.NewMessageService(repos)
	membership := service.NewMembershipService(repos, cache.NewTypedCache[[]model.ChatRoom](time.Minute, time.Minute))
	r := gin.New()
	api := r.Group("/api")
	mw := jwtauth.NewAuthMiddleware(auth).Auth()
	noop := func(c *gin.Context) { c.Next() }
	routes.SetupMessageRouter(api, ms, mw, noop)
	routes.SetupMembershipRouter(api, membership, mw, noop)
	routes.SetupChatroomRouter(api, service.NewChatRoomService(repos), mw, noop)
	api.POST("/auth/ws-authorize", mw, controller.WSAuthorize(ms))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api"+path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	require.Equal(t, 200, request("GET", "/chatrooms/search?q=General", "").Code)
	for _, query := range []string{"", "?limit=10", "?before_id=99"} {
		require.Equal(t, 403, request("GET", fmt.Sprintf("/chatrooms/%d/messages%s", room.ID, query), "").Code)
	}
	require.Equal(t, 403, request("POST", "/auth/ws-authorize", fmt.Sprintf(`{"room_id":%d,"kind":"join"}`, room.ID)).Code)
	require.Equal(t, 403, request("POST", "/memberships/add-user", fmt.Sprintf(`{"username":"bob","chatroomid":%d}`, room.ID)).Code)
	require.Equal(t, 403, request("GET", "/memberships/bob/chatrooms", "").Code)
	require.Equal(t, 200, request("POST", "/memberships/add-user", fmt.Sprintf(`{"username":"alice","chatroomid":%d}`, room.ID)).Code)
	require.Equal(t, 200, request("GET", fmt.Sprintf("/chatrooms/%d/messages", room.ID), "").Code)
	require.Equal(t, 403, request("POST", "/messages", fmt.Sprintf(`{"user_id":%d,"chat_room_id":%d,"content":"forged"}`, bob.ID, room.ID)).Code)
	require.Equal(t, 403, request("DELETE", fmt.Sprintf("/messages/%d", bobMsg.ID), "").Code)
	own := request("POST", "/messages", fmt.Sprintf(`{"user_id":%d,"chat_room_id":%d,"content":"own"}`, alice.ID, room.ID))
	require.Equal(t, 201, own.Code)
	var ownMsg model.Message
	require.NoError(t, json.Unmarshal(own.Body.Bytes(), &ownMsg))
	require.Equal(t, alice.ID, ownMsg.UserID)
	require.Equal(t, 204, request("DELETE", fmt.Sprintf("/messages/%d", ownMsg.ID), "").Code)
	event := &kafkapb.KafkaEvent{UserId: uint32(alice.ID), RoomId: uint32(room.ID), MsgType: "message", Content: []byte("from WS")}
	raw, err := proto.Marshal(event)
	require.NoError(t, err)
	grantResponse := request("POST", "/auth/ws-authorize", fmt.Sprintf(`{"room_id":%d,"kind":"message","digest":"%s"}`, room.ID, utils.EventDigest(raw)))
	require.Equal(t, 200, grantResponse.Code)
	var grant service.WSGrant
	require.NoError(t, json.Unmarshal(grantResponse.Body.Bytes(), &grant))
	require.NotEmpty(t, grant.CommandToken)
	_, _, err = auth.ValidateJWT(grant.CommandToken)
	require.Error(t, err, "command credentials must not authenticate as access tokens")
	producer := &recordingProducer{}
	ks := service.KafkaService{Producer: producer, MessageService: ms}
	ks.HandleOutboundEvent(event, "")
	require.Empty(t, producer.events)
	ks.HandleOutboundEvent(event, token)
	require.Empty(t, producer.events)
	tampered := proto.Clone(event).(*kafkapb.KafkaEvent)
	tampered.Content = []byte("tampered")
	ks.HandleOutboundEvent(tampered, grant.CommandToken)
	require.Empty(t, producer.events)
	ks.HandleOutboundEvent(proto.Clone(event).(*kafkapb.KafkaEvent), grant.CommandToken)
	require.Len(t, producer.events, 1)
	require.Equal(t, alice.ID, uint(producer.events[0].UserId))
	var count int64
	require.NoError(t, db.Model(&model.Message{}).Where("content = ?", "from WS").Count(&count).Error)
	require.EqualValues(t, 1, count)
	// A grant issued before membership removal is re-authorized at consumption.
	require.NoError(t, db.Where("user_id = ? AND chat_room_id = ?", alice.ID, room.ID).Delete(&model.UserChatRoom{}).Error)
	ks.HandleOutboundEvent(event, grant.CommandToken)
	require.Len(t, producer.events, 1)
	require.NoError(t, db.Create(&model.UserChatRoom{UserID: alice.ID, ChatRoomID: room.ID}).Error)
	require.NoError(t, auth.ForceLogoutAll(alice.ID))
	_, err = ms.CreateMessage(p, alice.ID, room.ID, "stale principal")
	require.ErrorIs(t, err, service.ErrUnauthenticated)
	ks.HandleOutboundEvent(event, grant.CommandToken)
	require.Len(t, producer.events, 1)
	require.Equal(t, 401, request("GET", fmt.Sprintf("/chatrooms/%d/messages", room.ID), "").Code)
	require.NoError(t, db.Model(&model.Message{}).Where("content = ?", "from WS").Count(&count).Error)
	require.EqualValues(t, 1, count)
	// Infrastructure errors must not masquerade as invalid credentials.
	require.NoError(t, db.Migrator().DropTable(&model.UserSession{}))
	require.Equal(t, 503, request("POST", "/auth/ws-authorize", `{}`).Code)

}
