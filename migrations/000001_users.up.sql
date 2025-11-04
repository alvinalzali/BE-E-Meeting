-- ==============================
-- TABLE: users
-- ==============================

CREATE TYPE user_role AS ENUM ('admin', 'user');
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE user_lang AS ENUM ('english', 'indonesia');

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



