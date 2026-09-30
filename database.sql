-- This is the SQL script that will be used to initialize the database schema.
-- We will evaluate you based on how well you design your database.
-- 1. How you design the tables.
-- 2. How you choose the data types and keys.
-- 3. How you name the fields.
-- In this assignment we will use PostgreSQL as the database.

-- Estates are immutable (no update endpoint), so "tree plot lies within the
-- estate" is checked in the application against the estate row it just read.
-- Enforcing it here would need a cross-table trigger.
CREATE TABLE estate (
	id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
	length INTEGER NOT NULL CHECK (length BETWEEN 1 AND 50000),
	width INTEGER NOT NULL CHECK (width BETWEEN 1 AND 50000),
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tree (
	id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
	estate_id UUID NOT NULL REFERENCES estate(id) ON DELETE CASCADE,
	x INTEGER NOT NULL CHECK (x >= 1),
	y INTEGER NOT NULL CHECK (y >= 1),
	height INTEGER NOT NULL CHECK (height BETWEEN 1 AND 30),
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	-- One tree per plot, race-free (duplicate -> 23505 -> HTTP 400).
	-- The index also covers the FK and per-estate reads; INCLUDE (height)
	-- makes stats and drone-plan eligible for index-only scans.
	CONSTRAINT tree_estate_plot_key UNIQUE (estate_id, x, y) INCLUDE (height)
);
