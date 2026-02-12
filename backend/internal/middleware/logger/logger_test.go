package logger_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	//    "github.com/sirupsen/logrus"
	"backend/internal/middleware/logger"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestLogrusLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defaultlogger, hook := test.NewNullLogger()
	router := gin.New()
	router.Use(logger.LogrusLogger(defaultlogger))
	router.GET("/ping", func(c *gin.Context) {
		time.Sleep(5 * time.Millisecond)
		c.JSON(200, gin.H{"msg": "pong"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)

	// Assert that one log entry was written
	require.Len(t, hook.Entries, 1)

	entry := hook.LastEntry()
	require.Equal(t, "/ping", entry.Data["path"])
	require.Equal(t, "GET", entry.Data["method"])
	require.Equal(t, 200, entry.Data["status"])
	require.Contains(t, entry.Message, "incoming request")
	require.NotZero(t, entry.Data["latency_ms"])
}
