package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/Zyx-98/auth-service/internal/dto"
	"github.com/Zyx-98/auth-service/internal/middleware"
	"github.com/Zyx-98/auth-service/internal/service"
	"github.com/Zyx-98/auth-service/pkg/apperror"
	"github.com/Zyx-98/auth-service/pkg/response"
	"github.com/Zyx-98/auth-service/pkg/validator"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type OAuthState struct {
	DeviceToken string `json:"device_token"`
}

type OAuthHandler struct {
	oauthService *service.OAuthService
	redisClient  *redis.Client
	logger       *zap.Logger
}

func NewOAuthHandler(oauthService *service.OAuthService, redisClient *redis.Client, logger *zap.Logger) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		redisClient:  redisClient,
		logger:       logger,
	}
}

func (h *OAuthHandler) GoogleLoginRedirect(c *gin.Context) {
	var req dto.GoogleLoginRedirectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	state := uuid.New().String()

	oauthState := OAuthState{DeviceToken: req.DeviceToken}
	if oauthState.DeviceToken == "" {
		if cookie, err := c.Cookie("device_token"); err == nil && cookie != "" {
			oauthState.DeviceToken = cookie
		}
	}

	stateJSON, err := json.Marshal(oauthState)
	if err != nil {
		response.Error(c, apperror.InternalServerError("Failed to create state", err))
		return
	}

	if err := h.redisClient.Set(c.Request.Context(), "oauth_state:"+state, stateJSON, 10*60*time.Second).Err(); err != nil {
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

	if req.Code == "" || req.State == "" {
		h.logger.Warn("Missing OAuth parameters")
		response.Error(c, apperror.BadRequest("Missing code or state parameter", nil))
		return
	}

	exists, err := h.redisClient.Exists(c.Request.Context(), "oauth_state:"+req.State).Result()
	if err != nil {
		h.logger.Error("Redis error checking state", zap.Error(err), zap.String("state", req.State))
	}
	if err != nil || exists == 0 {
		h.logger.Warn("State validation failed", zap.Error(err), zap.String("state", req.State), zap.Int64("exists", exists))
		response.Error(c, apperror.Unauthorized("Invalid or expired state"))
		return
	}

	h.logger.Debug("State validated", zap.String("state", req.State))

	stateJSON, err := h.redisClient.GetDel(c.Request.Context(), "oauth_state:"+req.State).Result()
	if err != nil {
		h.logger.Error("Failed to retrieve state data", zap.Error(err), zap.String("state", req.State))
		response.Error(c, apperror.InternalServerError("Failed to retrieve state", err))
		return
	}

	var oauthState OAuthState
	if err := json.Unmarshal([]byte(stateJSON), &oauthState); err != nil {
		h.logger.Error("Failed to unmarshal OAuth state", zap.Error(err), zap.String("state", req.State))
		response.Error(c, apperror.InternalServerError("Failed to retrieve state", err))
		return
	}

	deviceToken := oauthState.DeviceToken
	if deviceToken == "" {
		if cookie, err := c.Cookie("device_token"); err == nil && cookie != "" {
			deviceToken = cookie
		}
	}

	callbackResp, err := h.oauthService.HandleGoogleCallback(c.Request.Context(), req.Code, deviceToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}

	if !callbackResp.TOTPRequired {
		middleware.SetSecureCookie(c, "access_token", callbackResp.AccessToken, 15*60)
		middleware.SetSecureCookie(c, "refresh_token", callbackResp.RefreshToken, 7*24*60*60)
		if callbackResp.DeviceToken != "" {
			middleware.SetSecureCookie(c, "device_token", callbackResp.DeviceToken, 30*24*60*60)
		}
	}

	response.Ok(c, callbackResp)
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

	userAgent := c.Request.UserAgent()
	ip := c.ClientIP()

	oauthResp, err := h.oauthService.VerifyOAuthTOTP(c.Request.Context(), req.TOTPToken, req.Code, userAgent, ip, req.TrustDevice)
	if err != nil {
		response.Error(c, err)
		return
	}

	if oauthResp != nil {
		middleware.SetSecureCookie(c, "access_token", oauthResp.AccessToken, 15*60)
		middleware.SetSecureCookie(c, "refresh_token", oauthResp.RefreshToken, 7*24*60*60)
		if oauthResp.DeviceToken != "" {
			middleware.SetSecureCookie(c, "device_token", oauthResp.DeviceToken, 30*24*60*60)
		}
	}
	response.Ok(c, oauthResp)
}
