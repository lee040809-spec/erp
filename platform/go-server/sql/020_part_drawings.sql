-- Additive migration. Existing order_items.drawing_data is preserved.
CREATE TABLE IF NOT EXISTS part_drawings (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, corp_id BIGINT NOT NULL,
 order_item_id BIGINT NULL, source_drawing_id BIGINT NULL, replaces_id BIGINT NULL,
 storage_key VARCHAR(32) NOT NULL, sha256 VARCHAR(64) NOT NULL,
 file_name VARCHAR(255) NOT NULL, mime_type VARCHAR(100) NOT NULL, size BIGINT NOT NULL,
 drawing_number VARCHAR(100) NOT NULL DEFAULT '', version VARCHAR(100) NOT NULL DEFAULT '',
 notes VARCHAR(500) NOT NULL DEFAULT '', legacy_source BOOLEAN NOT NULL DEFAULT FALSE, is_current BOOLEAN NOT NULL DEFAULT FALSE,
 uploaded_by BIGINT NOT NULL, created_at DATETIME(3) NOT NULL,
 INDEX idx_drawing_item (corp_id,order_item_id,is_current,id),
 INDEX idx_drawing_owner (corp_id,uploaded_by,order_item_id), INDEX idx_drawing_source (source_drawing_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS drawing_events (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, corp_id BIGINT NOT NULL,
 order_item_id BIGINT NOT NULL, drawing_id BIGINT NOT NULL,
 action VARCHAR(30) NOT NULL, reason VARCHAR(500) NOT NULL DEFAULT '',
 actor_id BIGINT NOT NULL, created_at DATETIME(3) NOT NULL,
 INDEX idx_drawing_event (corp_id,order_item_id,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
