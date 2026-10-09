package repository

import (
	"database/sql"
	"portal-erp/internal/models"
)

type ResourceRepository struct {
	db *sql.DB
}

func NewResourceRepository(db *sql.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

// List 获取所有资源
func (r *ResourceRepository) List() ([]models.Resource, error) {
	query := `SELECT id, name, type, path, parent_id, sort_order FROM resources ORDER BY sort_order, id`
	rows, err := r.db.Query(query)
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

// GetByID 根据ID获取资源
func (r *ResourceRepository) GetByID(id int) (*models.Resource, error) {
	var res models.Resource
	var parentID sql.NullInt64
	query := `SELECT id, name, type, path, parent_id, sort_order FROM resources WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(&res.ID, &res.Name, &res.Type, &res.Path, &parentID, &res.SortOrder)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		res.ParentID = int(parentID.Int64)
	}
	return &res, nil
}

// GetByPath 根据路径获取资源（用于权限验证）
func (r *ResourceRepository) GetByPath(path string) (*models.Resource, error) {
	var res models.Resource
	var parentID sql.NullInt64
	query := `SELECT id, name, type, path, parent_id, sort_order FROM resources WHERE path = ?`
	err := r.db.QueryRow(query, path).Scan(&res.ID, &res.Name, &res.Type, &res.Path, &parentID, &res.SortOrder)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		res.ParentID = int(parentID.Int64)
	}
	return &res, nil
}

// Create 创建资源
func (r *ResourceRepository) Create(res *models.Resource) error {
	query := `INSERT INTO resources (name, type, path, parent_id, sort_order) VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.Exec(query, res.Name, res.Type, res.Path, res.ParentID, res.SortOrder)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	res.ID = int(id)
	return nil
}

// Update 更新资源
func (r *ResourceRepository) Update(res *models.Resource) error {
	query := `UPDATE resources SET name = ?, type = ?, path = ?, parent_id = ?, sort_order = ? WHERE id = ?`
	_, err := r.db.Exec(query, res.Name, res.Type, res.Path, res.ParentID, res.SortOrder, res.ID)
	return err
}

// Delete 删除资源
func (r *ResourceRepository) Delete(id int) error {
	// 先删除角色关联
	r.db.Exec("DELETE FROM role_resources WHERE resource_id = ?", id)
	_, err := r.db.Exec("DELETE FROM resources WHERE id = ?", id)
	return err
}

// GetMenusByRoleID 获取角色可见的菜单资源
func (r *ResourceRepository) GetMenusByRoleID(roleID int) ([]models.Resource, error) {
	query := `SELECT r.id, r.name, r.type, r.path, r.parent_id, r.sort_order
	          FROM resources r
	          INNER JOIN role_resources rr ON r.id = rr.resource_id
	          WHERE rr.role_id = ? AND r.type = 'menu'
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