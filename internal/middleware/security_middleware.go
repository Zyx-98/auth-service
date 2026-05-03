package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME sniffing attacks
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking (also covered by CSP frame-ancestors)
		c.Header("X-Frame-Options", "DENY")

		// Browser XSS filter (legacy, modern browsers use CSP)
		c.Header("X-XSS-Protection", "1; mode=block")

		// HTTPS enforcement (1 year, include subdomains)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Content Security Policy with nonce support
		nonce, ok := c.Get(NonceContextKey)
		nonceStr := ""
		if ok {
			nonceStr = fmt.Sprintf("'nonce-%s'", nonce)
		}

		csp := fmt.Sprintf(
			"default-src 'self'; "+
				"script-src 'self' %s; "+
				"style-src 'self' %s; "+
				"img-src 'self' data:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"form-action 'self'; "+
				"base-uri 'self'; "+
				"upgrade-insecure-requests",
			nonceStr, nonceStr,
		)
		c.Header("Content-Security-Policy", csp)

		// Referrer policy: send full URL to same-origin, origin-only to cross-origin
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Disable access to browser features
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Prevent Flash/PDF policy abuse
		c.Header("X-Permitted-Cross-Domain-Policies", "none")

		// Cross-Origin-Embedder-Policy isolates cross-origin resources
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")

		// Prevent cross-site embedding
		c.Header("Cross-Origin-Resource-Policy", "same-site")

		// Prevent DNS prefetching (slight privacy benefit)
		c.Header("X-DNS-Prefetch-Control", "off")

		c.Next()
	}
}
