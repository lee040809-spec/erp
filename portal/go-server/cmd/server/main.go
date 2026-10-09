package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"portal-erp/internal/config"
	"portal-erp/internal/handlers"
	"portal-erp/internal/middleware"
	"portal-erp/internal/repository"
	"portal-erp/internal/services"
	"portal-erp/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 根据 GO_ENV 加载不同的配置文件
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "local"
	}

	envFile := ".env." + env
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("Warning: Error loading %s: %v", envFile, err)
		} else {
			log.Printf("Loaded config from %s", envFile)
		}
	} else {
		godotenv.Load()
	}

	// 加载配置
	cfg := config.Load()

	// 连接数据库
	db, err := database.Connect(&database.Config{
		Host:     cfg.MySQLHost,
		User:     cfg.MySQLUser,
		Password: cfg.MySQLPassword,
		Database: cfg.MySQLDatabase,
		Port:     cfg.MySQLPort,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 初始化仓库
	userRepo := repository.NewUserRepository(db)
	superUserRepo := repository.NewSuperUserRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	resourceRepo := repository.NewResourceRepository(db)

	// 初始化服务
	authService := services.NewAuthService(
		userRepo,
		cfg.AdminUser,
		cfg.AdminPassword,
		cfg.MaxLoginAttempts,
		time.Duration(cfg.LockDurationMinutes)*time.Minute,
	)

	// 初始化管理员账号
	if err := authService.InitAdmin(); err != nil {
		log.Printf("Warning: Failed to init admin: %v", err)
	}

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userRepo, roleRepo, authService)
	superUserHandler := handlers.NewSuperUserHandler(superUserRepo, roleRepo, authService)
	companyHandler := handlers.NewCompanyHandler(companyRepo)
	roleHandler := handlers.NewRoleHandler(roleRepo, resourceRepo)
	resourceHandler := handlers.NewResourceHandler(resourceRepo, roleRepo)

	// 创建 Gin 路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/api/portal/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": "portal-v1.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 认证路由 (无需鉴权)
	auth := r.Group("/api/portal")
	{
		auth.POST("/login", authHandler.Login)
		auth.GET("/auth/status", authHandler.Status)
		auth.POST("/logout", authHandler.Logout)
	}

	// 需要鉴权的路由
	api := r.Group("/api/portal")
	api.Use(middleware.AuthMiddleware(authService))
	{
		// 公司管理
		api.GET("/companies", companyHandler.List)
		api.GET("/companies/:id", companyHandler.Get)
		api.POST("/companies", companyHandler.Create)
		api.PATCH("/companies/:id", companyHandler.Update)
		api.DELETE("/companies/:id", companyHandler.Delete)

		// 用户管理
		api.GET("/users", userHandler.List)
		api.GET("/users/:id", userHandler.Get)
		api.POST("/users", userHandler.Create)
		api.PATCH("/users/:id", userHandler.Update)
		api.DELETE("/users/:id", userHandler.Delete)
		api.POST("/users/:id/reset-password", userHandler.ResetPassword)
		api.POST("/users/:id/unlock", userHandler.Unlock)

		// 超级用户管理
		api.GET("/super-users", superUserHandler.List)
		api.GET("/super-users/:id", superUserHandler.Get)
		api.POST("/super-users", superUserHandler.Create)
		api.PATCH("/super-users/:id", superUserHandler.Update)
		api.DELETE("/super-users/:id", superUserHandler.Delete)
		api.POST("/super-users/:id/reset-password", superUserHandler.ResetPassword)
		api.POST("/super-users/:id/unlock", superUserHandler.Unlock)

		// 密码修改
		api.POST("/auth/change-password", authHandler.ChangePassword)

		// 角色管理
		api.GET("/roles", roleHandler.List)
		api.GET("/roles/:id", roleHandler.Get)
		api.POST("/roles", roleHandler.Create)
		api.PATCH("/roles/:id", roleHandler.Update)
		api.DELETE("/roles/:id", roleHandler.Delete)

		// 资源管理
		api.GET("/resources", resourceHandler.List)
		api.GET("/resources/menus", resourceHandler.GetMenus)
		api.GET("/resources/:id", resourceHandler.Get)
		api.POST("/resources", resourceHandler.Create)
		api.PATCH("/resources/:id", resourceHandler.Update)
		api.DELETE("/resources/:id", resourceHandler.Delete)
	}

	// 启动服务器
	log.Printf("Portal server running on :%d", cfg.ServerPort)
	if err := r.Run(fmt.Sprintf(":%d", cfg.ServerPort)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}