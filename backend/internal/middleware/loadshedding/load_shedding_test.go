package loadshedding_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"backend/internal/middleware/loadshedding"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Simulated slow handler
func slowHandler(c *gin.Context) {
	time.Sleep(200 * time.Millisecond)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func TestLoadShedding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Middleware with small limits for testing
	maxConcurrent := 2
	maxQueue := 2
	queueWait := 100 * time.Millisecond
	r.Use(loadshedding.LoadShedding(maxConcurrent, maxQueue, queueWait))

	r.GET("/test", slowHandler)

	// Run multiple requests concurrently
	totalRequests := 10
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	results := make([]int, totalRequests)
	for i := 0; i < totalRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)

			w := &mockResponseWriter{}
			r.ServeHTTP(w, req)
			results[idx] = w.status
		}(i)
	}

	wg.Wait()

	// Count 200 OK vs 503 overload
	okCount := 0
	overloadCount := 0
	for _, status := range results {
		if status == 200 {
			okCount++
		} else if status == 503 {
			overloadCount++
		}
	}

	t.Logf("200 OK: %d, 503 Overload: %d", okCount, overloadCount)

	// Require at least some requests succeeded and some failed
	require.Greater(t, okCount, 0, "At least some requests should succeed")
	require.Greater(t, overloadCount, 0, "Some requests should be rejected due to load")
}

// Minimal ResponseWriter mock
type mockResponseWriter struct {
	header http.Header
	status int
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}
