package workflowmigration

const RecordChangesDDL = `CREATE TABLE IF NOT EXISTS production_record_changes (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,
 kind VARCHAR(20) NOT NULL,record_id BIGINT NOT NULL,request_id VARCHAR(36) NOT NULL,request_hash CHAR(64) NOT NULL,
 reason VARCHAR(500) NOT NULL,before_json MEDIUMTEXT NOT NULL,after_json MEDIUMTEXT NOT NULL,
 before_quantity VARCHAR(40) NOT NULL,after_quantity VARCHAR(40) NOT NULL,before_date VARCHAR(10) NOT NULL,after_date VARCHAR(10) NOT NULL,
 created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_record_change_request(corp_id,request_id),KEY ix_record_change_item(corp_id,order_item_id,id),KEY ix_record_change_record(corp_id,kind,record_id,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
