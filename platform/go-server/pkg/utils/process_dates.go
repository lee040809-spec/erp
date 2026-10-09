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
	if (start == nil) != (end == nil) {
		return errors.New("计划开始和计划结束日期须一起填写或一起清空")
	}
	if start != nil && ProcessDay(*end).Before(ProcessDay(*start)) {
		return errors.New("计划结束日期不能早于计划开始日期")
	}
	return nil
}
