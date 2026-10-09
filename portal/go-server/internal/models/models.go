package models

import (
	"database/sql"
	"time"
)

// Company 公司模型
type Company struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Status      int       `json:"status"` // 1:启用, 0:禁用
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User 用户模型
type User struct {
	ID            int            `json:"id"`
	Username      string         `json:"username"`
	PasswordHash  string         `json:"-"` // 不返回给前端
	Email         string         `json:"email"`
	CompanyID     int            `json:"company_id"`
	CompanyName   string         `json:"company_name"` // 关联查询
	RoleID        int            `json:"role_id"`
	RoleName      string         `json:"role_name"` // 关联查询
	LoginAttempts int            `json:"login_attempts"`
	LockUntil     *time.Time     `json:"lock_until"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// SuperUser 超级用户模型
type SuperUser struct {
	ID            int        `json:"id"`
	Username      string     `json:"username"`
	PasswordHash  string     `json:"-"`
	Email         string     `json:"email"`
	RoleID        int        `json:"role_id"`
	RoleName      string     `json:"role_name"`
	LoginAttempts int        `json:"login_attempts"`
	LockUntil     *time.Time `json:"lock_until"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Role 角色模型
type Role struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	CompanyID   int       `json:"company_id"`
	CompanyName string    `json:"company_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Resource 资源模型
type Resource struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // menu, button, api
	Path      string `json:"path"`
	ParentID  int    `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
}

// RoleResource 角色-资源关联
type RoleResource struct {
	RoleID     int `json:"role_id"`
	ResourceID int `json:"resource_id"`
}

// NullTime 处理可能为NULL的时间字段
type NullTime struct {
	Time  time.Time
	Valid bool
}

func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Valid = false
		return nil
	}
	nt.Valid = true
	switch v := value.(type) {
	case time.Time:
		nt.Time = v
	case []byte:
		nt.Time, _ = time.Parse("2006-01-02 15:04:05", string(v))
	}
	return nil
}

func (nt NullTime) Value() (interface{}, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

// NullString 处理可能为NULL的字符串字段
type NullString struct {
	String string
	Valid  bool
}

func (ns *NullString) Scan(value interface{}) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}
	ns.String = s.String
	ns.Valid = s.Valid
	return nil
}