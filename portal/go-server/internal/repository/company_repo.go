package repository

import (
	"database/sql"
	"portal-erp/internal/models"
)

type CompanyRepository struct {
	db *sql.DB
}

func NewCompanyRepository(db *sql.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

// List 获取所有公司
func (r *CompanyRepository) List() ([]models.Company, error) {
	query := `SELECT id, name, code, description, status, created_at, updated_at FROM companies ORDER BY id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []models.Company
	for rows.Next() {
		var c models.Company
		var desc sql.NullString
		err := rows.Scan(&c.ID, &c.Name, &c.Code, &desc, &c.Status, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if desc.Valid {
			c.Description = desc.String
		}
		companies = append(companies, c)
	}
	return companies, nil
}

// GetByID 根据ID获取公司
func (r *CompanyRepository) GetByID(id int) (*models.Company, error) {
	var c models.Company
	var desc sql.NullString
	query := `SELECT id, name, code, description, status, created_at, updated_at FROM companies WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(&c.ID, &c.Name, &c.Code, &desc, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		c.Description = desc.String
	}
	return &c, nil
}

// Create 创建公司
func (r *CompanyRepository) Create(c *models.Company) error {
	query := `INSERT INTO companies (name, code, description, status) VALUES (?, ?, ?, ?)`
	result, err := r.db.Exec(query, c.Name, c.Code, c.Description, c.Status)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	c.ID = int(id)
	return nil
}

// Update 更新公司
func (r *CompanyRepository) Update(c *models.Company) error {
	query := `UPDATE companies SET name = ?, code = ?, description = ?, status = ? WHERE id = ?`
	_, err := r.db.Exec(query, c.Name, c.Code, c.Description, c.Status, c.ID)
	return err
}

// Delete 删除公司
func (r *CompanyRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM companies WHERE id = ?", id)
	return err
}