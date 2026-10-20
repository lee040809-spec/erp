package database

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Normal startup only reads existing tables. Schema changes require an explicit deployment step.
func CheckCoreTables(db *gorm.DB) error {
	queries := []string{}
	for _, table := range []string{"companies", "users", "roles", "permissions", "resources", "print_templates"} {
		queries = append(queries, "SELECT id FROM `"+table+"` WHERE 1=0")
	}
	// One read-only round trip resolves all required tables/columns without reading rows.
	// Never use Migrator.HasTable: it discards lookup errors and can mistake timeouts for absence.
	if err := db.Exec(strings.Join(queries, " UNION ALL ")).Error; err != nil {
		return fmt.Errorf("基础表只读检查失败（缺表时请执行数据库初始化或迁移）: %w", err)
	}
	return nil
}

func ConnectForStartup(ctx context.Context, cfg *Config, check func(*gorm.DB) error, retryLog func(int)) (*gorm.DB, error) {
	return retryStartup(ctx, func() (*gorm.DB, error) {
		db, err := connectContext(ctx, cfg)
		if err != nil {
			return nil, err
		}
		if err = check(db.WithContext(ctx)); err != nil {
			if pool, e := db.DB(); e == nil {
				_ = pool.Close()
			}
			return nil, err
		}
		return db, nil
	}, 2*time.Second, retryLog)
}

func retryStartup(ctx context.Context, attempt func() (*gorm.DB, error), delay time.Duration, retryLog func(int)) (*gorm.DB, error) {
	var last error
	for i := 1; i <= 3; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		db, err := attempt()
		if err == nil {
			return db, nil
		}
		last = err
		if i == 3 || !transientStartupError(err) {
			break
		}
		if retryLog != nil {
			retryLog(i + 1)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, last
}

func transientStartupError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1205, 1213, 2006, 2013:
			return true
		default:
			return false // Wrong credentials, missing tables and SQL errors must not trigger DDL or blind retries.
		}
	}
	var networkError net.Error
	return errors.As(err, &networkError) || errors.Is(err, driver.ErrBadConn) || errors.Is(err, mysqlDriver.ErrInvalidConn) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}
