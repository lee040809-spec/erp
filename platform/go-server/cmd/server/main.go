package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"machining-erp/internal/config"
	"machining-erp/internal/handlers"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/internal/services"
	"machining-erp/pkg/database"
	"machining-erp/pkg/jwt"

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
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(256 << 20)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				log.Printf("runtime memory heap=%dMiB inuse=%dMiB sys=%dMiB goroutines=%d gc=%d soft_limit=%dMiB", m.HeapAlloc>>20, m.HeapInuse>>20, m.Sys>>20, runtime.NumGoroutine(), m.NumGC, debug.SetMemoryLimit(-1)>>20)
			}
		}
	}()

	// 连接数据库 (GORM)
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

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()
	go database.KeepReadConnectionsWarm(ctx, sqlDB, 3)

	// 自动迁移新表
	if err := db.AutoMigrate(
		&models.Company{},
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Resource{},
		&models.PrintTemplate{},
	); err != nil {
		log.Fatalf("Failed to auto migrate: %v", err)
	}

	// 初始化仓库
	userRepo := repository.NewUserRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	resourceRepo := repository.NewResourceRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	orderItemRepo := repository.NewOrderItemRepository(db)
	processRepo := repository.NewOrderProcessRepository(db)
	materialRepo := repository.NewMaterialRepository(db)
	remnantRepo := repository.NewRemnantRepository(db)
	adventRuleRepo := repository.NewAdventRuleRepository(db)
	printTemplateRepo := repository.NewPrintTemplateRepository(db)

	// 初始化服务
	authService := services.NewAuthService(
		userRepo,
		companyRepo,
		resourceRepo,
		permissionRepo,
		roleRepo,
		jwt.JWTConfig{
			SecretKey:    cfg.JWTSecret,
			ExpiresHours: cfg.JWTExpiresHours,
			Issuer:       cfg.JWTIssuer,
		},
		cfg.MaxLoginAttempts,
		time.Duration(cfg.LockDurationMinutes)*time.Minute,
	)
	orderNumSvc := services.NewOrderNumberService(db)
	emailSvc := services.NewEmailService(cfg, userRepo)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	customerHandler := handlers.NewCustomerHandler(customerRepo, db)
	orderHandler := handlers.NewOrderHandler(orderRepo, orderItemRepo, processRepo, orderNumSvc, db)
	drawingHandler := handlers.NewPartDrawingHandler(db, os.Getenv("DRAWING_STORAGE_DIR"))
	itemHandler := handlers.NewOrderItemHandler(orderItemRepo, orderRepo, processRepo, db)
	processHandler := handlers.NewProcessHandler(processRepo, orderItemRepo, orderRepo, db)
	materialHandler := handlers.NewMaterialHandler(materialRepo, db)
	remnantHandler := handlers.NewRemnantHandler(remnantRepo, db)
	financeHandler := handlers.NewFinanceHandler(db)
	productionFinanceHandler := handlers.NewProductionFinanceHandler(db)
	productionWorkbenchHandler := handlers.NewProductionWorkbenchHandler(db)
	processTimelineHandler := handlers.NewProcessTimelineHandler(db)
	productionFlowHandler := handlers.NewProductionFlowHandler(db)
	outsourcingHandler := handlers.NewOutsourcingHandler(db)
	businessEventHandler := handlers.NewBusinessEventHandler(db)
	businessPolicyHandler := handlers.NewBusinessPolicyHandler(db)
	adventRuleHandler := handlers.NewAdventRuleHandler(adventRuleRepo, db)
	deliveryService := services.NewDeliveryService(db)
	reconciliationService := services.NewReconciliationService(db)
	printTemplateHandler := handlers.NewPrintTemplateHandler(printTemplateRepo, deliveryService, reconciliationService)
	userSettingsHandler := handlers.NewUserSettingsHandler(userRepo, emailSvc, authService)
	userHandler := handlers.NewUserHandler(userRepo, roleRepo, authService)
	roleHandler := handlers.NewRoleHandler(roleRepo, permissionRepo, resourceRepo, authService)
	resourceHandler := handlers.NewResourceHandler(resourceRepo, authService)

	// 创建 Gin 路由
	r := gin.New()

	// 中间件
	r.Use(gin.Recovery())
	r.Use(middleware.ResourceLimits())
	r.MaxMultipartMemory = 1 << 20
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[REQUEST] %s %s | %d | %v", method, path, status, latency)
	})
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/api/platform/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": "go-gorm",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 认证路由 (无需鉴权)
	auth := r.Group("/api/platform")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/check-user", authHandler.CheckUser)
		auth.GET("/auth/status", authHandler.Status)
		auth.POST("/logout", authHandler.Logout)
	}

	// 需要鉴权的路由
	api := r.Group("/api/platform")
	api.Use(middleware.AuthMiddleware(authService))
	api.GET("/system/runtime", handlers.RuntimeMetrics(db))
	api.GET("/order-items/:itemId/drawing", orderHandler.Drawing)
	api.GET("/order-items/:itemId/drawings", drawingHandler.List)
	api.POST("/drawings", drawingHandler.Upload)
	api.GET("/drawings/search", drawingHandler.Search)
	api.POST("/drawings/reference", drawingHandler.Reference)
	api.PATCH("/drawings/:drawingId", drawingHandler.State)
	api.GET("/drawings/:drawingId/file", drawingHandler.File)
	{
		// 当前用户信息
		api.GET("/me", authHandler.GetCurrentUser)

		// 客户
		api.GET("/customers", customerHandler.List)
		api.POST("/customers", customerHandler.Create)
		api.PATCH("/customers/:id", customerHandler.Update)
		api.DELETE("/customers/:id", customerHandler.Delete)

		// 订单
		api.GET("/orders", orderHandler.List)
		api.POST("/orders", orderHandler.Create)
		api.PATCH("/orders/:id", orderHandler.Update)
		api.DELETE("/orders/:id", orderHandler.Delete)

		api.GET("/production/items/:itemId/flow", productionFlowHandler.Read)
		api.POST("/production/items/:itemId/flow/activate", productionFlowHandler.Activate)
		api.PUT("/production/items/:itemId/delivery-batches", productionFlowHandler.Batches)
		api.POST("/production/items/:itemId/processes/:processId/quantities", productionFlowHandler.Report)
		api.GET("/outsourcing/:dispatchId/cost-status", outsourcingHandler.CostStatus)
		api.GET("/outsourcing", outsourcingHandler.List)
		api.GET("/outsourcing-suppliers", outsourcingHandler.Suppliers)
		api.POST("/outsourcing-suppliers", outsourcingHandler.SaveSupplier)
		api.PATCH("/outsourcing-suppliers/:supplierId", outsourcingHandler.SaveSupplier)
		api.GET("/outsourcing-orders", outsourcingHandler.Orders)
		api.POST("/outsourcing-orders", outsourcingHandler.CreateOrder)
		api.PATCH("/outsourcing-orders/:outsourceOrderId", outsourcingHandler.UpdateOrder)
		api.DELETE("/outsourcing-orders/:outsourceOrderId", outsourcingHandler.DeleteOrder)
		api.PATCH("/outsourcing/:dispatchId", outsourcingHandler.UpdateDispatch)
		api.DELETE("/outsourcing/:dispatchId", outsourcingHandler.DeleteDispatch)
		api.POST("/outsourcing-orders/:outsourceOrderId/cancel", outsourcingHandler.CancelOrder)
		api.POST("/outsourcing-orders/:outsourceOrderId/dispatch", outsourcingHandler.DispatchOrder)
		api.POST("/outsourcing", outsourcingHandler.Dispatch)
		api.POST("/outsourcing/:dispatchId/returns", outsourcingHandler.Receive)
		api.POST("/outsourcing/:dispatchId/rework-send", outsourcingHandler.Rework)
		api.POST("/outsourcing/:dispatchId/settlement", outsourcingHandler.Settle)
		api.GET("/production/workbench", productionWorkbenchHandler.List)
		api.GET("/production/items/:itemId/timeline", processTimelineHandler.List)
		api.PATCH("/production/items/:itemId/route", processTimelineHandler.SaveRoute)
		api.PATCH("/production/items/:itemId/processes/:processId/timeline", processTimelineHandler.Save)
		api.GET("/production/items/:itemId/plan", productionWorkbenchHandler.Plan)
		api.PATCH("/production/plans", productionWorkbenchHandler.Save)
		api.PATCH("/production/items/:itemId/quantities", productionWorkbenchHandler.SaveQuantities)
		api.GET("/production/items/:itemId/drawing", productionWorkbenchHandler.Detail)

		api.GET("/business-entry-policy", businessPolicyHandler.Get)
		api.PUT("/business-entry-policy", businessPolicyHandler.Save)
		api.GET("/orders/:id/business-records", businessEventHandler.List)
		api.POST("/orders/:id/business-records", businessEventHandler.Create)
		api.POST("/business-records/:recordId/resolve", businessEventHandler.Resolve)
		api.POST("/business-records/:recordId/reverse", businessEventHandler.Reverse)
		api.GET("/business-records/:recordId/delivery-print", businessEventHandler.Print)

		// 工作看板
		api.GET("/dashboard/items", orderHandler.GetDashboardItems)
		api.GET("/dashboard/stats", orderHandler.GetDashboardStats)

		// 订单项
		api.PATCH("/order-items/:itemId", itemHandler.Update)

		// 工序
		api.PATCH("/order-items/:itemId/processes/:processId", processHandler.Update)

		// 材料
		api.GET("/materials", materialHandler.List)
		api.POST("/materials", materialHandler.Create)

		// 余料
		api.GET("/remnants", remnantHandler.List)
		api.POST("/remnants", remnantHandler.Create)

		// 财务
		api.GET("/finance/reconciliation", financeHandler.GetReconciliation)
		api.GET("/finance/summary", productionFinanceHandler.GetSummary)
		api.GET("/finance/access", productionFinanceHandler.GetFinanceAccess)
		api.GET("/finance/history", productionFinanceHandler.PreviewHistory)
		api.POST("/finance/history", productionFinanceHandler.ImportHistory)
		api.GET("/finance/details", productionFinanceHandler.FinanceDetails)
		api.GET("/finance/pending-costs", productionFinanceHandler.PendingCosts)
		api.POST("/finance/allocations", productionFinanceHandler.AllocateCost)
		api.GET("/finance/cost-entries", productionFinanceHandler.ListCostEntries)
		api.POST("/finance/cost-entries", productionFinanceHandler.CreateCostEntry)
		api.POST("/finance/months/:month/close", productionFinanceHandler.CloseMonth)

		// 最终合格完工报工：产值以确认批次的数量和订单单价快照计算
		api.GET("/order-items/:itemId/completions", productionFinanceHandler.ListCompletions)
		api.POST("/order-items/:itemId/completions", productionFinanceHandler.CreateCompletion)

		// 预警规则
		api.GET("/advent-rules", adventRuleHandler.List)
		api.POST("/advent-rules", adventRuleHandler.Create)
		api.PATCH("/advent-rules/:id", adventRuleHandler.Update)
		api.DELETE("/advent-rules/:id", adventRuleHandler.Delete)
		// 打印模板
		api.GET("/print-templates", printTemplateHandler.List)
		api.GET("/print-templates/by-button", printTemplateHandler.FindByButtonKey) // 必须在 :id 路由之前
		api.POST("/print-templates", printTemplateHandler.Create)
		api.PATCH("/print-templates/:id", printTemplateHandler.Update)
		api.DELETE("/print-templates/:id", printTemplateHandler.Delete)
		api.POST("/print-templates/:id/import-excel", printTemplateHandler.ImportExcel)
		api.GET("/print-templates/:id/download-excel", printTemplateHandler.DownloadExcel)
		api.POST("/print-templates/:id/render", printTemplateHandler.RenderTemplate)
		api.POST("/print-templates/:id/generate-excel", printTemplateHandler.GeneratePrintExcel)

		// 用户设置
		api.GET("/user/info", userSettingsHandler.GetUserInfo)
		api.POST("/user/change-password", userSettingsHandler.ChangePassword)
		api.POST("/user/reset-password", userSettingsHandler.ResetPassword)
		api.POST("/user/change-email", userSettingsHandler.ChangeEmail)

		// 用户管理
		api.GET("/users", userHandler.List)
		api.POST("/users", userHandler.Create)
		api.PATCH("/users/:id", userHandler.Update)
		api.DELETE("/users/:id", userHandler.Delete)
		api.POST("/users/:id/reset-password", userHandler.ResetPassword)
		api.POST("/users/:id/bind-role", userHandler.BindRole)

		// 角色管理
		api.GET("/roles", roleHandler.List)
		api.POST("/roles", roleHandler.Create)
		api.PATCH("/roles/:id", roleHandler.Update)
		api.DELETE("/roles/:id", roleHandler.Delete)
		api.GET("/roles/:id/permissions", roleHandler.GetPermissions)
		api.GET("/roles/:id/resources", roleHandler.GetAllResources)
		api.POST("/roles/:id/permissions", roleHandler.BindPermission)
		api.DELETE("/roles/:id/permissions/:permission_id", roleHandler.UnbindPermission)

		// 资源管理（只读）
		api.GET("/resources", resourceHandler.List)
		api.GET("/resources/page-buttons", resourceHandler.GetWithPageResources)
	}

	// 启动服务器
	log.Printf("Go server running on :%d", cfg.ServerPort)
	server := &http.Server{Addr: fmt.Sprintf(":%d", cfg.ServerPort), Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	serverError := make(chan error, 1)
	go func() { serverError <- server.ListenAndServe() }()
	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Server stopped: %v", err)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("Shutdown: %v", err)
		}
	}
}
