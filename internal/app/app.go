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
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	router      *gin.Engine
	db          *gorm.DB
	cfg         *config.Config
	redisClient *redis.Client
}

func NewApp(router *gin.Engine, db *gorm.DB, cfg *config.Config, redisClient *redis.Client) *App {
	return &App{
		router:      router,
		db:          db,
		cfg:         cfg,
		redisClient: redisClient,
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
	authHandler := handler.NewAuthHandler(authService)

	googleOAuthClient := oauth.NewGoogleOAuthClient(
		a.cfg.OAuth.GoogleClientID,
		a.cfg.OAuth.GoogleClientSecret,
		a.cfg.OAuth.GoogleRedirectURL,
	)
	oauthService := service.NewOAuthService(googleOAuthClient, userRepo, roleRepo, permissionRepo, sessionRepo, jwtMaker)
	oauthHandler := handler.NewOAuthHandler(oauthService, a.redisClient)

	a.setupRoutes(authHandler, oauthHandler, jwtMaker)
}

func (a *App) setupRoutes(authHandler *handler.AuthHandler, oauthHandler *handler.OAuthHandler, jwtMaker *jwt.Maker) {
	public := a.router.Group("/auth")
	{
		public.POST("/register", authHandler.Register)
		public.POST("/login", authHandler.Login)
		public.POST("/refresh", authHandler.Refresh)
		public.POST("/introspect", authHandler.Introspect)
		public.GET("/login/google", oauthHandler.GoogleLoginRedirect)
		public.POST("/callback/google", oauthHandler.GoogleCallback)
	}

	protected := a.router.Group("/auth")
	protected.Use(middleware.AuthMiddleware(jwtMaker))
	{
		protected.POST("/logout", authHandler.Logout)
		protected.POST("/logout-all", authHandler.LogoutAll)
		protected.GET("/me", authHandler.GetProfile)
	}
}

func (a *App) Router() *gin.Engine {
	return a.router
}
