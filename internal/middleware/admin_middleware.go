package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
)

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("roles")
		if !exists {
			response.Error(c, apperror.Unauthorized("Missing roles"))
			c.Abort()
			return
		}

		roles, ok := rolesVal.([]string)
		if !ok {
			response.Error(c, apperror.InternalServerError("Invalid roles type", nil))
			c.Abort()
			return
		}

		isAdmin := false
		for _, role := range roles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			response.Error(c, apperror.Forbidden("Admin access required"))
			c.Abort()
			return
		}

		c.Next()
	}
}
