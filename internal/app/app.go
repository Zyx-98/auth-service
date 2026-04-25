package app

import (
	"github.com/gin-gonic/gin"
	"github.com/hatuan/auth-service/config"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/hatuan/auth-service/internal/handler"
	"github.com/hatuan/auth-service/internal/middleware"
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
) {
	jwtMaker := jwt.NewMaker(
		a.cfg.JWT.AccessSecret,
		a.cfg.JWT.RefreshSecret,
		a.cfg.JWT.AccessExpiry,
		a.cfg.JWT.RefreshExpiry,
	)

	authService := service.NewAuthService(userRepo, roleRepo, permissionRepo, sessionRepo, jwtMaker)

	totpManager, err := totppkg.NewTOTPManager(a.cfg.TOTP.Issuer, a.cfg.TOTP.EncryptionKey)
	if err != nil {
		a.logger.Fatal("Failed to initialize TOTP manager", zap.Error(err))
	}
	totpService := service.NewTOTPService(totpManager, userRepo)

	authHandler := handler.NewAuthHandler(authService, totpService)

	googleOAuthClient := oauth.NewGoogleOAuthClient(
		a.cfg.OAuth.GoogleClientID,
		a.cfg.OAuth.GoogleClientSecret,
		a.cfg.OAuth.GoogleRedirectURL,
	)
	oauthService := service.NewOAuthService(googleOAuthClient, userRepo, roleRepo, permissionRepo, sessionRepo, jwtMaker)
	oauthHandler := handler.NewOAuthHandler(oauthService, a.redisClient)

	totpHandler := handler.NewTOTPHandler(totpService)

	rbacService := service.NewRBACService(roleRepo, permissionRepo, userRepo)
	rbacHandler := handler.NewRBACHandler(rbacService)

	a.setupRoutes(authHandler, oauthHandler, totpHandler, rbacHandler, jwtMaker)
}

func (a *App) setupRoutes(authHandler *handler.AuthHandler, oauthHandler *handler.OAuthHandler, totpHandler *handler.TOTPHandler, rbacHandler *handler.RBACHandler, jwtMaker *jwt.Maker) {
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
		loginLimited.POST("/2fa/verify-login", authHandler.VerifyTwoFA)

		public.POST("/refresh", authHandler.Refresh)
		public.POST("/introspect", authHandler.Introspect)
		public.GET("/login/google", oauthHandler.GoogleLoginRedirect)
		public.POST("/callback/google", oauthHandler.GoogleCallback)
	}

	// Protected auth routes
	protected := a.router.Group("/auth")
	protected.Use(middleware.AuthMiddleware(jwtMaker))
	{
		protected.POST("/logout", authHandler.Logout)
		protected.POST("/logout-all", authHandler.LogoutAll)
		protected.GET("/me", authHandler.GetProfile)
		protected.POST("/2fa/setup", totpHandler.Setup)
		protected.POST("/2fa/verify", totpHandler.Verify)
		protected.POST("/2fa/disable", totpHandler.Disable)
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
	}
}

func (a *App) Router() *gin.Engine {
	return a.router
}
