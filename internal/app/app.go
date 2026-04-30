package app

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hatuan/auth-service/config"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/hatuan/auth-service/internal/handler"
	"github.com/hatuan/auth-service/internal/middleware"
	redisrepo "github.com/hatuan/auth-service/internal/repository/redis"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/jwt"
	"github.com/hatuan/auth-service/pkg/oauth"
	totppkg "github.com/hatuan/auth-service/pkg/totp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)


type App struct {
	router      *gin.Engine
	db          *gorm.DB
	cfg         *config.Config
	redisClient *redis.Client
	logger      *zap.Logger
}

func NewApp(router *gin.Engine, db *gorm.DB, cfg *config.Config, redisClient *redis.Client, logger *zap.Logger) *App {
	return &App{
		router:      router,
		db:          db,
		cfg:         cfg,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (a *App) Setup(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	sessionRepo repository.SessionRepository,
	auditLogRepo repository.AuditLogRepository,
) {
	// Load templates for OAuth callback
	a.router.LoadHTMLGlob("templates/*.html")

	jwtMaker := jwt.NewMaker(
		a.cfg.JWT.AccessSecret,
		a.cfg.JWT.RefreshSecret,
		a.cfg.JWT.AccessExpiry,
		a.cfg.JWT.RefreshExpiry,
	)

	trustedDeviceRepo := redisrepo.NewTrustedDeviceRepository(a.redisClient, 30*24*time.Hour)

	auditLogService := service.NewAuditLogService(auditLogRepo)

	authService := service.NewAuthService(userRepo, roleRepo, permissionRepo, sessionRepo, trustedDeviceRepo, jwtMaker, auditLogService)

	totpManager, err := totppkg.NewTOTPManager(a.cfg.TOTP.Issuer, a.cfg.TOTP.EncryptionKey)
	if err != nil {
		a.logger.Fatal("Failed to initialize TOTP manager", zap.Error(err))
	}
	totpService := service.NewTOTPService(totpManager, userRepo, auditLogService)

	authHandler := handler.NewAuthHandler(authService, totpService)

	googleOAuthClient := oauth.NewGoogleOAuthClient(
		a.cfg.OAuth.GoogleClientID,
		a.cfg.OAuth.GoogleClientSecret,
		a.cfg.OAuth.GoogleRedirectURL,
	)
	oauthService := service.NewOAuthService(googleOAuthClient, userRepo, roleRepo, permissionRepo, sessionRepo, trustedDeviceRepo, totpService, jwtMaker)
	oauthHandler := handler.NewOAuthHandler(oauthService, a.redisClient, a.logger)

	totpHandler := handler.NewTOTPHandler(totpService)

	auditLogHandler := handler.NewAuditLogHandler(auditLogService)

	rbacService := service.NewRBACService(roleRepo, permissionRepo, userRepo, auditLogService)
	rbacHandler := handler.NewRBACHandler(rbacService)

	a.setupRoutes(authHandler, oauthHandler, totpHandler, rbacHandler, auditLogHandler, jwtMaker)
	a.setupStaticFiles()
}

func (a *App) setupRoutes(authHandler *handler.AuthHandler, oauthHandler *handler.OAuthHandler, totpHandler *handler.TOTPHandler, rbacHandler *handler.RBACHandler, auditLogHandler *handler.AuditLogHandler, jwtMaker *jwt.Maker) {
	// Global middlewares
	a.router.Use(middleware.CORSMiddleware(a.cfg.CORS.AllowedOrigins))
	a.router.Use(middleware.SecurityHeadersMiddleware())
	a.router.Use(middleware.LoggerMiddleware(a.logger))

	// Public auth routes with rate limiting
	public := a.router.Group("/auth")
	public.Use(middleware.RateLimitMiddleware(a.redisClient, a.cfg.RateLimit.GlobalLimit))
	{
		registerLimited := public.Group("")
		registerLimited.Use(middleware.RateLimitMiddleware(a.redisClient, "3-M"))
		registerLimited.POST("/register", authHandler.Register)

		loginLimited := public.Group("")
		loginLimited.Use(middleware.RateLimitMiddleware(a.redisClient, a.cfg.RateLimit.LoginLimit))
		loginLimited.POST("/login", authHandler.Login)

		public.POST("/refresh", authHandler.Refresh)
		public.POST("/introspect", authHandler.Introspect)
		public.POST("/login/google", oauthHandler.GoogleLoginRedirect)
		public.GET("/callback/google", oauthHandler.GoogleCallback)
		public.POST("/verify-oauth-totp", oauthHandler.VerifyOAuthTOTP)
	}

	// Protected auth routes (requires full access token)
	protected := a.router.Group("/auth")
	protected.Use(middleware.AuthMiddleware(jwtMaker))
	{
		protected.POST("/logout", authHandler.Logout)
		protected.POST("/logout-all", authHandler.LogoutAll)
		protected.GET("/me", authHandler.GetProfile)
		protected.POST("/2fa/setup", totpHandler.Setup)
		protected.GET("/2fa/qrcode", totpHandler.GetQRCode)
		protected.POST("/2fa/verify", totpHandler.Verify)
		protected.POST("/2fa/disable", totpHandler.Disable)
		protected.GET("/trusted-devices", authHandler.GetTrustedDevices)
		protected.DELETE("/trusted-devices", authHandler.DeleteTrustedDevices)
		protected.GET("/me/audit-logs", auditLogHandler.GetMyAuditLogs)
	}

	// 2FA login verification route (allows temporary tokens)
	twoFA := a.router.Group("/auth")
	twoFA.Use(middleware.TwoFAMiddleware(jwtMaker))
	{
		twoFA.POST("/2fa/verify-login", authHandler.VerifyTwoFA)
	}

	// RBAC Routes (Admin only)
	admin := a.router.Group("/admin")
	admin.Use(middleware.AuthMiddleware(jwtMaker))
	admin.Use(middleware.AdminMiddleware())
	{
		// Role Management
		admin.POST("/roles", rbacHandler.CreateRole)
		admin.GET("/roles", rbacHandler.ListRoles)
		admin.GET("/roles/:id", rbacHandler.GetRole)
		admin.PUT("/roles/:id", rbacHandler.UpdateRole)
		admin.DELETE("/roles/:id", rbacHandler.DeleteRole)

		// Permission Management
		admin.POST("/permissions", rbacHandler.CreatePermission)
		admin.GET("/permissions", rbacHandler.ListPermissions)
		admin.DELETE("/permissions/:id", rbacHandler.DeletePermission)

		// User Role Assignment
		admin.POST("/users/:user_id/roles", rbacHandler.AssignRoleToUser)
		admin.DELETE("/users/:user_id/roles/:role_id", rbacHandler.RemoveRoleFromUser)
		admin.GET("/users/:user_id/roles", rbacHandler.GetUserRoles)

		// Audit Logs
		admin.GET("/audit-logs", auditLogHandler.GetAuditLogs)
	}
}

func (a *App) Router() *gin.Engine {
	return a.router
}

func (a *App) setupStaticFiles() {
	paths := []string{
		"./web/dist",
		"/app/web/dist",
		"web/dist",
	}

	var found string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			found = path
			a.logger.Info("Serving frontend from", zap.String("path", path))
			break
		}
	}

	if found == "" {
		a.logger.Warn("Frontend directory not found, API-only mode")
		return
	}

	// Use NoRoute to serve the SPA index.html for unmatched routes
	a.router.NoRoute(func(c *gin.Context) {
		// Serve static files first
		indexPath := found + "/index.html"
		if _, err := os.Stat(found + c.Request.URL.Path); err == nil {
			// File exists, serve it
			c.File(found + c.Request.URL.Path)
			return
		}
		// File doesn't exist, serve index.html for SPA routing
		c.File(indexPath)
	})
}
