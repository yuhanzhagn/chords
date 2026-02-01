package logger

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

func LogrusLogger(log *logrus.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()

        log.WithFields(logrus.Fields{
            "status":   c.Writer.Status(),
            "method":   c.Request.Method,
            "path":     c.Request.URL.Path,
            "latency_ms":  time.Since(start).Nanoseconds(),
            "clientIP": c.ClientIP(),
        }).Info("incoming request")
    }
}

