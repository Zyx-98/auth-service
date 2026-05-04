package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/domain/entity"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/hash"
	"github.com/hatuan/auth-service/pkg/jwt"
)

type AuthService struct {
	userRepo          repository.UserRepository
	roleRepo          repository.RoleRepository
	permissionRepo    repository.PermissionRepository
	sessionRepo       repository.SessionRepository
	trustedDeviceRepo repository.TrustedDeviceRepository
	jwtMaker          *jwt.Maker
	auditLogService   *AuditLogService
}

func NewAuthService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	sessionRepo repository.SessionRepository,
	trustedDeviceRepo repository.TrustedDeviceRepository,
	jwtMaker *jwt.Maker,
	auditLogService *AuditLogService,
) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		roleRepo:          roleRepo,
		permissionRepo:    permissionRepo,
		sessionRepo:       sessionRepo,
		trustedDeviceRepo: trustedDeviceRepo,
		jwtMaker:          jwtMaker,
		auditLogService:   auditLogService,
	}
}

func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.TokenResponse, error) {
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to check email availability", err)
	}

	if existingUser != nil {
		return nil, apperror.Conflict("Email already in use", nil)
	}

	hashedPassword, err := hash.Hash(req.Password)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to hash password", err)
	}

	user := entity.NewUser(req.Email)
	user.PasswordHash = &hashedPassword
	user.IsVerified = true

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperror.InternalServerError("Failed to create user", err)
	}

	userRole, err := s.roleRepo.GetByName(ctx, "user")
	if err != nil {
		return nil, apperror.InternalServerError("Failed to get user role", err)
	}

	if userRole != nil {
		if err := s.roleRepo.AssignRoleToUser(ctx, user.ID, userRole.ID); err != nil {
			return nil, apperror.InternalServerError("Failed to assign role", err)
		}
	}

	if err := s.auditLogService.LogAuthEvent(ctx, &user.ID, "user.register", "create", "success", nil); err != nil {
		return nil, apperror.InternalServerError("Failed to log audit event", err)
	}

	return s.IssueTokens(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest, userAgent, ip string) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil || user.PasswordHash == nil {
		return nil, apperror.Unauthorized("Invalid email or password")
	}

	if !hash.Compare(*user.PasswordHash, req.Password) {
		return nil, apperror.Unauthorized("Invalid email or password")
	}

	if user.TOTPEnabled {
		if s.isTrustedDevice(ctx, user.ID, req.DeviceToken, userAgent, ip) {
			tokenResp, err := s.IssueTokens(ctx, user)
			if err != nil {
				return nil, err
			}
			if err = s.auditLogService.LogAuthEvent(ctx, &user.ID, "user.login", "authenticate", "success", map[string]any{
				"ip":             ip,
				"user_agent":     userAgent,
				"trusted_device": true,
			}); err != nil {
				return nil, apperror.InternalServerError("Failed to log audit event", err)
			}
			return &dto.LoginResponse{
				Token: tokenResp,
			}, nil
		}

		tempToken, err := s.createTempToken(ctx, user)
		if err != nil {
			return nil, err
		}
		if err = s.auditLogService.LogAuthEvent(ctx, &user.ID, "user.login", "authenticate", "success", map[string]any{
			"ip":           ip,
			"user_agent":   userAgent,
			"requires_2fa": true,
		}); err != nil {
			return nil, apperror.InternalServerError("Failed to log audit event", err)
		}
		return &dto.LoginResponse{
			RequiresTwoFA: true,
			TempToken:     tempToken,
		}, nil
	}

	tokenResp, err := s.IssueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	if err = s.auditLogService.LogAuthEvent(ctx, &user.ID, "user.login", "authenticate", "success", map[string]any{
		"ip":         ip,
		"user_agent": userAgent,
	}); err != nil {
		return nil, apperror.InternalServerError("Failed to log audit event", err)
	}

	return &dto.LoginResponse{
		Token: tokenResp,
	}, nil
}

func (s *AuthService) isTrustedDevice(ctx context.Context, userID uuid.UUID, deviceToken, userAgent, ip string) bool {
	if deviceToken != "" {
		trusted, err := s.trustedDeviceRepo.Exists(ctx, userID, deviceToken)
		if err == nil && trusted {
			return true
		}
	}

	trusted, err := s.trustedDeviceRepo.IsTrustedByUserAgentAndIP(ctx, userID, userAgent, ip)
	return err == nil && trusted
}

func (s *AuthService) Refresh(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
	claims, err := s.jwtMaker.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, apperror.Unauthorized("Invalid or expired refresh token")
	}

	exists, err := s.sessionRepo.Exists(ctx, claims.JTI)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to verify session", err)
	}

	if !exists {
		return nil, apperror.Unauthorized("Refresh token has been revoked")
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.Unauthorized("User not found")
	}

	if err := s.sessionRepo.DeleteByJTI(ctx, claims.JTI); err != nil {
		return nil, apperror.InternalServerError("Failed to revoke old token", err)
	}

	tokenResp, err := s.IssueTokens(ctx, user)
	if err == nil {
		if err := s.auditLogService.LogAuthEvent(ctx, &user.ID, "token.refresh", "refresh", "success", nil); err != nil {
			return nil, apperror.InternalServerError("Failed to log audit event", err)
		}
	}
	return tokenResp, err
}

