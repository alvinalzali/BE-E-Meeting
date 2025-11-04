-- ==============================
-- TABLE: rooms
-- ==============================

CREATE TYPE room_type AS ENUM ('small', 'medium', 'large');

CREATE TABLE rooms (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    room_type room_type NOT NULL,
    capacity INT NOT NULL,
    price_per_hour DECIMAL(12,2) NOT NULL,
    picture_url VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

