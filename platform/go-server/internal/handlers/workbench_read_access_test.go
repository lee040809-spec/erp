package handlers

import (
	"github.com/gin-gonic/gin"
	"machining-erp/pkg/database"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWorkbenchReadAccessTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" || os.Getenv("MYSQL_DATABASE") != "platform_test" {
		t.Skip("explicit test database required")
	}
	db, err := database.Connect(&database.Config{Host: os.Getenv("MYSQL_HOST"), Port: 3306, User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD"), Database: os.Getenv("MYSQL_DATABASE")})
	if err != nil {
		t.Fatal("connection failed")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	pool.SetConnMaxIdleTime(0)
	defer pool.Close()
	queries := []string{
		"CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)",
		"CREATE TEMPORARY TABLE permissions(role_id BIGINT,corp_id BIGINT,resource_id BIGINT)",
		"CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100))",
		"INSERT INTO users VALUES(1,0,10),(2,0,20),(3,0,30),(4,1,10)",
		"INSERT INTO resources VALUES(1,'production_dashboard'),(2,'production_orders'),(3,'dashboard'),(4,'orders')",
		"INSERT INTO permissions VALUES(10,0,1),(20,0,3),(30,0,4),(10,1,2)",
	}
	for _, q := range queries {
		if err = db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	h := NewProductionWorkbenchHandler(db)
	for _, tc := range []struct {
		id, corp    int64
		role        string
		read, write bool
	}{{1, 0, "user", true, true}, {2, 0, "user", true, false}, {3, 0, "user", false, false}, {4, 1, "user", true, true}, {1, 1, "user", false, false}, {99, 0, "admin", true, true}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Set("userID", tc.id)
		c.Set("corpID", tc.corp)
		c.Set("roleType", tc.role)
		if h.allowed(c, false) != tc.read || h.allowed(c, true) != tc.write || h.canManage(c) != tc.write {
			t.Fatalf("scope/permission changed for user %d corp %d", tc.id, tc.corp)
		}
	}
}
