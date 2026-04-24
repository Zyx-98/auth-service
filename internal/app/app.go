package app

import (
	"github.com/gin-gonic/gin"
	"github.com/hatuan/auth-service/config"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/hatuan/auth-service/internal/handler"
	"github.com/hatuan/auth-service/internal/middleware"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/jwt"
	"gorm.io/gorm"
)

type App struct {
	router *gin.Engine
	db     *gorm.DB
	cfg    *config.Config
}

func NewApp(router *gin.Engine, db *gorm.DB, cfg *config.Config) *App {
	return &App{
		router: router,
		db:     db,
		cfg:    cfg,
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

	a.setupRoutes(authHandler, jwtMaker)
}

func (a *App) setupRoutes(authHandler *handler.AuthHandler, jwtMaker *jwt.Maker) {
	public := a.router.Group("/auth")
	{
		public.POST("/register", authHandler.Register)
		public.POST("/login", authHandler.Login)
		public.POST("/refresh", authHandler.Refresh)
		public.POST("/introspect", authHandler.Introspect)
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
