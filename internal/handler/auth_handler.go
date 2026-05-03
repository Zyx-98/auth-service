package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/internal/middleware"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
	"github.com/hatuan/auth-service/pkg/validator"
)

type AuthHandler struct {
	authService *service.AuthService
	totpService *service.TOTPService
}

func NewAuthHandler(authService *service.AuthService, totpService *service.TOTPService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		totpService: totpService,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	tokenResp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	if tokenResp != nil {
		middleware.SetSecureCookie(c, "access_token", tokenResp.AccessToken, 15*60)
		middleware.SetSecureCookie(c, "refresh_token", tokenResp.RefreshToken, 7*24*60*60)
	}
	response.Created(c, tokenResp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	if req.DeviceToken == "" {
		if deviceToken, err := c.Cookie("device_token"); err == nil && deviceToken != "" {
			req.DeviceToken = deviceToken
		}
	}

	userAgent := c.Request.UserAgent()
	ip := c.ClientIP()

	loginResp, err := h.authService.Login(c.Request.Context(), &req, userAgent, ip)
	if err != nil {
		response.Error(c, err)
		return
	}

	if loginResp != nil && loginResp.Token != nil {
		middleware.SetSecureCookie(c, "access_token", loginResp.Token.AccessToken, 15*60)
		middleware.SetSecureCookie(c, "refresh_token", loginResp.Token.RefreshToken, 7*24*60*60)
		if loginResp.DeviceToken != "" {
			middleware.SetSecureCookie(c, "device_token", loginResp.DeviceToken, 30*24*60*60)
		}
	}
	response.Ok(c, loginResp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	tokenResp, err := h.authService.Refresh(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	if tokenResp != nil {
		middleware.SetSecureCookie(c, "access_token", tokenResp.AccessToken, 15*60)
		middleware.SetSecureCookie(c, "refresh_token", tokenResp.RefreshToken, 7*24*60*60)
	}
	response.Ok(c, tokenResp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	jti, exists := c.Get("jti")

	if exists {
		if err := h.authService.Logout(c.Request.Context(), jti.(string)); err != nil {
			response.Error(c, apperror.InternalServerError("Failed to logout", err))
			return
		}
	}

	middleware.ClearSecureCookie(c, "access_token")
	middleware.ClearSecureCookie(c, "refresh_token")
	middleware.ClearSecureCookie(c, "device_token")
	response.NoContent(c)
}

func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperror.Unauthorized("Missing user ID"))
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Error(c, apperror.InternalServerError("Invalid user ID type", nil))
		return
	}

	if err := h.authService.LogoutAllDevices(c.Request.Context(), userID); err != nil {
		response.Error(c, apperror.InternalServerError("Failed to logout from all devices", err))
		return
	}

	middleware.ClearSecureCookie(c, "access_token")
	middleware.ClearSecureCookie(c, "refresh_token")
	middleware.ClearSecureCookie(c, "device_token")
	response.NoContent(c)
}

func (h *AuthHandler) Introspect(c *gin.Context) {
	var req dto.IntrospectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	introspectResp := h.authService.Introspect(c.Request.Context(), &req)
	response.Ok(c, introspectResp)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperror.Unauthorized("Missing user ID"))
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Error(c, apperror.InternalServerError("Invalid user ID type", nil))
		return
	}

	profile, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, profile)
}

func (h *AuthHandler) VerifyTwoFA(c *gin.Context) {
	var req dto.TOTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperror.Unauthorized("Missing user ID"))
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Error(c, apperror.InternalServerError("Invalid user ID type", nil))
		return
	}

	valid, err := h.totpService.VerifyLogin(c.Request.Context(), userID, req.Code)
	if err != nil {
		response.Error(c, err)
		return
	}

	if !valid {
		response.Error(c, apperror.Unauthorized("Invalid 2FA code"))
		return
	}

	userAgent := c.Request.UserAgent()
	ip := c.ClientIP()

	loginResp, err := h.authService.IssueTempTokensWithTrust(c.Request.Context(), userID, userAgent, ip, req.TrustDevice)
	if err != nil {
		response.Error(c, err)
		return
	}

	if loginResp != nil && loginResp.Token != nil {
		middleware.SetSecureCookie(c, "access_token", loginResp.Token.AccessToken, 15*60)
		middleware.SetSecureCookie(c, "refresh_token", loginResp.Token.RefreshToken, 7*24*60*60)
		if loginResp.DeviceToken != "" {
			middleware.SetSecureCookie(c, "device_token", loginResp.DeviceToken, 30*24*60*60)
		}
	}
	response.Ok(c, loginResp)
}

func (h *AuthHandler) GetTrustedDevices(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperror.Unauthorized("Missing user ID"))
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Error(c, apperror.InternalServerError("Invalid user ID type", nil))
		return
	}

	devices, err := h.authService.GetTrustedDevices(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, apperror.InternalServerError("Failed to fetch trusted devices", err))
		return
	}

	response.Ok(c, devices)
}

func (h *AuthHandler) DeleteTrustedDevices(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperror.Unauthorized("Missing user ID"))
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Error(c, apperror.InternalServerError("Invalid user ID type", nil))
		return
	}

	if err := h.authService.RevokeTrustedDevices(c.Request.Context(), userID); err != nil {
		response.Error(c, apperror.InternalServerError("Failed to revoke trusted devices", err))
		return
	}

	response.NoContent(c)
}
