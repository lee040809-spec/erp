-- 订单查询优化索引
-- 先删除旧索引，再创建新索引

-- orders 表索引
DROP INDEX idx_orders_corp_status ON orders;
DROP INDEX idx_orders_status_date ON orders;
DROP INDEX idx_orders_number ON orders;
DROP INDEX idx_orders_customer ON orders;

CREATE INDEX idx_orders_corp_status_date ON orders(corp_id, status, start_date, id);
CREATE INDEX idx_orders_number ON orders(corp_id, order_number);
CREATE INDEX idx_orders_customer ON orders(corp_id, customer_short_name(50));

-- order_items 表索引
DROP INDEX idx_order_items_order ON order_items;
DROP INDEX idx_order_items_due_date ON order_items;
DROP INDEX idx_order_items_status ON order_items;
DROP INDEX idx_order_items_part_number ON order_items;

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_due_date ON order_items(order_id, due_date);
CREATE INDEX idx_order_items_status ON order_items(order_id, status);

-- order_processes 表索引
DROP INDEX idx_order_processes_item ON order_processes;

CREATE INDEX idx_order_processes_item ON order_processes(order_item_id, sort_order);

-- advent_rules 表索引
DROP INDEX idx_advent_rules_corp_type ON advent_rules;

CREATE INDEX idx_advent_rules_corp_type ON advent_rules(corp_id, ruleType);