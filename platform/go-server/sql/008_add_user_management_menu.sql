-- 添加用户管理、角色管理、资源管理菜单资源
INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`) VALUES
('menu', 'user_management', '用户管理', NULL, 'user_management', 'Users', 20, 'business', 'active'),
('menu', 'role_management', '角色管理', NULL, 'role_management', 'Shield', 21, 'business', 'active'),
('menu', 'resource_management', '资源管理', NULL, 'resource_management', 'Layers', 22, 'business', 'active');