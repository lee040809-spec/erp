package database

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	User     string
	Password string
	Database string
	Port     int
}

func Connect(cfg *Config) (*gorm.DB, error) {
	// 添加超时参数优化连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local&charset=utf8mb4&timeout=5s&readTimeout=10s&writeTimeout=10s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			LogLevel: logger.Warn, SlowThreshold: time.Second,
			ParameterizedQueries: true, Colorful: false,
		}), // Do not format large drawings/templates into SQL log messages.
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(20)                  // 增加最大连接数
	sqlDB.SetMaxIdleConns(10)                  // 增加空闲连接数
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // 连接最大存活时间30分钟
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接最大存活时间10分钟

	return db, nil
}
