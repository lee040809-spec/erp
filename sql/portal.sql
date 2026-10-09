-- Portal 数据库初始化脚本
-- 数据库名: portal

-- 公司表
CREATE TABLE IF NOT EXISTS companies (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(255),
    status TINYINT DEFAULT 1 COMMENT '1:启用, 0:禁用',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 角色表
CREATE TABLE IF NOT EXISTS roles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    company_id INT DEFAULT NULL,
    description VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    company_id INT DEFAULT NULL,
    role_id INT DEFAULT 2,
    login_attempts INT DEFAULT 0,
    lock_until DATETIME NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE SET NULL,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 超级用户表
CREATE TABLE IF NOT EXISTS super_users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    role_id INT DEFAULT 1,
    login_attempts INT DEFAULT 0,
    lock_until DATETIME NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 资源表
CREATE TABLE IF NOT EXISTS resources (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type ENUM('menu', 'button', 'api') NOT NULL,
    path VARCHAR(255) NOT NULL,
    parent_id INT DEFAULT 0,
    sort_order INT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 角色-资源关联表
CREATE TABLE IF NOT EXISTS role_resources (
    id INT AUTO_INCREMENT PRIMARY KEY,
    role_id INT NOT NULL,
    resource_id INT NOT NULL,
    UNIQUE KEY unique_role_resource (role_id, resource_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 初始化角色
INSERT INTO roles (id, name, description) VALUES
(1, '管理员', '系统管理员，拥有所有权限'),
(2, '普通用户', '普通用户，基础权限')
ON DUPLICATE KEY UPDATE name=VALUES(name);

-- 初始化资源（菜单）
INSERT INTO resources (id, name, type, path, parent_id, sort_order) VALUES
(1, '仪表盘', 'menu', '/portal/dashboard', 0, 1),
(2, '用户管理', 'menu', '/portal/users', 0, 2),
(3, '角色管理', 'menu', '/portal/roles', 0, 3),
(4, '资源管理', 'menu', '/portal/resources', 0, 4),
(10, '公司管理', 'menu', '/portal/companies', 0, 5),
(5, '查看用户', 'button', '/portal/users/view', 2, 1),
(6, '创建用户', 'button', '/portal/users/create', 2, 2),
(7, '编辑用户', 'button', '/portal/users/edit', 2, 3),
(8, '删除用户', 'button', '/portal/users/delete', 2, 4),
(9, '重置密码', 'button', '/portal/users/reset-password', 2, 5)
ON DUPLICATE KEY UPDATE name=VALUES(name);

-- 管理员角色拥有所有资源
INSERT IGNORE INTO role_resources (role_id, resource_id)
SELECT 1, id FROM resources;

-- 普通用户角色拥有仪表盘权限
INSERT IGNORE INTO role_resources (role_id, resource_id) VALUES
(2, 1);