CREATE TABLE IF NOT EXISTS business_entry_policies (
 corp_id BIGINT NOT NULL PRIMARY KEY,
 policy TEXT NOT NULL,
 role_modes TEXT NOT NULL,
 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS order_business_records (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 corp_id BIGINT NOT NULL,
 order_id BIGINT NOT NULL,
 order_item_id BIGINT NULL,
 kind VARCHAR(20) NOT NULL,
 quantity INT NOT NULL DEFAULT 0,
 business_date DATE NULL,
 process_name VARCHAR(80) NOT NULL DEFAULT '',
 category VARCHAR(40) NOT NULL DEFAULT '',
 severity VARCHAR(20) NOT NULL DEFAULT 'normal',
 description VARCHAR(500) NOT NULL DEFAULT '',
 notes VARCHAR(500) NOT NULL DEFAULT '',
 source_type VARCHAR(20) NOT NULL DEFAULT 'business',
 request_id VARCHAR(80) NOT NULL,
 status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
 created_by BIGINT NOT NULL,
 resolved_by BIGINT NULL,
 resolved_at DATETIME NULL,
 resolution VARCHAR(500) NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE KEY uq_business_request (corp_id,request_id),
 KEY ix_business_order (corp_id,order_id,kind),
 KEY ix_business_item (corp_id,order_item_id,kind,status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO order_business_records (corp_id,order_id,order_item_id,kind,quantity,business_date,notes,source_type,request_id,status,created_by)
SELECT corp_id,order_id,id,'scrap',scrap_quantity,NULL,'历史累计报废存量；原始日期未知，保留原数量','legacy',CONCAT('LEGACY-SCRAP-',id),'confirmed',0
FROM order_items i WHERE COALESCE(i.scrap_quantity,0)>0 AND NOT EXISTS (SELECT 1 FROM order_business_records r WHERE r.corp_id=i.corp_id AND r.order_item_id=i.id AND r.kind='scrap');

INSERT IGNORE INTO order_business_records (corp_id,order_id,order_item_id,kind,quantity,business_date,notes,source_type,request_id,status,created_by)
SELECT corp_id,order_id,id,'delivery',delivered_quantity,NULL,'历史累计交货存量；原始日期未知，保留原数量','legacy',CONCAT('LEGACY-DELIVERY-',id),'confirmed',0
FROM order_items i WHERE COALESCE(i.delivered_quantity,0)>0 AND NOT EXISTS (SELECT 1 FROM order_business_records r WHERE r.corp_id=i.corp_id AND r.order_item_id=i.id AND r.kind='delivery');
