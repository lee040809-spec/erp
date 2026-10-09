-- 添加生产相关菜单资源
-- 执行此SQL后，需要为相应角色分配这些菜单的权限

INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`) VALUES
('menu', 'production_dashboard', '生产工作看板', NULL, 'production_dashboard', 'LayoutDashboard', 10, 'business', 'active'),
('menu', 'production_orders', '生产订单管理', NULL, 'production_orders', 'ClipboardList', 11, 'business', 'active'),
('menu', 'production_overdue', '生产逾期订单', NULL, 'production_overdue', 'AlertCircle', 12, 'business', 'active'),
('menu', 'production_warning', '生产告警订单', NULL, 'production_warning', 'AlertTriangle', 13, 'business', 'active'),
('menu', 'production_imminent', '生产临期订单', NULL, 'production_imminent', 'Clock', 14, 'business', 'active');