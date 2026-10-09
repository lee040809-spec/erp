-- 015: separate contracted quantity, physical production and revenue-bearing completions.
-- Backup first. Rerunnable: nullable columns mark rows awaiting backfill; existing values and outputs are preserved.
SET @ddl = IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='order_items' AND column_name='production_quantity')=0,'ALTER TABLE order_items ADD COLUMN production_quantity INT NULL DEFAULT NULL','SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
UPDATE order_items i LEFT JOIN (SELECT corp_id,order_item_id,SUM(qualified_quantity) AS qualified FROM production_completions WHERE status='confirmed' GROUP BY corp_id,order_item_id) p ON p.corp_id=i.corp_id AND p.order_item_id=i.id
SET i.production_quantity=GREATEST(i.quantity,COALESCE(p.qualified,0)+COALESCE(i.scrap_quantity,0)) WHERE i.production_quantity IS NULL;
SET @ddl = IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='production_completions' AND column_name='billable_quantity')=0,'ALTER TABLE production_completions ADD COLUMN billable_quantity DECIMAL(12,3) NULL DEFAULT NULL','SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
UPDATE production_completions SET billable_quantity=qualified_quantity WHERE billable_quantity IS NULL;
