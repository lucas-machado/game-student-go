ALTER TABLE users ADD COLUMN stripe_id TEXT;
ALTER TABLE users ADD CONSTRAINT stripe_id_unique UNIQUE (stripe_id);
