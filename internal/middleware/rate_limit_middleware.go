package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimitMiddleware(redisClient *redisclient.Client, rate string) gin.HandlerFunc {
	rate_parsed, err := limiter.NewRateFromFormatted(rate)
	if err != nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	store, err := redisstore.NewStore(redisClient)
	if err != nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	instance := limiter.New(store, rate_parsed)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		limit, err := instance.Get(c, clientIP)
		if err != nil {
			response.Error(c, apperror.InternalServerError("Rate limit check failed", err))
			c.Abort()
			return
		}

		if limit.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    "rate_limit_exceeded",
				"message": "Too many requests, please try again later",
				"retry_after": limit.Reset,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func RateLimitByUserMiddleware(redisClient *redisclient.Client, rate string) gin.HandlerFunc {
	rate_parsed, err := limiter.NewRateFromFormatted(rate)
	if err != nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	store, err := redisstore.NewStore(redisClient)
	if err != nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	instance := limiter.New(store, rate_parsed)

	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			c.Next()
			return
		}

		limit, err := instance.Get(c, userID.String())
		if err != nil {
			response.Error(c, apperror.InternalServerError("Rate limit check failed", err))
			c.Abort()
			return
		}

		if limit.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    "rate_limit_exceeded",
				"message": "Too many requests, please try again later",
				"retry_after": limit.Reset,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
