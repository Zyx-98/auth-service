package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		latency := time.Since(startTime)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		if statusCode >= 400 {
			logger.Warn(
				"HTTP request",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.String("client_ip", clientIP),
				zap.String("user_agent", userAgent),
				zap.Duration("latency", latency),
				zap.String("error", c.Errors.ByType(gin.ErrorTypeBind).String()),
			)
		} else {
			logger.Debug(
				"HTTP request",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.String("client_ip", clientIP),
				zap.Duration("latency", latency),
			)
		}
	}
}
