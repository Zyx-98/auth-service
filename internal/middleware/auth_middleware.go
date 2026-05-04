package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/jwt"
	"github.com/hatuan/auth-service/pkg/response"
)

func AuthMiddleware(jwtMaker *jwt.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token := ""

		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		if token == "" {
			if cookieToken, err := c.Cookie("access_token"); err == nil && cookieToken != "" {
				token = cookieToken
			}
		}

		if token == "" {
			response.Error(c, apperror.Unauthorized("Missing authorization header or cookie"))
			c.Abort()
			return
		}
		claims, err := jwtMaker.VerifyAccessToken(token)
		if err != nil {
			response.Error(c, apperror.Unauthorized("Invalid or expired token"))
			c.Abort()
			return
		}

		// Reject temporary tokens on general auth middleware
		if isTempToken(claims.Permissions) {
			response.Error(c, apperror.Unauthorized("Temporary token cannot access this endpoint"))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("jti", claims.JTI)

		c.Next()
	}
}

func TwoFAMiddleware(jwtMaker *jwt.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token := ""

		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		if token == "" {
			if cookieToken, err := c.Cookie("access_token"); err == nil && cookieToken != "" {
				token = cookieToken
			}
		}

		if token == "" {
			response.Error(c, apperror.Unauthorized("Missing authorization header or cookie"))
			c.Abort()
			return
		}
		claims, err := jwtMaker.VerifyAccessToken(token)
		if err != nil {
			response.Error(c, apperror.Unauthorized("Invalid or expired token"))
			c.Abort()
			return
		}

		if !hasPermission(claims.Permissions, "2fa:verify") {
			response.Error(c, apperror.Unauthorized("Insufficient permissions for 2FA verification"))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("jti", claims.JTI)

		c.Next()
	}
}

func isTempToken(permissions []string) bool {
	return len(permissions) > 0 && permissions[0] == "2fa:verify" && len(permissions) == 1
}

func hasPermission(permissions []string, required string) bool {
	for _, perm := range permissions {
		if perm == required {
			return true
		}
	}
	return false
}

func OptionalAuthMiddleware(jwtMaker *jwt.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token := ""

		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		if token == "" {
			if cookieToken, err := c.Cookie("access_token"); err == nil && cookieToken != "" {
				token = cookieToken
			}
		}

		if token == "" {
			c.Next()
			return
		}
		claims, err := jwtMaker.VerifyAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("jti", claims.JTI)

		c.Next()
	}
}
