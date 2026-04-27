package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
	"github.com/hatuan/auth-service/pkg/validator"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type OAuthHandler struct {
	oauthService  *service.OAuthService
	redisClient   *redis.Client
	logger        *zap.Logger
}

func NewOAuthHandler(oauthService *service.OAuthService, redisClient *redis.Client, logger *zap.Logger) *OAuthHandler {
	return &OAuthHandler{
		oauthService:  oauthService,
		redisClient:   redisClient,
		logger:        logger,
	}
}

func (h *OAuthHandler) GoogleLoginRedirect(c *gin.Context) {
	state := uuid.New().String()

	if err := h.redisClient.Set(c.Request.Context(), "oauth_state:"+state, "true", 10*60*time.Second).Err(); err != nil {
		h.logger.Error("Failed to store OAuth state in Redis", zap.Error(err), zap.String("state", state))
		response.Error(c, apperror.InternalServerError("Failed to store state", err))
		return
	}

	h.logger.Debug("OAuth state stored", zap.String("state", state))
	authURL := h.oauthService.GetGoogleAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	h.logger.Debug("OAuth callback received", zap.String("state", state), zap.String("code", code[:min(10, len(code))]))

	if code == "" || state == "" {
		h.logger.Warn("Missing OAuth parameters", zap.String("code", code), zap.String("state", state))
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Missing code or state parameter",
		})
		return
	}

	exists, err := h.redisClient.Exists(c.Request.Context(), "oauth_state:"+state).Result()
	if err != nil {
		h.logger.Error("Redis error checking state", zap.Error(err), zap.String("state", state))
	}
	if err != nil || exists == 0 {
		h.logger.Warn("State validation failed", zap.Error(err), zap.String("state", state), zap.Int64("exists", exists))
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{
			"error": "Invalid or expired state",
		})
		return
	}

	h.logger.Debug("State validated", zap.String("state", state))

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
		"totp_required": callbackResp.TOTPRequired,
		"totp_token":    callbackResp.TOTPToken,
	})
}

func (h *OAuthHandler) VerifyOAuthTOTP(c *gin.Context) {
	var req dto.TOTPVerifyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	oauthResp, err := h.oauthService.VerifyOAuthTOTP(c.Request.Context(), req.TOTPToken, req.Code)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, oauthResp)
}
