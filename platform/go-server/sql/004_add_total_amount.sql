-- 添加订单总金额列
ALTER TABLE orders ADD COLUMN total_amount DECIMAL(12,2) DEFAULT 0 COMMENT '订单总金额';