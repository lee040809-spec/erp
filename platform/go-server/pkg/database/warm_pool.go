package database

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

// Keep a small pool ready during local ERP use. Ping is read-only, and failures
// are retried on the next tick; business writes are never retried here.
func KeepReadConnectionsWarm(ctx context.Context, db *sql.DB, size int) {
	if size < 1 {
		return
	}
	if size > 3 {
		size = 3
	}
	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()
	for {
		warmContext, cancel := context.WithTimeout(ctx, 8*time.Second)
		connections := make([]*sql.Conn, size)
		var wait sync.WaitGroup
		for i := range connections {
			wait.Add(1)
			go func(i int) {
				defer wait.Done()
				conn, err := db.Conn(warmContext)
				if err != nil {
					return
				}
				connections[i] = conn
				_ = conn.PingContext(warmContext)
			}(i)
		}
		wait.Wait()
		for _, conn := range connections {
			if conn != nil {
				conn.Close()
			}
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
