-- ==============================
-- ENUM DEFINITIONS
-- ==============================

CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE status_reservation AS ENUM ('booked', 'paid', 'cancel');
CREATE TYPE user_role AS ENUM ('admin', 'user');
CREATE TYPE snack_unit AS ENUM ('person', 'box');
CREATE TYPE room_type AS ENUM ('small', 'medium', 'large');
CREATE TYPE user_lang AS ENUM ('english', 'indonesia');
CREATE TYPE category_snack AS ENUM ('Breakfast', 'Lunch', 'Dinner');

-- ==============================
-- TABLE: users
-- ==============================

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    status user_status DEFAULT 'active',
    role user_role DEFAULT 'user',
    lang user_lang DEFAULT 'english',
    avatar_url VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

-- ==============================
-- TABLE: rooms
-- ==============================

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

-- ==============================
-- TABLE: rooms
-- ==============================

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

-- ==============================
-- TABLE: snacks
-- ==============================

CREATE TABLE snacks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category category_snack NOT NULL,
    unit snack_unit NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

-- ==============================
-- TABLE: reservations
-- ==============================

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

