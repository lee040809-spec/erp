package utils

import (
	"errors"
	"time"
)

var processDateZone = time.FixedZone("Asia/Shanghai", 8*3600)

// ProcessDay compares calendar dates in Beijing, including legacy timestamps.
func ProcessDay(t time.Time) time.Time {
	t = t.In(processDateZone)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, processDateZone)
}

func ValidateProcessPlan(start, end *time.Time) error {
	// An unknown duration may have only a start or end date; compare known dates only.
	if start != nil && end != nil && ProcessDay(*end).Before(ProcessDay(*start)) {
		return errors.New("计划结束日期不能早于计划开始日期")
	}
	return nil
}
