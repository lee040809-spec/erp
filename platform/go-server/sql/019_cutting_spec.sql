-- Apply once after backing up order_processes. Existing rows remain NULL.
ALTER TABLE order_processes ADD COLUMN cutting_spec VARCHAR(500) NULL COMMENT '下料尺寸规格';
