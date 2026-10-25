ALTER TABLE production_completions ADD COLUMN pricing_status VARCHAR(16) NOT NULL DEFAULT 'priced';

CREATE TABLE IF NOT EXISTS cost_entry_requests (
 corp_id BIGINT NOT NULL,request_id VARCHAR(36) NOT NULL,request_hash CHAR(64) NOT NULL,cost_entry_id BIGINT NOT NULL,
 PRIMARY KEY(corp_id,request_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS completion_price_changes (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,production_completion_id BIGINT NOT NULL,
 unit_price DECIMAL(15,4) NOT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_completion_price(corp_id,production_completion_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS delivery_reconciliations (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,customer_id BIGINT NOT NULL,
 statement_no VARCHAR(60) NOT NULL,customer_name VARCHAR(255) NOT NULL DEFAULT '',business_date DATE NOT NULL,amount DECIMAL(15,2) NOT NULL,
 status VARCHAR(16) NOT NULL DEFAULT 'confirmed',notes VARCHAR(500) NOT NULL DEFAULT '',
 request_id VARCHAR(36) NOT NULL,request_hash CHAR(64) NOT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 reversed_by BIGINT NULL,reversed_at DATETIME(3) NULL,reversal_reason VARCHAR(500) NULL,
 UNIQUE KEY uq_delivery_recon_request(corp_id,request_id),KEY ix_delivery_recon_customer(corp_id,customer_id,business_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS delivery_reconciliation_lines (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,reconciliation_id BIGINT NOT NULL,
 delivery_record_id BIGINT NOT NULL,active_delivery_id BIGINT NULL,order_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,
 order_number VARCHAR(100) NOT NULL,part_name VARCHAR(255) NOT NULL,part_number VARCHAR(255) NOT NULL,
 business_date DATE NOT NULL,quantity INT NOT NULL,unit_price DECIMAL(15,4) NOT NULL,amount DECIMAL(15,2) NOT NULL,
 UNIQUE KEY uq_delivery_recon_active(corp_id,active_delivery_id),KEY ix_delivery_recon_lines(corp_id,reconciliation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
