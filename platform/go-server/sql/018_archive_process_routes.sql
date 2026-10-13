-- Archive removed operations without deleting physical production, outsourcing or financial history.
-- Existing rows remain active: deleted_at defaults to NULL. Run after backing up the database.
ALTER TABLE order_processes ADD COLUMN deleted_at DATETIME(3) NULL;
