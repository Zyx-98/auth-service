package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/domain/entity"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/pkg/apperror"
	jwtpkg "github.com/hatuan/auth-service/pkg/jwt"
	"github.com/hatuan/auth-service/pkg/oauth"
)

type OAuthService struct {
	googleClient        *oauth.GoogleOAuthClient
	userRepo            repository.UserRepository
	roleRepo            repository.RoleRepository
	permissionRepo      repository.PermissionRepository
	sessionRepo         repository.SessionRepository
	trustedDeviceRepo   repository.TrustedDeviceRepository
	totpService         *TOTPService
	jwtMaker            *jwtpkg.Maker
}

func NewOAuthService(
	googleClient *oauth.GoogleOAuthClient,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	sessionRepo repository.SessionRepository,
	trustedDeviceRepo repository.TrustedDeviceRepository,
	totpService *TOTPService,
	jwtMaker *jwtpkg.Maker,
) *OAuthService {
	return &OAuthService{
		googleClient:      googleClient,
		userRepo:          userRepo,
		roleRepo:          roleRepo,
		permissionRepo:    permissionRepo,
		sessionRepo:       sessionRepo,
		trustedDeviceRepo: trustedDeviceRepo,
		totpService:       totpService,
		jwtMaker:          jwtMaker,
	}
}

func (s *OAuthService) GetGoogleAuthURL(state string) string {
	return s.googleClient.GetAuthURL(state)
}

func (s *OAuthService) VerifyOAuthTOTP(ctx context.Context, totpToken string, code string, userAgent string, ip string, trustDevice bool) (*dto.OAuthCallbackResponse, error) {
	claims, err := s.jwtMaker.VerifyAccessToken(totpToken)
	if err != nil {
		return nil, apperror.Unauthorized("Invalid or expired TOTP token")
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	valid, err := s.totpService.VerifyLogin(ctx, user.ID, code)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, apperror.Unauthorized("Invalid TOTP code")
	}

	return s.buildFullOAuthResponse(ctx, user, userAgent, ip, trustDevice)
}

func (s *OAuthService) buildFullOAuthResponse(ctx context.Context, user *entity.User, userAgent string, ip string, trustDevice bool) (*dto.OAuthCallbackResponse, error) {
	roleNames := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roleNames[i] = r.Name
	}

	permissions, err := s.permissionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		permissions = []*entity.Permission{}
	}

	permissionNames := make([]string, len(permissions))
	for i, p := range permissions {
		permissionNames[i] = p.Name
	}

	tokenPair, err := s.jwtMaker.CreateTokenPair(user.ID, user.Email, roleNames, permissionNames)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to create tokens", err)
	}

	jti, err := s.jwtMaker.ExtractJTI(tokenPair.RefreshToken)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to extract JTI", err)
	}

	session := entity.NewSession(user.ID, jti, tokenPair.RefreshToken, time.Now().Add(7*24*time.Hour))
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		return nil, apperror.InternalServerError("Failed to save session", err)
	}

	resp := &dto.OAuthCallbackResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(15 * 60),
		TokenType:    "Bearer",
		TOTPRequired: false,
	}

	if trustDevice {
		deviceToken := s.generateDeviceToken()
		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		device := entity.NewTrustedDevice(user.ID, deviceToken, userAgent, ip, expiresAt)

		if err := s.trustedDeviceRepo.Save(ctx, device); err != nil {
			return nil, apperror.InternalServerError("Failed to save trusted device", err)
		}

		resp.DeviceToken = deviceToken
	}

	return resp, nil
}

func (s *OAuthService) HandleGoogleCallback(ctx context.Context, code string, deviceToken string, userAgent string, ip string) (*dto.OAuthCallbackResponse, error) {
	googleUser, err := s.googleClient.ExchangeCode(ctx, code)
	if err != nil {
		return nil, apperror.Unauthorized("Failed to exchange Google code")
	}

	if googleUser.Email == "" {
		return nil, apperror.BadRequest("Google user email not found", nil)
	}

	user, err := s.userRepo.GetByGoogleID(ctx, googleUser.ID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to check user", err)
	}

	isNewUser := false

	if user == nil {
		user, err = s.userRepo.GetByEmail(ctx, googleUser.Email)
		if err != nil {
			return nil, apperror.InternalServerError("Failed to check email", err)
		}

		if user == nil {
			user = entity.NewUser(googleUser.Email)
			isNewUser = true
		}

		user.GoogleID = &googleUser.ID
		user.IsVerified = googleUser.VerifiedEmail

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, apperror.InternalServerError("Failed to create user", err)
		}

		if isNewUser {
			userRole, err := s.roleRepo.GetByName(ctx, "user")
			if err == nil && userRole != nil {
				_ = s.roleRepo.AssignRoleToUser(ctx, user.ID, userRole.ID)
			}
		}
	} else {
		user.IsVerified = googleUser.VerifiedEmail
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, apperror.InternalServerError("Failed to update user", err)
		}
	}

	if err := s.refreshUserRoles(ctx, user); err != nil {
		return nil, err
	}

	return s.buildOAuthResponse(ctx, user, isNewUser, deviceToken, userAgent, ip)
}

func (s *OAuthService) refreshUserRoles(ctx context.Context, user *entity.User) error {
	rolePtrs, err := s.roleRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return apperror.InternalServerError("Failed to fetch roles", err)
	}

	user.Roles = make([]entity.Role, len(rolePtrs))
	for i, r := range rolePtrs {
		if r != nil {
			user.Roles[i] = *r
		}
	}
	return nil
}

func (s *OAuthService) buildOAuthResponse(ctx context.Context, user *entity.User, isNewUser bool, deviceToken string, userAgent string, ip string) (*dto.OAuthCallbackResponse, error) {
	if user.TOTPEnabled {
		// Check if this device is already trusted
		if deviceToken != "" {
			trusted, err := s.trustedDeviceRepo.Exists(ctx, user.ID, deviceToken)
			if err == nil && trusted {
				// Device is trusted, skip 2FA
				return s.buildFullOAuthResponse(ctx, user, userAgent, ip, false)
			}
		}

		totpToken, err := s.jwtMaker.CreateCustomToken(user.ID, user.Email, []string{"totp:verify"}, 10*time.Minute, "")
		if err != nil {
			return nil, apperror.InternalServerError("Failed to create TOTP token", err)
		}

		return &dto.OAuthCallbackResponse{
			TOTPRequired: true,
			TOTPToken:    totpToken,
			IsNewUser:    isNewUser,
		}, nil
	}

	roleNames := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roleNames[i] = r.Name
	}

	permissions, err := s.permissionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		permissions = []*entity.Permission{}
	}

	permissionNames := make([]string, len(permissions))
	for i, p := range permissions {
		permissionNames[i] = p.Name
	}

	tokenPair, err := s.jwtMaker.CreateTokenPair(user.ID, user.Email, roleNames, permissionNames)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to create tokens", err)
	}

	jti, err := s.jwtMaker.ExtractJTI(tokenPair.RefreshToken)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to extract JTI", err)
	}

	session := entity.NewSession(user.ID, jti, tokenPair.RefreshToken, time.Now().Add(7*24*time.Hour))
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		return nil, apperror.InternalServerError("Failed to save session", err)
	}

	expiresIn := int64(15 * 60)

	return &dto.OAuthCallbackResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		IsNewUser:    isNewUser,
		TOTPRequired: false,
	}, nil
}

func (s *OAuthService) generateDeviceToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}
