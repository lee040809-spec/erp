-- 添加对账管理菜单
INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`) VALUES
('menu', 'reconciliation', '对账管理', NULL, '/reconciliation', 'FileText', 7, 'business', 'active');

-- 更新后续菜单的排序
UPDATE `resources` SET `sort_order` = 8 WHERE `resource_key` = 'inventory';
UPDATE `resources` SET `sort_order` = 9 WHERE `resource_key` = 'finance';
UPDATE `resources` SET `sort_order` = 10 WHERE `resource_key` = 'advent_rules';

-- 添加对账管理按钮资源（预览和配置共用同一个按钮）
SET @reconciliation_menu_id = (SELECT id FROM resources WHERE resource_key = 'reconciliation' AND resource_type = 'menu');
INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`) VALUES
('button', 'btn-config-reconciliation', '对账单', @reconciliation_menu_id, NULL, NULL, 1, 'business', 'active');