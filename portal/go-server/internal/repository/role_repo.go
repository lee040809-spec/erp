package repository

import (
	"database/sql"
	"portal-erp/internal/models"
)

type RoleRepository struct {
	db *sql.DB
}

func NewRoleRepository(db *sql.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// List 获取所有角色
func (r *RoleRepository) List() ([]models.Role, error) {
	query := `SELECT r.id, r.name, r.company_id, r.description, r.created_at, c.name as company_name
	          FROM roles r LEFT JOIN companies c ON r.company_id = c.id ORDER BY r.id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		var companyID sql.NullInt64
		var desc, companyName sql.NullString
		err := rows.Scan(&role.ID, &role.Name, &companyID, &desc, &role.CreatedAt, &companyName)
		if err != nil {
			return nil, err
		}
		if companyID.Valid {
			role.CompanyID = int(companyID.Int64)
		}
		if desc.Valid {
			role.Description = desc.String
		}
		if companyName.Valid {
			role.CompanyName = companyName.String
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// GetByID 根据ID获取角色
func (r *RoleRepository) GetByID(id int) (*models.Role, error) {
	var role models.Role
	var companyID sql.NullInt64
	var desc, companyName sql.NullString
	query := `SELECT r.id, r.name, r.company_id, r.description, r.created_at, c.name as company_name
	          FROM roles r LEFT JOIN companies c ON r.company_id = c.id WHERE r.id = ?`
	err := r.db.QueryRow(query, id).Scan(&role.ID, &role.Name, &companyID, &desc, &role.CreatedAt, &companyName)
	if err != nil {
		return nil, err
	}
	if companyID.Valid {
		role.CompanyID = int(companyID.Int64)
	}
	if desc.Valid {
		role.Description = desc.String
	}
	if companyName.Valid {
		role.CompanyName = companyName.String
	}
	return &role, nil
}

// Create 创建角色
func (r *RoleRepository) Create(role *models.Role) error {
	query := `INSERT INTO roles (name, company_id, description, created_at) VALUES (?, ?, ?, NOW())`
	result, err := r.db.Exec(query, role.Name, role.CompanyID, role.Description)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	role.ID = int(id)
	return nil
}

// Update 更新角色
func (r *RoleRepository) Update(role *models.Role) error {
	query := `UPDATE roles SET name = ?, company_id = ?, description = ? WHERE id = ?`
	_, err := r.db.Exec(query, role.Name, role.CompanyID, role.Description, role.ID)
	return err
}

// Delete 删除角色
func (r *RoleRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM roles WHERE id = ?", id)
	return err
}

// GetResourcesByRoleID 获取角色关联的资源
func (r *RoleRepository) GetResourcesByRoleID(roleID int) ([]models.Resource, error) {
	query := `SELECT r.id, r.name, r.type, r.path, r.parent_id, r.sort_order
	          FROM resources r
	          INNER JOIN role_resources rr ON r.id = rr.resource_id
	          WHERE rr.role_id = ?
	          ORDER BY r.sort_order`
	rows, err := r.db.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []models.Resource
	for rows.Next() {
		var res models.Resource
		var parentID sql.NullInt64
		err := rows.Scan(&res.ID, &res.Name, &res.Type, &res.Path, &parentID, &res.SortOrder)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			res.ParentID = int(parentID.Int64)
		}
		resources = append(resources, res)
	}
	return resources, nil
}

// UpdateRoleResources 更新角色的资源关联
func (r *RoleRepository) UpdateRoleResources(roleID int, resourceIDs []int) error {
	// 先删除旧的关联
	r.db.Exec("DELETE FROM role_resources WHERE role_id = ?", roleID)

	// 添加新的关联
	for _, resourceID := range resourceIDs {
		r.db.Exec("INSERT INTO role_resources (role_id, resource_id) VALUES (?, ?)", roleID, resourceID)
	}
	return nil
}