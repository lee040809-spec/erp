package database

import (
	"log"
	"net"
	"os"
	"strconv"
	"time"

	driver "github.com/go-sql-driver/mysql"
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

func driverConfig(cfg *Config) *driver.Config {
	c := driver.NewConfig()
	c.User, c.Passwd, c.DBName = cfg.User, cfg.Password, cfg.Database
	c.Net, c.Addr = "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	c.ParseTime, c.Loc = true, time.Local
	c.Params = map[string]string{"charset": "utf8mb4"}
	c.Timeout, c.ReadTimeout, c.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	// Driver escaping preserves bound parameters and removes prepare/close round trips.
	c.InterpolateParams = true
	return c
}

func Connect(cfg *Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(driverConfig(cfg).FormatDSN()), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			LogLevel: logger.Warn, SlowThreshold: time.Second,
			ParameterizedQueries: true, Colorful: false,
		}),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	// Windows driver cannot probe stale sockets; retire idle connections before
	// intermediate network equipment can silently discard them.
	sqlDB.SetConnMaxIdleTime(30 * time.Second)
	return db, nil
}
