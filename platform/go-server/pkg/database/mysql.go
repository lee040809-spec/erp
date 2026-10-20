package database

import (
	"context"
	"database/sql"
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
	Host        string
	User        string
	Password    string
	Database    string
	Port        int
	ReadTimeout time.Duration
}

func driverConfig(cfg *Config) *driver.Config {
	c := driver.NewConfig()
	c.User, c.Passwd, c.DBName = cfg.User, cfg.Password, cfg.Database
	c.Net, c.Addr = "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	c.ParseTime, c.Loc = true, time.Local
	c.Params = map[string]string{"charset": "utf8mb4"}
	c.Timeout, c.ReadTimeout, c.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	if cfg.ReadTimeout > 0 {
		c.ReadTimeout = cfg.ReadTimeout
	}
	// Driver escaping preserves bound parameters and removes prepare/close round trips.
	c.InterpolateParams = true
	return c
}

func Connect(cfg *Config) (*gorm.DB, error) {
	return connectContext(context.Background(), cfg)
}

// GORM's version query and automatic Ping otherwise ignore the startup deadline.
type startupPool struct {
	*sql.DB
	ctx context.Context
}

func (p *startupPool) QueryRowContext(_ context.Context, query string, args ...interface{}) *sql.Row {
	return p.DB.QueryRowContext(p.ctx, query, args...)
}
func (p *startupPool) Ping() error                 { return p.DB.PingContext(p.ctx) }
func (p *startupPool) GetDBConn() (*sql.DB, error) { return p.DB, nil }

func connectContext(ctx context.Context, cfg *Config) (*gorm.DB, error) {
	connector, err := driver.NewConnector(driverConfig(cfg))
	if err != nil {
		return nil, err
	}
	pool := sql.OpenDB(connector)
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: &startupPool{DB: pool, ctx: ctx}}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			LogLevel: logger.Warn, SlowThreshold: time.Second,
			ParameterizedQueries: true, Colorful: false,
		}),
	})
	if err != nil {
		_ = pool.Close()
		return nil, err
	}
	// Later requests must use their own contexts, not the completed startup context.
	db.ConnPool, db.Statement.ConnPool = pool, pool
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
