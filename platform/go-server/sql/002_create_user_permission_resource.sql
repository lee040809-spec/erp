-- 企业表
CREATE TABLE IF NOT EXISTS `companies` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '企业名称',
  `short_name` varchar(50) DEFAULT NULL COMMENT '企业简称',
  `address` varchar(200) DEFAULT NULL COMMENT '地址',
  `phone` varchar(20) DEFAULT NULL COMMENT '电话',
  `status` varchar(20) DEFAULT 'active' COMMENT '状态: active/inactive',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='企业表';

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `corp_id` bigint NOT NULL DEFAULT 0 COMMENT '企业ID',
  `role_id` bigint DEFAULT NULL COMMENT '角色ID',
  `role_type` varchar(20) DEFAULT NULL COMMENT '角色类型: admin/user',
  `username` varchar(50) NOT NULL COMMENT '账号',
  `password` varchar(255) NOT NULL COMMENT '密码',
  `email` varchar(100) DEFAULT NULL COMMENT '邮箱',
  `email_sent_success` tinyint(1) DEFAULT 0 COMMENT '邮箱发送成功标识',
  `name` varchar(50) DEFAULT NULL COMMENT '姓名',
  `nick_name` varchar(50) DEFAULT NULL COMMENT '别名',
  `phone` varchar(20) DEFAULT NULL COMMENT '电话',
  `status` varchar(20) DEFAULT 'active' COMMENT '状态: active/inactive',
  `expired_at` datetime DEFAULT NULL COMMENT '账号有效期，NULL表示永久有效',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_corp_username` (`corp_id`, `username`),
  KEY `idx_corp_id` (`corp_id`),
  KEY `idx_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 角色表
CREATE TABLE IF NOT EXISTS `roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `corp_id` bigint NOT NULL DEFAULT 0 COMMENT '企业ID',
  `name` varchar(50) NOT NULL COMMENT '角色名称',
  `account_type` varchar(20) NOT NULL COMMENT '账户类型: admin/user',
  `status` varchar(20) DEFAULT 'active' COMMENT '状态: active/inactive',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_corp_id` (`corp_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

-- 权限表
CREATE TABLE IF NOT EXISTS `permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `corp_id` bigint NOT NULL DEFAULT 0 COMMENT '企业ID',
  `role_id` bigint NOT NULL COMMENT '角色ID',
  `resource_id` bigint NOT NULL COMMENT '资源ID',
  `permission` varchar(20) DEFAULT 'read' COMMENT '权限类型: read/write/admin',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_corp_id` (`corp_id`),
  KEY `idx_role_id` (`role_id`),
  KEY `idx_resource_id` (`resource_id`),
  UNIQUE KEY `idx_role_resource` (`role_id`, `resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='权限表';

-- 资源表
CREATE TABLE IF NOT EXISTS `resources` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `resource_type` varchar(20) NOT NULL COMMENT '资源类型: menu/api/button',
  `resource_key` varchar(100) NOT NULL COMMENT '资源唯一标识',
  `name` varchar(100) NOT NULL COMMENT '资源名称',
  `parent_id` bigint DEFAULT NULL COMMENT '父资源ID',
  `path` varchar(200) DEFAULT NULL COMMENT '路由路径',
  `icon` varchar(50) DEFAULT NULL COMMENT '图标',
  `sort_order` int DEFAULT 0 COMMENT '排序',
  `platform_type` varchar(20) DEFAULT 'business' COMMENT '平台类型: business/manage',
  `status` varchar(20) DEFAULT 'active' COMMENT '状态',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_resource_key` (`resource_key`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='资源表';