func (s *AuthService) Logout(ctx context.Context, jti string) error {
	if err := s.sessionRepo.DeleteByJTI(ctx, jti); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) LogoutAllDevices(ctx context.Context, userID uuid.UUID) error {
	return s.sessionRepo.DeleteByUserID(ctx, userID)
}

func (s *AuthService) Introspect(ctx context.Context, req *dto.IntrospectRequest) *dto.IntrospectResponse {
	claims, err := s.jwtMaker.VerifyAccessToken(req.Token)
	if err != nil {
		return &dto.IntrospectResponse{Valid: false}
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return &dto.IntrospectResponse{Valid: false}
	}

	permissions, err := s.permissionRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		permissions = []*entity.Permission{}
	}

	permissionNames := make([]string, len(permissions))
	for i, p := range permissions {
		permissionNames[i] = p.Name
	}

	expiresAt := int64(0)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Unix()
	}

	return &dto.IntrospectResponse{
		Valid:       true,
		UserID:      claims.UserID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: permissionNames,
		ExpiresAt:   expiresAt,
	}
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	roleNames := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roleNames[i] = r.Name
	}

	return &dto.UserProfileResponse{
		ID:          user.ID,
		Email:       user.Email,
		Roles:       roleNames,
		TOTPEnabled: user.TOTPEnabled,
	}, nil
}

func (s *AuthService) IssueTokens(ctx context.Context, user *entity.User) (*dto.TokenResponse, error) {
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

	claims, err := s.jwtMaker.VerifyRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to extract token claims", err)
	}

	session := entity.NewSession(user.ID, claims.JTI, tokenPair.RefreshToken, claims.ExpiresAt.Time)
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		return nil, apperror.InternalServerError("Failed to save session", err)
	}

	accessClaims, err := s.jwtMaker.VerifyAccessToken(tokenPair.AccessToken)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to extract access token claims", err)
	}

	expiresIn := int64(0)
	if accessClaims.ExpiresAt != nil {
		expiresIn = accessClaims.ExpiresAt.Unix() - time.Now().Unix()
	}

	return &dto.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) createTempToken(ctx context.Context, user *entity.User) (string, error) {
	token, err := s.jwtMaker.CreateCustomToken(user.ID, user.Email, []string{"2fa:verify"}, 5*time.Minute, "")
	if err != nil {
		return "", apperror.InternalServerError("Failed to create temp token", err)
	}
	return token, nil
}

func (s *AuthService) IssueTempTokens(ctx context.Context, userID uuid.UUID) (*dto.TokenResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	return s.IssueTokens(ctx, user)
}

func (s *AuthService) IssueTokensWithTrust(ctx context.Context, userID uuid.UUID, userAgent, ip string, trustDevice bool) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	tokenResp, err := s.IssueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	loginResp := &dto.LoginResponse{
		Token: tokenResp,
	}

	if trustDevice {
		deviceToken := s.generateDeviceToken()
		deviceName := s.generateDeviceName(userAgent)
		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		device := entity.NewTrustedDevice(userID, deviceToken, userAgent, ip, deviceName, expiresAt)

		if err := s.trustedDeviceRepo.Save(ctx, device); err != nil {
			return nil, apperror.InternalServerError("Failed to save trusted device", err)
		}

		loginResp.DeviceToken = deviceToken
	}

	return loginResp, nil
}

func (s *AuthService) generateDeviceToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}

func (s *AuthService) generateDeviceName(userAgent string) string {
	browsers := map[*regexp.Regexp]string{
		regexp.MustCompile(`(?i)edg[e/]`):                    "Edge",
		regexp.MustCompile(`(?i)chrome`):                     "Chrome",
		regexp.MustCompile(`(?i)safari`):                     "Safari",
		regexp.MustCompile(`(?i)firefox`):                    "Firefox",
		regexp.MustCompile(`(?i)opera`):                      "Opera",
	}

	systems := map[*regexp.Regexp]string{
		regexp.MustCompile(`(?i)windows`):                    "Windows",
		regexp.MustCompile(`(?i)macintosh|mac os x`):         "macOS",
		regexp.MustCompile(`(?i)linux`):                      "Linux",
		regexp.MustCompile(`(?i)iphone|ios`):                 "iPhone",
		regexp.MustCompile(`(?i)android`):                    "Android",
	}

	browser := "Unknown Browser"
	for pattern, name := range browsers {
		if pattern.MatchString(userAgent) {
			browser = name
			break
		}
	}

	system := "Unknown OS"
	for pattern, name := range systems {
		if pattern.MatchString(userAgent) {
			system = name
			break
		}
	}

	return browser + " on " + system
}

func (s *AuthService) GetTrustedDevices(ctx context.Context, userID uuid.UUID) ([]*entity.TrustedDevice, error) {
	return s.trustedDeviceRepo.GetByUserID(ctx, userID)
}

func (s *AuthService) RevokeTrustedDevices(ctx context.Context, userID uuid.UUID) error {
	return s.trustedDeviceRepo.DeleteByUserID(ctx, userID)
}
