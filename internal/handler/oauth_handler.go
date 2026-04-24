package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
	"github.com/hatuan/auth-service/pkg/validator"
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
	var req dto.GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	exists, err := h.redisClient.Exists(c.Request.Context(), "oauth_state:"+req.State).Result()
	if err != nil || exists == 0 {
		response.Error(c, apperror.Unauthorized("Invalid or expired state"))
		return
	}

	h.redisClient.Del(c.Request.Context(), "oauth_state:"+req.State)

	callbackResp, err := h.oauthService.HandleGoogleCallback(c.Request.Context(), req.Code)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, callbackResp)
}
