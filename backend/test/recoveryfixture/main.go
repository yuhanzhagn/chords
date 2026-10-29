// An explicit integration-test helper. It always creates/removes a private temp
// SQLite file, binds loopback port 0, and stops when its parent's stdin closes.
package main

import (
	"backend/internal/app"
	"backend/internal/cache"
	"backend/internal/controller"
	"backend/internal/middleware/jwtauth"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/routes"
	"backend/internal/service"
	"backend/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type probeProducer struct {
	url    string
	client *http.Client
}

func (p probeProducer) PublishContext(ctx context.Context, _ string, _, raw []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	response, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return fmt.Errorf("test broker status %d", response.StatusCode)
	}
	return nil
}
func (probeProducer) Close() error { return nil }
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	dir, err := os.MkdirTemp("", "chords-s2-probe-")
	must(err)
	defer os.RemoveAll(dir)
	// This fixed key belongs only to this short-lived isolated test server.
	must(os.Setenv("JWT_SECRET", "chords-isolated-s2-probe-test-secret-32bytes"))
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "probe.sqlite")+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(50)"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	must(err)
	sqlDB, err := db.DB()
	must(err)
	defer sqlDB.Close()
	must(app.InitDB(db))
	must(db.Create(&model.User{ID: 1, Username: "probe", Email: "probe@example.test", Password: "unused-test-hash"}).Error)
	must(db.Create(&model.UserSession{UserID: 1, SessionID: "probe-session", ExpiresAt: time.Now().Add(time.Hour)}).Error)
	creator := uint(1)
	must(db.Create(&model.ChatRoom{ID: 1, Name: "probe", CreatorID: &creator}).Error)
	must(db.Create(&model.UserChatRoom{UserID: 1, ChatRoomID: 1}).Error)
	token, _, _, err := utils.GenerateJWT("probe", "probe-session")
	must(err)
	repos := repo.NewRepoContainer(db)
	messages := service.NewMessageService(repos)
	auth := service.NewAuthService(repos, cache.NewTypedCache[service.BlockEntry](time.Minute, time.Minute))
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	api := router.Group("/api")
	authFunc := jwtauth.NewAuthMiddleware(auth).Auth()
	routes.SetupMessageRouter(api, messages, authFunc, func(*gin.Context) {})
	api.POST("/auth/ws-authorize", authFunc, controller.WSAuthorize(messages))
	// Test-only controls share the test access credential; never used by production.
	api.POST("/probe/revoke", authFunc, func(c *gin.Context) {
		must(db.Model(&model.UserSession{}).Where("user_id=1").Update("revoked", true).Error)
		c.Status(204)
	})
	api.POST("/probe/remove-member", authFunc, func(c *gin.Context) { must(db.Where("user_id=1").Delete(&model.UserChatRoom{}).Error); c.Status(204) })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	must(err)
	server := &http.Server{Handler: router, ReadHeaderTimeout: time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	relay := service.NewRelay(repos, probeProducer{os.Getenv("CHORDS_PROBE_BROKER"), &http.Client{Timeout: time.Second}})
	relay.Interval = 50 * time.Millisecond
	done := make(chan struct{})
	go func() { defer close(done); relay.Run(ctx) }()
	go server.Serve(listener)
	must(json.NewEncoder(os.Stdout).Encode(map[string]string{"url": "http://" + listener.Addr().String(), "token": token}))
	_, _ = io.Copy(io.Discard, os.Stdin)
	cancel()
	<-done
	shutdown, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	_ = server.Shutdown(shutdown)
}
