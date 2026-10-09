CREATE TABLE IF NOT EXISTS production_plans (
 id BIGINT NOT NULL AUTO_INCREMENT,
 corp_id BIGINT NOT NULL,
 order_item_id BIGINT NOT NULL,
 assignee VARCHAR(100) NOT NULL DEFAULT '',
 equipment VARCHAR(100) NOT NULL DEFAULT '',
 planned_start DATE NULL,
 planned_end DATE NULL,
 estimated_hours DECIMAL(10,2) NOT NULL DEFAULT 0,
 blocked_reason VARCHAR(500) NOT NULL DEFAULT '',
 notes VARCHAR(500) NOT NULL DEFAULT '',
 updated_by BIGINT NOT NULL,
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 PRIMARY KEY(id),
 UNIQUE KEY uk_production_plan_item(corp_id,order_item_id),
 KEY idx_production_plan_end(corp_id,planned_end)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
