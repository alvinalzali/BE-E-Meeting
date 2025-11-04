-- ==============================
-- TABLE: reservations
-- ==============================

CREATE TYPE status_reservation AS ENUM ('booked', 'paid', 'cancel');

CREATE TABLE reservations (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    contact_name VARCHAR(100),
    contact_phone VARCHAR(50),
    contact_company VARCHAR(255),
    duration_minute INT,
    total_participants INT,
    add_snack BOOLEAN DEFAULT FALSE,
    subtotal_snack DECIMAL(14,2),
    subtotal_room DECIMAL(14,2),
    total DECIMAL(14,2),
    note TEXT,
    status_reservation status_reservation DEFAULT 'booked',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

