-- 为所有表添加公司ID字段
-- 执行前请备份数据库
-- 如果列已存在会报错，可忽略

-- 1. orders 表
ALTER TABLE orders ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';
ALTER TABLE orders ADD COLUMN contact_name VARCHAR(100) DEFAULT NULL COMMENT '联系人姓名';

-- 2. order_items 表
ALTER TABLE order_items ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';

-- 3. order_processes 表
ALTER TABLE order_processes ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';

-- 4. customers 表
ALTER TABLE customers ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';

-- 5. contacts 表
ALTER TABLE contacts ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';

-- 6. materials 表
ALTER TABLE materials ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';

-- 7. remnants 表
ALTER TABLE remnants ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';

-- 8. advent_rules 表
ALTER TABLE advent_rules ADD COLUMN corp_id INT DEFAULT 0 COMMENT '公司id';