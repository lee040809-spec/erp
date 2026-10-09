package utils

import (
	"database/sql"
	"time"
)

// FormatDate 格式化日期为 YYYY-MM-DD
func FormatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// ParseDate 解析日期字符串
func ParseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ParseDateOrNil 解析日期，空字符串返回 nil
func ParseDateOrNil(s string) interface{} {
	if s == "" || s == "-" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return t
}

// DateToNullTime 将日期字符串转换为 sql.NullTime
func DateToNullTime(s string) sql.NullTime {
	if s == "" || s == "-" {
		return sql.NullTime{Valid: false}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}
