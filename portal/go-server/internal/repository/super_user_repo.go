package repository

import (
	"database/sql"
	"portal-erp/internal/models"
	"time"
)

type SuperUserRepository struct {
	db *sql.DB
}

func NewSuperUserRepository(db *sql.DB) *SuperUserRepository {
	return &SuperUserRepository{db: db}
}

// GetByUsername 根据用户名查询超级用户
func (r *SuperUserRepository) GetByUsername(username string) (*models.SuperUser, error) {
	var user models.SuperUser
	var lockUntil sql.NullTime
	query := `SELECT id, username, password_hash, email, role_id, login_attempts, lock_until, created_at, updated_at
	          FROM super_users WHERE username = ?`
	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.RoleID,
		&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lockUntil.Valid {
		user.LockUntil = &lockUntil.Time
	}

	r.db.QueryRow("SELECT name FROM roles WHERE id = ?", user.RoleID).Scan(&user.RoleName)
	return &user, nil
}

// GetByID 根据ID查询超级用户
func (r *SuperUserRepository) GetByID(id int) (*models.SuperUser, error) {
	var user models.SuperUser
	var lockUntil sql.NullTime
	query := `SELECT id, username, password_hash, email, role_id, login_attempts, lock_until, created_at, updated_at
	          FROM super_users WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.RoleID,
		&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lockUntil.Valid {
		user.LockUntil = &lockUntil.Time
	}

	r.db.QueryRow("SELECT name FROM roles WHERE id = ?", user.RoleID).Scan(&user.RoleName)
	return &user, nil
}

// List 获取所有超级用户
func (r *SuperUserRepository) List() ([]models.SuperUser, error) {
	query := `SELECT su.id, su.username, su.email, su.role_id, su.login_attempts, su.lock_until, su.created_at, su.updated_at, r.name as role_name
	          FROM super_users su LEFT JOIN roles r ON su.role_id = r.id ORDER BY su.id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.SuperUser
	for rows.Next() {
		var user models.SuperUser
		var lockUntil sql.NullTime
		var roleName sql.NullString
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.RoleID,
			&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt, &roleName,
		)
		if err != nil {
			return nil, err
		}
		if lockUntil.Valid {
			user.LockUntil = &lockUntil.Time
		}
		if roleName.Valid {
			user.RoleName = roleName.String
		}
		users = append(users, user)
	}
	return users, nil
}

// Create 创建超级用户
func (r *SuperUserRepository) Create(user *models.SuperUser) error {
	query := `INSERT INTO super_users (username, password_hash, email, role_id, created_at, updated_at)
	          VALUES (?, ?, ?, ?, NOW(), NOW())`
	result, err := r.db.Exec(query, user.Username, user.PasswordHash, user.Email, user.RoleID)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = int(id)
	return nil
}

// Update 更新超级用户
func (r *SuperUserRepository) Update(user *models.SuperUser) error {
	query := `UPDATE super_users SET email = ?, role_id = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, user.Email, user.RoleID, user.ID)
	return err
}

// UpdatePassword 更新密码
func (r *SuperUserRepository) UpdatePassword(id int, passwordHash string) error {
	query := `UPDATE super_users SET password_hash = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, passwordHash, id)
	return err
}

// UpdateLoginAttempts 更新登录尝试次数
func (r *SuperUserRepository) UpdateLoginAttempts(id int, attempts int) error {
	query := `UPDATE super_users SET login_attempts = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, attempts, id)
	return err
}

// LockUser 锁定超级用户
func (r *SuperUserRepository) LockUser(id int, lockUntil time.Time) error {
	query := `UPDATE super_users SET lock_until = ?, login_attempts = 0, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, lockUntil, id)
	return err
}

// UnlockUser 解锁超级用户
func (r *SuperUserRepository) UnlockUser(id int) error {
	query := `UPDATE super_users SET lock_until = NULL, login_attempts = 0, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// Delete 删除超级用户
func (r *SuperUserRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM super_users WHERE id = ?", id)
	return err
}