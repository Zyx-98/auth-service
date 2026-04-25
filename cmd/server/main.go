package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hatuan/auth-service/config"
	"github.com/hatuan/auth-service/internal/app"
	postgresrepo "github.com/hatuan/auth-service/internal/repository/postgres"
	redisrepo "github.com/hatuan/auth-service/internal/repository/redis"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger, err := initLogger(cfg.Server.Env)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Config loaded", zap.Int("port", cfg.Server.Port), zap.String("env", cfg.Server.Env))

	var db *gorm.DB
	var redisClient *redis.Client

	// Start server first so Cloud Run health check passes
	// Then attempt to initialize dependencies asynchronously
	gin.SetMode(ginMode(cfg.Server.Env))
	router := gin.Default()

	// Add a basic health endpoint that works before DB is ready
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		logger.Info("Starting server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Try to initialize database with retries
	for i := 0; i < 5; i++ {
		var initErr error
		db, initErr = initDB(cfg.Database.URL, logger)
		if initErr == nil {
			break
		}
		logger.Warn("Failed to initialize database, retrying...", zap.Error(initErr), zap.Int("attempt", i+1))
		time.Sleep(time.Duration(i+1) * 3 * time.Second)
	}
	if db == nil {
		logger.Fatal("Failed to initialize database after retries")
	}

	// Try to initialize Redis with retries
	for i := 0; i < 5; i++ {
		var initErr error
		redisClient, initErr = initRedis(cfg.Redis.Addr, cfg.Redis.Password)
		if initErr == nil {
			break
		}
		logger.Warn("Failed to initialize Redis, retrying...", zap.Error(initErr), zap.Int("attempt", i+1))
		time.Sleep(time.Duration(i+1) * 3 * time.Second)
	}
	if redisClient == nil {
		logger.Fatal("Failed to initialize Redis after retries")
	}
	defer redisClient.Close()

	if err := runMigrations(cfg.Database.URL, logger); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	// Setup the app with all repositories
	userRepo := postgresrepo.NewUserRepository(db)
	roleRepo := postgresrepo.NewRoleRepository(db)
	permissionRepo := postgresrepo.NewPermissionRepository(db)
	sessionRepo := redisrepo.NewSessionRepository(redisClient, cfg.JWT.RefreshExpiry)

	authApp := app.NewApp(router, db, cfg, redisClient, logger)
	authApp.Setup(userRepo, roleRepo, permissionRepo, sessionRepo)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
}

func initLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func initDB(dsn string, logger *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Info("Database connection established")
	return db, nil
}

func initRedis(addr, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

func runMigrations(dsn string, logger *zap.Logger) error {
	// DSN already includes postgres:// prefix
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	logger.Info("Migrations completed")
	return nil
}

func ginMode(env string) string {
	if env == "production" {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}

