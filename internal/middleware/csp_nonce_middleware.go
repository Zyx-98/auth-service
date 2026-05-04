package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const NonceContextKey = "csp_nonce"

func CSPNonceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := generateNonce()
		c.Set(NonceContextKey, nonce)
		c.Header("X-CSP-Nonce", nonce)
		c.Next()
	}
}

func generateNonce() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}
