-- migrations/000008_add_payments.down.sql

DROP TRIGGER IF EXISTS set_payments_updated_at ON payments;
DROP TABLE IF EXISTS payments;
DROP TYPE IF EXISTS payment_purpose;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_method;
