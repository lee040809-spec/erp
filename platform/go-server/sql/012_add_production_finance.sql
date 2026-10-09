-- 生产产值与成本核算基础表
-- 设计原则：只新增表和字段，不删除旧字段；旧数据不会自动转为精确产值。

ALTER TABLE `order_items`
  ADD COLUMN `supply_mode` VARCHAR(32) NOT NULL DEFAULT 'self_purchased' COMMENT 'self_purchased:自购材料 customer_supplied:客户来料' AFTER `other_cost`,
  ADD COLUMN `currency` CHAR(3) NOT NULL DEFAULT 'CNY' AFTER `supply_mode`,
  ADD COLUMN `tax_inclusive` TINYINT(1) NOT NULL DEFAULT 0 AFTER `currency`;

CREATE TABLE IF NOT EXISTS `production_completions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `corp_id` BIGINT NOT NULL,
  `order_id` BIGINT NOT NULL,
  `order_item_id` BIGINT NOT NULL,
  `completion_no` VARCHAR(64) NOT NULL,
  `business_date` DATE NOT NULL,
  `qualified_quantity` DECIMAL(12,3) NOT NULL,
  `unit_price_snapshot` DECIMAL(15,4) NOT NULL,
  `output_value` DECIMAL(18,2) NOT NULL,
  `supply_mode_snapshot` VARCHAR(32) NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `tax_inclusive` TINYINT(1) NOT NULL DEFAULT 0,
  `status` VARCHAR(20) NOT NULL DEFAULT 'confirmed',
  `notes` VARCHAR(500) DEFAULT NULL,
  `created_by` BIGINT NOT NULL,
  `confirmed_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `reversed_by` BIGINT DEFAULT NULL,
  `reversed_at` DATETIME(3) DEFAULT NULL,
  `reversal_reason` VARCHAR(500) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_completion_corp_no` (`corp_id`, `completion_no`),
  KEY `idx_completion_corp_month` (`corp_id`, `business_date`),
  KEY `idx_completion_item_status` (`corp_id`, `order_item_id`, `status`),
  KEY `idx_completion_order` (`corp_id`, `order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='最终合格完工批次及产值快照';

CREATE TABLE IF NOT EXISTS `cost_entries` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `corp_id` BIGINT NOT NULL,
  `cost_type` VARCHAR(32) NOT NULL COMMENT 'material/outsourcing/tool/fixture/labor/manufacturing/other_direct/period',
  `amount` DECIMAL(18,2) NOT NULL,
  `tax_amount` DECIMAL(18,2) NOT NULL DEFAULT 0.00,
  `net_amount` DECIMAL(18,2) NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `business_date` DATE NOT NULL,
  `accounting_month` CHAR(7) NOT NULL,
  `order_id` BIGINT DEFAULT NULL,
  `order_item_id` BIGINT DEFAULT NULL,
  `production_completion_id` BIGINT DEFAULT NULL,
  `supplier_name` VARCHAR(255) DEFAULT NULL,
  `source_type` VARCHAR(50) DEFAULT NULL,
  `source_id` VARCHAR(100) DEFAULT NULL,
  `status` VARCHAR(20) NOT NULL DEFAULT 'confirmed',
  `notes` VARCHAR(500) DEFAULT NULL,
  `created_by` BIGINT NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_cost_corp_month_type` (`corp_id`, `accounting_month`, `cost_type`, `status`),
  KEY `idx_cost_item` (`corp_id`, `order_item_id`),
  KEY `idx_cost_completion` (`corp_id`, `production_completion_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='成本和期间费用明细';

CREATE TABLE IF NOT EXISTS `cost_allocations` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `corp_id` BIGINT NOT NULL,
  `cost_entry_id` BIGINT NOT NULL,
  `production_completion_id` BIGINT NOT NULL,
  `allocated_amount` DECIMAL(18,2) NOT NULL,
  `allocation_basis` VARCHAR(100) NOT NULL,
  `rule_version` VARCHAR(50) NOT NULL DEFAULT 'v1',
  `created_by` BIGINT NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_allocation_cost_completion` (`corp_id`, `cost_entry_id`, `production_completion_id`),
  KEY `idx_allocation_completion` (`corp_id`, `production_completion_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='共用成本分配记录';

CREATE TABLE IF NOT EXISTS `monthly_closings` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `corp_id` BIGINT NOT NULL,
  `accounting_month` CHAR(7) NOT NULL,
  `status` VARCHAR(20) NOT NULL DEFAULT 'open',
  `rule_version` VARCHAR(50) NOT NULL DEFAULT 'v1',
  `closed_by` BIGINT DEFAULT NULL,
  `closed_at` DATETIME(3) DEFAULT NULL,
  `notes` VARCHAR(500) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_closing_corp_month` (`corp_id`, `accounting_month`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='月结状态';

-- 财务页按钮资源。仅补充不存在的记录，不覆盖现有权限配置。
SET @finance_menu_id = (SELECT id FROM resources WHERE resource_key = 'finance' AND resource_type = 'menu' LIMIT 1);
INSERT INTO resources (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`)
SELECT 'button', 'btn-view-finance-summary', '查看经营报表', @finance_menu_id, NULL, NULL, 1, 'business', 'active'
WHERE @finance_menu_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM resources WHERE resource_key = 'btn-view-finance-summary');

INSERT INTO resources (`resource_type`, `resource_key`, `name`, `parent_id`, `path`, `icon`, `sort_order`, `platform_type`, `status`)
SELECT 'button', 'btn-manage-cost-entry', '录入成本费用', @finance_menu_id, NULL, NULL, 2, 'business', 'active'
WHERE @finance_menu_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM resources WHERE resource_key = 'btn-manage-cost-entry');
