// Package workflowtest creates connection-local fixtures only; it never alters persistent tables.
package workflowtest

import (
	"gorm.io/gorm"
	"machining-erp/internal/workflowmigration"
	"strings"
	"testing"
)

func Tables(t *testing.T, db *gorm.DB, temporaryParents ...string) {
	t.Helper()
	for _, ddl := range workflowmigration.Tables {
		if e := db.Exec(strings.Replace(ddl, "CREATE TABLE IF NOT EXISTS", "CREATE TEMPORARY TABLE", 1)).Error; e != nil {
			t.Fatal(e)
		}
	}
	allowed := map[string]bool{}
	for _, table := range temporaryParents {
		allowed[table] = true
	}
	for _, c := range workflowmigration.Columns {
		if !allowed[c.Table] {
			continue
		}
		rows, e := db.Raw("SHOW COLUMNS FROM " + c.Table + " LIKE '" + c.Name + "'").Rows()
		if e != nil {
			t.Fatal(e)
		}
		has := rows.Next()
		rows.Close()
		if !has {
			if e = db.Exec("ALTER TABLE " + c.Table + " ADD COLUMN " + c.Name + " " + c.Definition).Error; e != nil {
				t.Fatal(e)
			}
		}
	}
}
