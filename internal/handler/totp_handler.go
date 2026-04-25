package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
	"github.com/hatuan/auth-service/pkg/validator"
)

type TOTPHandler struct {
	totpService *service.TOTPService
}

func NewTOTPHandler(totpService *service.TOTPService) *TOTPHandler {
	return &TOTPHandler{
		totpService: totpService,
	}
}

func (h *TOTPHandler) Setup(c *gin.Context) {
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

	setupResp, err := h.totpService.Setup(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, setupResp)
}

func (h *TOTPHandler) Verify(c *gin.Context) {
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

	verifyResp, err := h.totpService.Verify(c.Request.Context(), userID, req.Code)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, verifyResp)
}

func (h *TOTPHandler) Disable(c *gin.Context) {
	var req dto.TOTPDisableRequest
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

	disableResp, err := h.totpService.Disable(c.Request.Context(), userID, req.Code)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, disableResp)
}
