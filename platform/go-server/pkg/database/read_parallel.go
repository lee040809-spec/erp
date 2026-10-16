package database

import (
	"gorm.io/gorm"
	"sync"
)

// ReadParallel joins every task before returning. It is only for independent reads.
// Transaction-bound reads stay serial on their single connection.
func ReadParallel(db *gorm.DB, tasks ...func(*gorm.DB) error) error {
	if _, ok := db.Statement.ConnPool.(gorm.TxCommitter); ok {
		for _, task := range tasks {
			if err := task(db); err != nil {
				return err
			}
		}
		return nil
	}
	errors := make([]error, len(tasks))
	limit := make(chan struct{}, 3)
	var wait sync.WaitGroup
	for i, task := range tasks {
		wait.Add(1)
		go func(i int, task func(*gorm.DB) error) {
			defer wait.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			errors[i] = task(db.Session(&gorm.Session{}))
		}(i, task)
	}
	wait.Wait()
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
