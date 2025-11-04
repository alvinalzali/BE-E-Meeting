-- ==============================
-- TABLE: snacks
-- ==============================

CREATE TYPE snack_unit AS ENUM ('person', 'box');
CREATE TYPE category_snack AS ENUM ('Breakfast', 'Lunch', 'Dinner');

CREATE TABLE snacks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category category_snack NOT NULL,
    unit snack_unit NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

