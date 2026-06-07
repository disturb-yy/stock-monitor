package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/gin-gonic/gin"
)

const HeaderRequestID = "X-Request-Id"

type requestIDKey struct{}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = newRequestID()
		}

		c.Header(HeaderRequestID, requestID)

		ctx := context.WithValue(c.Request.Context(), requestIDKey{}, requestID)
		ctx = logger.WithContext(ctx, "request_id", requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte("stock-market-monitor"))
	}
	return hex.EncodeToString(b[:])
}

func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		return requestID
	}
	return ""
}
