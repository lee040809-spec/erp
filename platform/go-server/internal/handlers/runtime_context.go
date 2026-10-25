package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/runtimeinfo"
	"machining-erp/pkg/response"
	"net"
	"os"
	"strings"
	"time"
)

// No passwords, database usernames or server paths are exposed.
func RuntimeContext(db *gorm.DB, host, database string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		pool, e := db.DB()
		ready := e == nil
		if ready {
			ready = pool.PingContext(ctx) == nil
		}
		source := "远端数据库联机"
		ip := net.ParseIP(host)
		if strings.EqualFold(host, "localhost") || ip != nil && ip.IsLoopback() {
			source = "数据库与后端同机"
		}
		// Same-host database is not proof of an offline-capable installation.
		build := "未知"
		if file, e := os.Executable(); e == nil {
			if stat, e := os.Stat(file); e == nil {
				build = stat.ModTime().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04")
			}
		}
		backupStatus := "未接入备份状态，请管理员核对备份文件"
		if backup, e := runtimeinfo.ReadBackup(os.Getenv("ERP_BACKUP_STATUS_FILE"), host, database); e == nil {
			backupStatus = "数据库最近备份：" + backup.CompletedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04") + "（图纸文件需另行备份）"
		}
		response.Success(c, gin.H{"source": source, "company": c.GetString("corpName"), "corp_id": middleware.GetCorpID(c), "database_ready": ready, "build_time": build, "version": "开发中", "backup_status": backupStatus})
	}
}
