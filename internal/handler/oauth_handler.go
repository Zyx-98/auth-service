package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
	"github.com/redis/go-redis/v9"
)

type OAuthHandler struct {
	oauthService  *service.OAuthService
	redisClient   *redis.Client
}

func NewOAuthHandler(oauthService *service.OAuthService, redisClient *redis.Client) *OAuthHandler {
	return &OAuthHandler{
		oauthService:  oauthService,
		redisClient:   redisClient,
	}
}

func (h *OAuthHandler) GoogleLoginRedirect(c *gin.Context) {
	state := uuid.New().String()

	if err := h.redisClient.Set(c.Request.Context(), "oauth_state:"+state, "true", 10*60).Err(); err != nil {
		response.Error(c, apperror.InternalServerError("Failed to store state", err))
		return
	}

	authURL := h.oauthService.GetGoogleAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Missing code or state parameter",
		})
		return
	}

	exists, err := h.redisClient.Exists(c.Request.Context(), "oauth_state:"+state).Result()
	if err != nil || exists == 0 {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{
			"error": "Invalid or expired state",
		})
		return
	}

	h.redisClient.Del(c.Request.Context(), "oauth_state:"+state)

	callbackResp, err := h.oauthService.HandleGoogleCallback(c.Request.Context(), code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "oauth_callback.html", gin.H{
		"access_token":  callbackResp.AccessToken,
		"refresh_token": callbackResp.RefreshToken,
		"expires_in":    callbackResp.ExpiresIn,
		"token_type":    callbackResp.TokenType,
		"is_new_user":   callbackResp.IsNewUser,
	})
}
