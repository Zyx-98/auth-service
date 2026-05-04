package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/Zyx-98/auth-service/internal/domain/repository"
	"github.com/Zyx-98/auth-service/internal/dto"
	"github.com/Zyx-98/auth-service/pkg/apperror"
	totppkg "github.com/Zyx-98/auth-service/pkg/totp"
)

type TOTPService struct {
	totpManager     *totppkg.TOTPManager
	userRepo        repository.UserRepository
	auditLogService *AuditLogService
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

	if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.setup", "create", "success", nil); err != nil {
		return nil, apperror.InternalServerError("Failed to log audit event", err)
	}

	return &dto.TOTPSetupResponse{
		Secret:   secretInfo.Secret,
		QRCode:   secretInfo.QRCode,
		OTPAuth:  secretInfo.OTPAuth,
		Verified: false,
	}, nil
}

func (s *TOTPService) EnableTwoFA(ctx context.Context, userID uuid.UUID, code string) (*dto.TOTPVerifyResponse, error) {
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
		if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.enable", "create", "failure", nil); err != nil {
			return nil, apperror.InternalServerError("Failed to log audit event", err)
		}
		return &dto.TOTPVerifyResponse{Verified: false}, nil
	}

	user.TOTPEnabled = true

	plainCodes, err := totppkg.GenerateBackupCodes(8)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to generate backup codes", err)
	}

	encryptedCodes := make([]string, len(plainCodes))
	for i, plainCode := range plainCodes {
		encrypted, err := s.totpManager.EncryptSecret(plainCode)
		if err != nil {
			return nil, apperror.InternalServerError("Failed to encrypt backup code", err)
		}
		encryptedCodes[i] = encrypted
	}

	backupCodesJSON, err := json.Marshal(encryptedCodes)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to marshal backup codes", err)
	}

	backupCodesStr := string(backupCodesJSON)
	user.BackupCodes = &backupCodesStr

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.InternalServerError("Failed to save user with backup codes", err)
	}

	if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.enable", "create", "success", nil); err != nil {
		return nil, apperror.InternalServerError("Failed to log audit event", err)
	}

	return &dto.TOTPVerifyResponse{
		Verified:    true,
		BackupCodes: plainCodes,
	}, nil
}

func (s *TOTPService) VerifyBackupCode(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return false, apperror.NotFound("User not found")
	}

	if user.BackupCodes == nil {
		if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.backup_code", "verify", "failure", nil); err != nil {
			return false, apperror.InternalServerError("Failed to log audit event", err)
		}
		return false, nil
	}

	var encryptedCodes []string
	if err := json.Unmarshal([]byte(*user.BackupCodes), &encryptedCodes); err != nil {
		return false, apperror.InternalServerError("Failed to parse backup codes", err)
	}

	for i, encrypted := range encryptedCodes {
		decrypted, err := s.totpManager.DecryptSecret(encrypted)
		if err != nil {
			continue
		}

		if decrypted == code {
			encryptedCodes = append(encryptedCodes[:i], encryptedCodes[i+1:]...)
			backupCodesJSON, err := json.Marshal(encryptedCodes)
			if err != nil {
				return false, apperror.InternalServerError("Failed to marshal backup codes", err)
			}

			backupCodesStr := string(backupCodesJSON)
			user.BackupCodes = &backupCodesStr

			if err := s.userRepo.Update(ctx, user); err != nil {
				return false, apperror.InternalServerError("Failed to update backup codes", err)
			}

			if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.backup_code", "verify", "success", nil); err != nil {
				return false, apperror.InternalServerError("Failed to log audit event", err)
			}

			return true, nil
		}
	}

	if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.backup_code", "verify", "failure", nil); err != nil {
		return false, apperror.InternalServerError("Failed to log audit event", err)
	}

	return false, nil
}

func (s *TOTPService) GetUnusedBackupCodeCount(ctx context.Context, userID uuid.UUID) (int, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return 0, apperror.NotFound("User not found")
	}

	if user.BackupCodes == nil {
		return 0, nil
	}

	var codes []string
	if err := json.Unmarshal([]byte(*user.BackupCodes), &codes); err != nil {
		return 0, apperror.InternalServerError("Failed to parse backup codes", err)
	}

	return len(codes), nil
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

	if valid {
		return true, nil
	}

	backupValid, err := s.VerifyBackupCode(ctx, userID, code)
	if err != nil {
		return false, err
	}

	return backupValid, nil
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
		backupValid, err := s.VerifyBackupCode(ctx, userID, code)
		if err != nil {
			return nil, err
		}
		if !backupValid {
			if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.disable", "delete", "failure", nil); err != nil {
				return nil, apperror.InternalServerError("Failed to log audit event", err)
			}
			return &dto.TOTPDisableResponse{Disabled: false}, nil
		}
	}

	user.TOTPEnabled = false
	user.TOTPSecret = nil
	user.BackupCodes = nil

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.InternalServerError("Failed to disable 2FA", err)
	}

	if err := s.auditLogService.LogAuthEvent(ctx, &userID, "2fa.disable", "delete", "success", nil); err != nil {
		return nil, apperror.InternalServerError("Failed to log audit event", err)
	}

	return &dto.TOTPDisableResponse{Disabled: true}, nil
}
