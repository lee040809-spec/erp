package database

import (
	"database/sql"
	"errors"
	"gorm.io/gorm"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadParallelBoundedAndJoined(t *testing.T) {
	db := &gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{}}
	var active, peak, done atomic.Int32
	want := errors.New("first query failure")
	tasks := make([]func(*gorm.DB) error, 8)
	for i := range tasks {
		n := i
		tasks[i] = func(*gorm.DB) error {
			now := active.Add(1)
			for {
				old := peak.Load()
				if now <= old || peak.CompareAndSwap(old, now) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			active.Add(-1)
			done.Add(1)
			if n == 0 {
				return want
			}
			return nil
		}
	}
	if err := ReadParallel(db, tasks...); !errors.Is(err, want) {
		t.Fatal("original error not preserved")
	}
	if peak.Load() > 3 || peak.Load() < 2 || done.Load() != 8 || active.Load() != 0 {
		t.Fatalf("peak=%d done=%d active=%d", peak.Load(), done.Load(), active.Load())
	}
}
func TestReadParallelTransactionsRemainSerial(t *testing.T) {
	db := &gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{ConnPool: &sql.Tx{}}}
	calls := 0
	if err := ReadParallel(db, func(got *gorm.DB) error {
		if got != db {
			t.Fatal("transaction replaced")
		}
		calls++
		return nil
	}, func(*gorm.DB) error {
		if calls != 1 {
			t.Fatal("transaction task order changed")
		}
		calls++
		return nil
	}); err != nil || calls != 2 {
		t.Fatal("serial reads failed")
	}
}
