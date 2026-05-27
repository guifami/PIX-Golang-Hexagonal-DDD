package httpadapter

import (
	"time"

	"go-api/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.NewString()

		log := logger.Log.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)

		c.Set("logger", log)
		c.Request = c.Request.WithContext(logger.WithContext(c.Request.Context(), log))

		c.Next()

		log.Info("request completed",
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

func LoggerFromContext(c *gin.Context) *zap.Logger {
	if l, exists := c.Get("logger"); exists {
		if log, ok := l.(*zap.Logger); ok {
			return log
		}
	}
	return logger.Log
}
