-- 添加送货管理菜单资源
-- 执行此SQL后，需要为相应角色分配这些菜单的权限

INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`) VALUES
('menu', 'production_delivery', '送货管理', NULL, 'production_delivery', 'Truck', 15, 'business', 'active');