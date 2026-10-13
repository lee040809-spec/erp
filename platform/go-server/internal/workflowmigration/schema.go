package workflowmigration

type Column struct{ Table, Name, Definition string }

var Columns = []Column{
	{"order_processes", "cutting_spec", "VARCHAR(500) NULL COMMENT '下料尺寸规格'"},
	{"order_processes", "deleted_at", "DATETIME(3) NULL"},
	{"production_completions", "delivery_batch_id", "BIGINT NULL"},
	{"order_business_records", "delivery_batch_id", "BIGINT NULL"},
	{"cost_entries", "delivery_batch_id", "BIGINT NULL"},
	{"outsourcing_dispatches", "supplier_id", "BIGINT NULL"},
	{"outsourcing_dispatches", "outsourcing_order_id", "BIGINT NULL"},
	{"outsourcing_dispatches", "outsourcing_order_line_id", "BIGINT NULL"},
	{"outsourcing_dispatches", "scope", "VARCHAR(20) NOT NULL DEFAULT 'selected'"},
}
var Tables = []string{
	`CREATE TABLE IF NOT EXISTS outsourcing_changes (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,entity_type VARCHAR(20) NOT NULL,entity_id BIGINT NOT NULL,action VARCHAR(20) NOT NULL,
 reason VARCHAR(500) NOT NULL DEFAULT '',before_json MEDIUMTEXT NOT NULL,after_json MEDIUMTEXT NOT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 KEY ix_outsource_change(corp_id,entity_type,entity_id,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS order_delivery_batches (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,order_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,
 batch_no VARCHAR(40) NOT NULL,quantity INT NOT NULL,production_quantity INT NOT NULL,due_date DATE NOT NULL,notes VARCHAR(500) NOT NULL DEFAULT '',created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_delivery_batch(corp_id,order_item_id,batch_no),KEY ix_batch_item(corp_id,order_item_id,id),KEY ix_batch_due(corp_id,due_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS process_quantity_totals (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,order_process_id BIGINT NOT NULL,
 started_quantity INT NOT NULL DEFAULT 0,completed_quantity INT NOT NULL DEFAULT 0,scrap_quantity INT NOT NULL DEFAULT 0,activated_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_process_quantity(corp_id,order_process_id),KEY ix_flow_item(corp_id,order_item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS process_quantity_events (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,order_process_id BIGINT NOT NULL,
 kind VARCHAR(20) NOT NULL,quantity INT NOT NULL,business_date DATE NULL,notes VARCHAR(500) NOT NULL DEFAULT '',source_type VARCHAR(30) NOT NULL,
 request_id VARCHAR(64) NOT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_quantity_request(corp_id,request_id),KEY ix_quantity_timeline(corp_id,order_item_id,business_date,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS outsourcing_dispatches (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,order_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,order_process_id BIGINT NOT NULL,delivery_batch_id BIGINT NULL,
 dispatch_no VARCHAR(60) NOT NULL,supplier_name VARCHAR(100) NOT NULL,quantity INT NOT NULL,unit_price DECIMAL(15,4) NOT NULL,
 supplier_id BIGINT NULL,outsourcing_order_id BIGINT NULL,outsourcing_order_line_id BIGINT NULL,scope VARCHAR(20) NOT NULL DEFAULT 'selected',
 dispatch_date DATE NOT NULL,expected_return_date DATE NOT NULL,open_quantity INT NOT NULL,rework_waiting_quantity INT NOT NULL DEFAULT 0,rework_out_quantity INT NOT NULL DEFAULT 0,
 accepted_quantity INT NOT NULL DEFAULT 0,concession_quantity INT NOT NULL DEFAULT 0,scrap_quantity INT NOT NULL DEFAULT 0,status VARCHAR(20) NOT NULL,
 request_id VARCHAR(36) NOT NULL,notes VARCHAR(500) NOT NULL DEFAULT '',created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_outsource_request(corp_id,request_id),KEY ix_outsource_item(corp_id,order_item_id,id),KEY ix_outsource_process(corp_id,order_process_id),KEY ix_outsource_status(corp_id,status,expected_return_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS outsourcing_events (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,dispatch_id BIGINT NOT NULL,kind VARCHAR(20) NOT NULL,quantity INT NOT NULL,
 good_quantity INT NOT NULL DEFAULT 0,concession_quantity INT NOT NULL DEFAULT 0,rework_quantity INT NOT NULL DEFAULT 0,scrap_quantity INT NOT NULL DEFAULT 0,
 business_date DATE NOT NULL,notes VARCHAR(500) NOT NULL DEFAULT '',request_id VARCHAR(36) NOT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_outsource_event_request(corp_id,request_id),KEY ix_outsource_events(corp_id,dispatch_id,business_date,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS outsourcing_settlements (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,dispatch_id BIGINT NOT NULL,quantity INT NOT NULL,amount DECIMAL(15,2) NOT NULL,tax_amount DECIMAL(15,2) NOT NULL,net_amount DECIMAL(15,2) NOT NULL,
 business_date DATE NOT NULL,reference_no VARCHAR(100) NOT NULL DEFAULT '',notes VARCHAR(500) NOT NULL DEFAULT '',cost_entry_id BIGINT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_outsource_settlement(corp_id,dispatch_id),KEY ix_settlement_cost(corp_id,cost_entry_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS outsourcing_suppliers (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,name VARCHAR(100) NOT NULL,
 contact_name VARCHAR(100) NOT NULL DEFAULT '',phone VARCHAR(60) NOT NULL DEFAULT '',address VARCHAR(300) NOT NULL DEFAULT '',capabilities VARCHAR(300) NOT NULL DEFAULT '',notes VARCHAR(500) NOT NULL DEFAULT '',active TINYINT(1) NOT NULL DEFAULT 1,
 created_at DATETIME(3) NOT NULL,updated_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_supplier_name(corp_id,name),KEY ix_supplier_active(corp_id,active,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS outsourcing_orders (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,order_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,supplier_id BIGINT NOT NULL,supplier_name VARCHAR(100) NOT NULL,
 order_no VARCHAR(60) NOT NULL,scope VARCHAR(20) NOT NULL,delivery_batch_id BIGINT NULL,unit_price DECIMAL(15,4) NOT NULL DEFAULT 0,order_date DATE NOT NULL,expected_return_date DATE NOT NULL,status VARCHAR(30) NOT NULL,notes VARCHAR(500) NOT NULL DEFAULT '',request_id VARCHAR(36) NOT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,
 UNIQUE KEY uq_outsource_order_request(corp_id,request_id),KEY ix_outsource_order_item(corp_id,order_item_id,id),KEY ix_outsource_order_supplier(corp_id,supplier_id,status,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS outsourcing_order_lines (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,outsourcing_order_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,order_process_id BIGINT NOT NULL,process_name VARCHAR(100) NOT NULL,sequence INT NOT NULL,quantity INT NOT NULL,dispatched_quantity INT NOT NULL DEFAULT 0,started_quantity INT NOT NULL DEFAULT 0,unit_price DECIMAL(15,4) NOT NULL DEFAULT 0,
 UNIQUE KEY uq_outsource_order_process(corp_id,outsourcing_order_id,order_process_id),KEY ix_outsource_line_process(corp_id,order_item_id,order_process_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS outsourcing_dispatch_processes (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,dispatch_id BIGINT NOT NULL,order_process_id BIGINT NOT NULL,sequence INT NOT NULL,
 UNIQUE KEY uq_dispatch_route(corp_id,dispatch_id,order_process_id),KEY ix_dispatch_route(corp_id,dispatch_id,sequence)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
}
