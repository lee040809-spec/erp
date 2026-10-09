-- 初始化企业数据 (id=0, 名称:裕合森)
INSERT INTO `companies` (`id`, `name`, `short_name`, `status`) VALUES
(0, '裕合森', '裕合森', 'active')
ON DUPLICATE KEY UPDATE `name` = '裕合森';

-- 初始化角色数据
INSERT INTO `roles` (`corp_id`, `name`, `account_type`, `status`) VALUES
(0, '管理员', 'admin', 'active'),
(0, '普通用户', 'user', 'active');

-- 插入菜单资源数据 (业务平台)
INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`) VALUES
-- 一级菜单
('menu', 'dashboard', '工作看板', NULL, '/dashboard', 'LayoutDashboard', 1, 'business'),
('menu', 'orders', '订单管理', NULL, '/orders', 'ClipboardList', 2, 'business'),
('menu', 'overdue', '逾期订单', NULL, '/overdue', 'AlertCircle', 3, 'business'),
('menu', 'warning_orders', '告警订单', NULL, '/warning', 'AlertTriangle', 4, 'business'),
('menu', 'imminent_orders', '临期订单', NULL, '/imminent', 'Clock', 5, 'business'),
('menu', 'customers', '客户管理', NULL, '/customers', 'Users', 6, 'business'),
('menu', 'inventory', '库存管理', NULL, '/inventory', 'Package', 7, 'business'),
('menu', 'finance', '财务报表', NULL, '/finance', 'BarChart3', 8, 'business'),
('menu', 'advent_rules', '规则配置', NULL, '/advent-rules', 'Settings', 9, 'business');

-- 插入默认管理员用户 (密码: yhs@2026, 永久有效)
INSERT INTO `users` (`corp_id`, `role_type`, `username`, `password`, `email`, `name`, `nick_name`, `status`, `email_sent_success`, `expired_at`) VALUES
(0, 'admin', 'admin', '$2a$10$rRjhckU.KOYj5vhpeUa9Du.Lw9UyzD/yTJHbega8fa55rtu20CQEu', 'admin@example.com', '管理员', '管理员', 'active', false, NULL)
ON DUPLICATE KEY UPDATE `password` = '$2a$10$rRjhckU.KOYj5vhpeUa9Du.Lw9UyzD/yTJHbega8fa55rtu20CQEu';

-- 插入yhs用户 (密码: yhs@2026, 永久有效)
INSERT IGNORE INTO `users` (`corp_id`, `role_type`, `username`, `password`, `email`, `name`, `nick_name`, `status`, `email_sent_success`, `expired_at`) VALUES
(0, 'admin', 'yhs', '$2a$10$7HNdK.qrJLG4gwjpseL8QuiVQHW2QcXjzRvcZ7vUTQ32gfhjRINNS', 'yhs@example.com', 'YHS', 'YHS', 'active', false, NULL);

-- 为管理员角色分配所有菜单权限（管理员通过role_type=admin直接获取所有菜单，不需要权限记录）
-- 为普通用户角色分配菜单权限示例
INSERT IGNORE INTO `permissions` (`corp_id`, `role_id`, `resource_id`, `permission`)
SELECT 0, r.id, res.id, 'read'
FROM `roles` r, `resources` res
WHERE r.name = '普通用户' AND r.corp_id = 0;