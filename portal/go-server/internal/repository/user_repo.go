package repository

import (
	"database/sql"
	"portal-erp/internal/models"
	"time"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByUsername 根据用户名查询用户
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	var lockUntil sql.NullTime
	var companyID sql.NullInt64
	query := `SELECT id, username, password_hash, email, company_id, role_id, login_attempts, lock_until, created_at, updated_at
	          FROM users WHERE username = ?`
	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email, &companyID, &user.RoleID,
		&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lockUntil.Valid {
		user.LockUntil = &lockUntil.Time
	}
	if companyID.Valid {
		user.CompanyID = int(companyID.Int64)
	}

	// 获取角色名称
	r.db.QueryRow("SELECT name FROM roles WHERE id = ?", user.RoleID).Scan(&user.RoleName)

	// 获取公司名称
	if user.CompanyID > 0 {
		r.db.QueryRow("SELECT name FROM companies WHERE id = ?", user.CompanyID).Scan(&user.CompanyName)
	}

	return &user, nil
}

// GetByID 根据ID查询用户
func (r *UserRepository) GetByID(id int) (*models.User, error) {
	var user models.User
	var lockUntil sql.NullTime
	var companyID sql.NullInt64
	query := `SELECT id, username, password_hash, email, company_id, role_id, login_attempts, lock_until, created_at, updated_at
	          FROM users WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email, &companyID, &user.RoleID,
		&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lockUntil.Valid {
		user.LockUntil = &lockUntil.Time
	}
	if companyID.Valid {
		user.CompanyID = int(companyID.Int64)
	}

	r.db.QueryRow("SELECT name FROM roles WHERE id = ?", user.RoleID).Scan(&user.RoleName)
	if user.CompanyID > 0 {
		r.db.QueryRow("SELECT name FROM companies WHERE id = ?", user.CompanyID).Scan(&user.CompanyName)
	}

	return &user, nil
}

// List 获取所有用户列表
func (r *UserRepository) List() ([]models.User, error) {
	query := `SELECT u.id, u.username, u.email, u.company_id, u.role_id, u.login_attempts, u.lock_until, u.created_at, u.updated_at,
	          r.name as role_name, c.name as company_name
	          FROM users u
	          LEFT JOIN roles r ON u.role_id = r.id
	          LEFT JOIN companies c ON u.company_id = c.id
	          ORDER BY u.id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var lockUntil sql.NullTime
		var companyID sql.NullInt64
		var roleName, companyName sql.NullString
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &companyID, &user.RoleID,
			&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt, &roleName, &companyName,
		)
		if err != nil {
			return nil, err
		}
		if lockUntil.Valid {
			user.LockUntil = &lockUntil.Time
		}
		if companyID.Valid {
			user.CompanyID = int(companyID.Int64)
		}
		if roleName.Valid {
			user.RoleName = roleName.String
		}
		if companyName.Valid {
			user.CompanyName = companyName.String
		}
		users = append(users, user)
	}
	return users, nil
}

// Create 创建用户
func (r *UserRepository) Create(user *models.User) error {
	query := `INSERT INTO users (username, password_hash, email, company_id, role_id, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := r.db.Exec(query, user.Username, user.PasswordHash, user.Email, user.CompanyID, user.RoleID)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = int(id)
	return nil
}

// Update 更新用户基本信息
func (r *UserRepository) Update(user *models.User) error {
	query := `UPDATE users SET email = ?, company_id = ?, role_id = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, user.Email, user.CompanyID, user.RoleID, user.ID)
	return err
}

// UpdatePassword 更新密码
func (r *UserRepository) UpdatePassword(id int, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, passwordHash, id)
	return err
}

// UpdateLoginAttempts 更新登录尝试次数
func (r *UserRepository) UpdateLoginAttempts(id int, attempts int) error {
	query := `UPDATE users SET login_attempts = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, attempts, id)
	return err
}

// LockUser 锁定用户
func (r *UserRepository) LockUser(id int, lockUntil time.Time) error {
	query := `UPDATE users SET lock_until = ?, login_attempts = 0, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, lockUntil, id)
	return err
}

// UnlockUser 解锁用户
func (r *UserRepository) UnlockUser(id int) error {
	query := `UPDATE users SET lock_until = NULL, login_attempts = 0, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// Delete 删除用户
func (r *UserRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}