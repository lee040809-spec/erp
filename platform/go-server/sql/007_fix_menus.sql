-- 修复：重新插入原有菜单资源（如果被删除）
INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`) VALUES
('menu', 'dashboard', '工作看板', NULL, '/dashboard', 'LayoutDashboard', 1, 'business', 'active'),
('menu', 'orders', '订单管理', NULL, '/orders', 'ClipboardList', 2, 'business', 'active'),
('menu', 'overdue', '逾期订单', NULL, '/overdue', 'AlertCircle', 3, 'business', 'active'),
('menu', 'warning_orders', '告警订单', NULL, '/warning', 'AlertTriangle', 4, 'business', 'active'),
('menu', 'imminent_orders', '临期订单', NULL, '/imminent', 'Clock', 5, 'business', 'active'),
('menu', 'customers', '客户管理', NULL, '/customers', 'Users', 6, 'business', 'active'),
('menu', 'inventory', '库存管理', NULL, '/inventory', 'Package', 7, 'business', 'active'),
('menu', 'finance', '财务报表', NULL, '/finance', 'BarChart3', 8, 'business', 'active'),
('menu', 'advent_rules', '规则配置', NULL, '/advent-rules', 'Settings', 9, 'business', 'active')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `status` = 'active';

-- 查看所有菜单
SELECT id, resource_key, name, sort_order FROM resources WHERE platform_type = 'business' ORDER BY sort_order;