-- 打印模板表（如果已存在，需要手动添加新字段）
CREATE TABLE IF NOT EXISTS `print_templates` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `corp_id` bigint NOT NULL DEFAULT 0,
  `name` varchar(100) NOT NULL COMMENT '模板名称',
  `menu_route` varchar(100) COMMENT '菜单路由',
  `button_key` varchar(100) COMMENT '按钮唯一标识',
  `button_name` varchar(50) COMMENT '按钮名称',
  `template` longtext COMMENT 'HTML模板内容',
  `excel_data` longblob COMMENT '原始Excel文件数据',
  `excel_filename` varchar(255) COMMENT '原始Excel文件名',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_print_templates_corp_id` (`corp_id`),
  INDEX `idx_print_templates_button_key` (`button_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='打印模板表';

-- 如果表已存在，执行以下ALTER语句添加新字段
-- ALTER TABLE `print_templates` ADD COLUMN `template` longtext COMMENT 'HTML模板内容' AFTER `button_name`;
-- ALTER TABLE `print_templates` ADD COLUMN `excel_data` longblob COMMENT '原始Excel文件数据' AFTER `template`;
-- ALTER TABLE `print_templates` ADD COLUMN `excel_filename` varchar(255) COMMENT '原始Excel文件名' AFTER `excel_data`;
-- ALTER TABLE `print_templates` DROP COLUMN IF EXISTS `preview`;

-- 为resources表添加page_resources字段
ALTER TABLE `resources` ADD COLUMN IF NOT EXISTS `page_resources` JSON COMMENT '内部页面资源(按钮等)' AFTER `sort_order`;

-- 更新送货管理菜单的page_resources，添加"预览送货单"按钮
UPDATE `resources`
SET `page_resources` = JSON_ARRAY(
  JSON_OBJECT('key', 'btn-preview-delivery-note', 'name', '预览送货单', 'type', 'button')
)
WHERE `resource_key` = 'production_delivery`;

-- 添加打印模板菜单资源
INSERT INTO `resources` (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`) VALUES
('menu', 'print_template', '打印模板', NULL, 'print_template', 'FileText', 20, 'business', 'active');