package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const traceIDHeader = "X-Request-ID"

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		traceID := c.Request.Header.Get(traceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}
		c.Set("trace_id", traceID)

		c.Writer.Header().Set(traceIDHeader, traceID)

		c.Next()

		logger.Info("http_request",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

func generateTraceID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}

