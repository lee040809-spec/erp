package handlers

import (
	"runtime"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
)

func RuntimeMetrics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if middleware.GetRoleType(c) != "admin" {
			c.AbortWithStatusJSON(403, gin.H{"error": "仅管理员可查看运行指标"})
			return
		}
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		pool, err := db.DB()
		if err != nil {
			c.JSON(500, gin.H{"error": "无法读取连接池状态"})
			return
		}
		stats := pool.Stats()
		c.JSON(200, gin.H{"heap_alloc_bytes": m.HeapAlloc, "heap_inuse_bytes": m.HeapInuse, "runtime_sys_bytes": m.Sys, "goroutines": runtime.NumGoroutine(), "gc_count": m.NumGC, "soft_memory_limit_bytes": debug.SetMemoryLimit(-1), "db_open": stats.OpenConnections, "db_in_use": stats.InUse, "db_idle": stats.Idle, "db_wait_count": stats.WaitCount})
	}
}
