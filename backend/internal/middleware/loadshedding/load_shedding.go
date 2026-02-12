package loadshedding

import (
	//"net/http"
	"github.com/gin-gonic/gin"
	"time"
)

func LoadShedding(maxConcurrent, maxQueue int, queueWait time.Duration) gin.HandlerFunc {
	sem := make(chan struct{}, maxConcurrent)
	queue := make(chan struct{}, maxQueue)

	return func(c *gin.Context) {
		// Try to enter the queue
		select {
		case queue <- struct{}{}:
			// ok
		default:
			//Add fallback here
			FallbackHandler(c)
			// c.JSON(503, gin.H{"error": "server overloaded"})
			// c.Abort()
			return
		}
		defer func() { <-queue }()

		// Try to acquire worker slot
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		case <-time.After(queueWait):
			c.JSON(503, gin.H{"error": "server busy, timeout"})
			c.Abort()
			return
		}
	}
}

func FallbackHandler(c *gin.Context) {
	// You can provide cached/default data here
	response := gin.H{
		"message": "server busy, try again later",
		"fallback": gin.H{
			"user":            "guest",
			"recommendations": []string{"item1", "item2"},
		},
	}

	c.JSON(200, response) // or 503 if you prefer
	c.Abort()
}
