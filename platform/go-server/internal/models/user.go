package models

import (
	"time"
)

// UserRoleType 用户角色类型
type UserRoleType string

const (
	UserRoleTypeAdmin UserRoleType = "admin" // 管理员
	UserRoleTypeUser  UserRoleType = "user"  // 普通用户
)

// User 用户表
type User struct {
	ID               int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID           int64         `gorm:"column:corp_id;uniqueIndex:idx_corp_username;not null" json:"corp_id"`
	RoleID           *int64        `gorm:"column:role_id;index" json:"role_id,omitempty"`
	RoleType         *UserRoleType `gorm:"column:role_type;type:varchar(20)" json:"role_type,omitempty"`
	Username         string        `gorm:"column:username;size:50;uniqueIndex:idx_corp_username;not null" json:"username"` // 账号
	Password         string        `gorm:"column:password;size:255;not null" json:"-"`                                     // 密码
	Email            string        `gorm:"column:email;size:100" json:"email"`                                             // 邮箱
	EmailSentSuccess *bool         `gorm:"column:email_sent_success;default:false" json:"email_sent_success,omitempty"`    // 邮箱发送成功标识
	Name             string        `gorm:"column:name;size:50" json:"name"`
	NickName         *string       `gorm:"column:nick_name;size:50" json:"nick_name,omitempty"`
	Phone            string        `gorm:"column:phone;size:20" json:"phone,omitempty"`
	Status           string        `gorm:"column:status;size:20;default:'active'" json:"status"`
	ExpiredAt        *time.Time    `gorm:"column:expired_at" json:"expired_at,omitempty"` // 账号有效期，null表示永久有效
	CreatedAt        *time.Time    `gorm:"column:created_at;autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt        *time.Time    `gorm:"column:updated_at;autoUpdateTime" json:"updated_at,omitempty"`
	Role             *Role         `gorm:"foreignKey:RoleID" json:"role,omitempty"` // 关联角色
}

func (User) TableName() string { return "users" }

// Permission 权限表
type Permission struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID     int64     `gorm:"column:corp_id;index;not null" json:"corp_id"`
	RoleID     int64     `gorm:"column:role_id;index;not null" json:"role_id"`
	ResourceID int64     `gorm:"column:resource_id;index;not null" json:"resource_id"`
	Permission string    `gorm:"column:permission;size:20;default:'read'" json:"permission"` // read, write, admin
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Permission) TableName() string { return "permissions" }

// ResourceType 资源类型
type ResourceType string

const (
	ResourceTypeMenu   ResourceType = "menu"
	ResourceTypeAPI    ResourceType = "api"
	ResourceTypeButton ResourceType = "button"
)

// PlatformType 平台类型
type PlatformType string

const (
	PlatformTypeBusiness PlatformType = "business" // 业务平台
	PlatformTypeManage   PlatformType = "manage"   // 管理平台
)

// Resource 资源表
type Resource struct {
	ID            int64        `gorm:"primaryKey;autoIncrement" json:"id"`
	ResourceType  string       `gorm:"column:resource_type;size:20;not null" json:"resource_type"` // menu, api, button
	ResourceKey   string       `gorm:"column:resource_key;size:100;uniqueIndex;not null" json:"resource_key"`
	Name          string       `gorm:"column:name;size:100;not null" json:"name"`
	ParentID      *int64       `gorm:"column:parent_id;index" json:"parent_id,omitempty"`
	Path          string       `gorm:"column:path;size:200" json:"path,omitempty"`
	Icon          string       `gorm:"column:icon;size:50" json:"icon,omitempty"`
	SortOrder     int          `gorm:"column:sort_order;default:0" json:"sort_order"`
	PageResources string       `gorm:"column:page_resources;type:json" json:"page_resources,omitempty"`      // 内部页面资源(按钮等)
	PlatformType  PlatformType `gorm:"column:platform_type;size:20;default:'business'" json:"platform_type"` // business, manage
	Status        string       `gorm:"column:status;size:20;default:'active'" json:"status"`
	CreatedAt     time.Time    `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Children      []Resource   `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (Resource) TableName() string { return "resources" }
