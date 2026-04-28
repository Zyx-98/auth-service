package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/pkg/apperror"
	totppkg "github.com/hatuan/auth-service/pkg/totp"
)

type TOTPService struct {
	totpManager         *totppkg.TOTPManager
	userRepo            repository.UserRepository
	auditLogService     *AuditLogService
}

func NewTOTPService(totpManager *totppkg.TOTPManager, userRepo repository.UserRepository, auditLogService *AuditLogService) *TOTPService {
	return &TOTPService{
		totpManager:     totpManager,
		userRepo:        userRepo,
		auditLogService: auditLogService,
	}
}

func (s *TOTPService) Setup(ctx context.Context, userID uuid.UUID) (*dto.TOTPSetupResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	secretInfo, err := s.totpManager.GenerateSecret(user.Email)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to generate TOTP secret", err)
	}

	encryptedSecret, err := s.totpManager.EncryptSecret(secretInfo.Secret)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to encrypt secret", err)
	}

	user.TOTPSecret = &encryptedSecret
	user.TOTPEnabled = false

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.InternalServerError("Failed to save TOTP secret", err)
	}

	s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.setup", "create", "success", nil)

	return &dto.TOTPSetupResponse{
		Secret:   secretInfo.Secret,
		QRCode:   secretInfo.QRCode,
		OTPAuth:  secretInfo.OTPAuth,
		Verified: false,
	}, nil
}

func (s *TOTPService) Verify(ctx context.Context, userID uuid.UUID, code string) (*dto.TOTPVerifyResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	if user.TOTPSecret == nil {
		return nil, apperror.BadRequest("2FA not set up", nil)
	}

	valid, err := s.totpManager.Verify(*user.TOTPSecret, code)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to verify code", err)
	}

	if !valid {
		s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.verify", "verify", "failure", nil)
		return &dto.TOTPVerifyResponse{Verified: false}, nil
	}

	user.TOTPEnabled = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.InternalServerError("Failed to enable 2FA", err)
	}

	s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.verify", "verify", "success", nil)

	return &dto.TOTPVerifyResponse{Verified: true}, nil
}

func (s *TOTPService) VerifyLogin(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return false, apperror.NotFound("User not found")
	}

	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return false, apperror.BadRequest("2FA not enabled", nil)
	}

	valid, err := s.totpManager.Verify(*user.TOTPSecret, code)
	if err != nil {
		return false, apperror.InternalServerError("Failed to verify code", err)
	}

	return valid, nil
}

func (s *TOTPService) GetQRCode(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	if user.TOTPSecret == nil {
		return nil, apperror.BadRequest("2FA not set up", nil)
	}

	decryptedSecret, err := s.totpManager.DecryptSecret(*user.TOTPSecret)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to decrypt secret", err)
	}

	qrImageBytes, err := s.totpManager.GenerateQRCode(decryptedSecret, user.Email)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to generate QR code", err)
	}

	return qrImageBytes, nil
}

func (s *TOTPService) Disable(ctx context.Context, userID uuid.UUID, code string) (*dto.TOTPDisableResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return nil, apperror.BadRequest("2FA not enabled", nil)
	}

	valid, err := s.totpManager.Verify(*user.TOTPSecret, code)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to verify code", err)
	}

	if !valid {
		s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.disable", "delete", "failure", nil)
		return &dto.TOTPDisableResponse{Disabled: false}, nil
	}

	user.TOTPEnabled = false
	user.TOTPSecret = nil

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.InternalServerError("Failed to disable 2FA", err)
	}

	s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.disable", "delete", "success", nil)

	return &dto.TOTPDisableResponse{Disabled: true}, nil
}
