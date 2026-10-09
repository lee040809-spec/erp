-- 当系统只有一个有效公司时，将全部租户数据统一到该公司的当前 ID。
-- 用于公司主键发生调整后修复 users、orders 等表的 corp_id 不一致。

SET @active_company_count = (SELECT COUNT(*) FROM companies WHERE status = 'active');
SET @target_corp_id = (SELECT MIN(id) FROM companies WHERE status = 'active');

UPDATE users SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE roles SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE permissions SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE customers SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE contacts SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE orders SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE order_items SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE order_processes SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE materials SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE remnants SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE advent_rules SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE print_templates SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE production_completions SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE cost_entries SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE cost_allocations SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);
UPDATE monthly_closings SET corp_id = @target_corp_id WHERE @active_company_count = 1 AND (corp_id IS NULL OR corp_id <> @target_corp_id);

SELECT @active_company_count AS active_company_count, @target_corp_id AS target_corp_id;
