-- 将历史单公司数据从默认 corp_id=0 归属到唯一的有效公司。
-- 安全条件：只有数据库中恰好存在一个 active 公司时才会更新。

SET @active_company_count = (SELECT COUNT(*) FROM companies WHERE status = 'active');
SET @target_corp_id = (SELECT MIN(id) FROM companies WHERE status = 'active');

UPDATE users SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE roles SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE permissions SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE customers SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE contacts SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE orders SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE order_items SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE order_processes SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE materials SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE remnants SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE advent_rules SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE print_templates SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE production_completions SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE cost_entries SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE cost_allocations SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;
UPDATE monthly_closings SET corp_id = @target_corp_id WHERE corp_id = 0 AND @active_company_count = 1;

SELECT @active_company_count AS active_company_count, @target_corp_id AS target_corp_id;
