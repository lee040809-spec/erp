-- Apply once after a full database backup. Existing rows remain NULL: no historical timestamps are invented.
ALTER TABLE order_processes
 ADD COLUMN planned_start_at DATETIME(3) NULL,
 ADD COLUMN planned_end_at DATETIME(3) NULL,
 ADD COLUMN actual_start_at DATETIME(3) NULL,
 ADD COLUMN actual_end_at DATETIME(3) NULL;

CREATE TABLE IF NOT EXISTS process_time_events (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 corp_id BIGINT NOT NULL,
 order_item_id BIGINT NOT NULL,
 process_id BIGINT NOT NULL,
 before_nodes TEXT NOT NULL,
 after_nodes TEXT NOT NULL,
 reason VARCHAR(500) NOT NULL DEFAULT '',
 source VARCHAR(32) NOT NULL,
 created_by BIGINT NOT NULL,
 created_at DATETIME(3) NOT NULL,
 KEY ix_process_time_item (corp_id,order_item_id,id),
 KEY ix_process_time_process (corp_id,process_id,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
