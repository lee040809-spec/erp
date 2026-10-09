package handlers

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModulePermissionConnectionFailureIsNotForbidden(t *testing.T) {
	pool, e := sql.Open("mysql", "")
	if e != nil {
		t.Fatal(e)
	}
	pool.Close()
	db, e := gorm.Open(mysql.New(mysql.Config{Conn: pool, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("roleType", "user")
	c.Set("userID", int64(1))
	c.Set("corpID", int64(0))
	if requireModuleAccess(db, c, "orders") || rec.Code != 500 || !strings.Contains(rec.Body.String(), "数据库连接暂时异常") {
		t.Fatalf("connection failure incorrectly represented: %d %s", rec.Code, rec.Body.String())
	}
	if hasModuleAccess(db, c, "orders") {
		t.Fatal("failed permission query granted access")
	}
}
