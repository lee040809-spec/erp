-- 修改现有表的主键为 bigint

-- orders 表
ALTER TABLE `orders` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `orders` MODIFY COLUMN `corp_id` bigint DEFAULT 0;
ALTER TABLE `orders` MODIFY COLUMN `customer_id` bigint DEFAULT NULL;
ALTER TABLE `orders` MODIFY COLUMN `contact_id` bigint DEFAULT NULL;

-- order_items 表
ALTER TABLE `order_items` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `order_items` MODIFY COLUMN `order_id` bigint NOT NULL;
ALTER TABLE `order_items` MODIFY COLUMN `corp_id` bigint DEFAULT 0;

-- order_processes 表
ALTER TABLE `order_processes` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `order_processes` MODIFY COLUMN `order_item_id` bigint NOT NULL;
ALTER TABLE `order_processes` MODIFY COLUMN `corp_id` bigint DEFAULT 0;

-- customers 表
ALTER TABLE `customers` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `customers` MODIFY COLUMN `corp_id` bigint DEFAULT 0;

-- contacts 表
ALTER TABLE `contacts` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `contacts` MODIFY COLUMN `corp_id` bigint DEFAULT 0;
ALTER TABLE `contacts` MODIFY COLUMN `customer_id` bigint NOT NULL;

-- materials 表
ALTER TABLE `materials` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `materials` MODIFY COLUMN `corp_id` bigint DEFAULT 0;

-- remnants 表
ALTER TABLE `remnants` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `remnants` MODIFY COLUMN `corp_id` bigint DEFAULT 0;
ALTER TABLE `remnants` MODIFY COLUMN `material_id` bigint DEFAULT NULL;

-- advent_rules 表
ALTER TABLE `advent_rules` MODIFY COLUMN `id` bigint NOT NULL AUTO_INCREMENT;
ALTER TABLE `advent_rules` MODIFY COLUMN `corp_id` bigint DEFAULT 0;