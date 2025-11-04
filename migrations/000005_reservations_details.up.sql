-- ==============================
-- TABLE: reservation_details
-- ==============================


CREATE TABLE reservation_details (
    id SERIAL PRIMARY KEY,
    reservation_id INT REFERENCES reservations(id) ON DELETE CASCADE,
    room_id INT REFERENCES rooms(id) ON DELETE CASCADE,
    room_name VARCHAR(100) NOT NULL,
    room_price DECIMAL(12,2) NOT NULL,
    snack_id INT REFERENCES snacks(id) ON DELETE SET NULL,
    snack_name VARCHAR(100) NOT NULL,
    snack_price DECIMAL(12,2) NOT NULL,
    duration_minute INT,
    total_participants INT,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    total_snack DECIMAL(14,2),
    total_room DECIMAL(14,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);