package middleware

import (
	"github.com/gin-gonic/gin"
)

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME sniffing attacks
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable browser XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// HTTPS enforcement (1 year, include subdomains)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy
		// - default-src 'self': Only same-origin by default
		// - script-src: 'unsafe-inline' and 'unsafe-eval' needed for QR code generation
		//   (qrcode.js library uses inline script evaluation). Consider CSP nonce in future.
		// - style-src: 'unsafe-inline' for inline styles
		// - img-src: 'self' data: allows both same-origin images and data: URIs
		//   (needed for QR code display as data: URI from canvas)
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")

		// Referrer policy: send full URL to same-origin, origin-only to cross-origin
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Disable access to browser features
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
