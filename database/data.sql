-- ============================================
-- SEED DATA FOR ROOMS, SNACKS, RESERVATIONS, RESERVATION_DETAILS
-- ============================================

--insert initial users
INSERT INTO users (username, email, password_hash, name, status, role, lang, created_at)
VALUES 
('admin', 'admin@gmail.com','$2a$10$RzavPkF7UEuTStyVzZPYDelPVAL8I/OWx3SGmcbaNvrQRk86Ihuii', 'admin', 'active', 'admin', 'english', NOW()),
('user', 'user@gmail.com','$2a$10$xxV.4wmjnKeIgf6N.ndITeHr6O.H9TIVDD//4PSmEY57scTONLrmS', 'user', 'active', 'user', 'english', NOW());
-- pass : Admin@40 or User@40


-- Insert initial rooms
INSERT INTO rooms (name, room_type, capacity, price_per_hour, picture_url)
VALUES 
('Room Sakura', 'small', 5, 150000, 'https://example.com/room-sakura.jpg'),
('Room Fuji', 'medium', 10, 300000, 'https://example.com/room-fuji.jpg'),
('Room Tokyo', 'large', 20, 500000, 'https://example.com/room-tokyo.jpg');

-- Insert initial snacks
INSERT INTO snacks (name, category, unit, price)
VALUES
('Breakfast Set', 'Breakfast', 'box', 50000),
('Lunch Bento', 'Lunch', 'box', 75000),
('Dinner Buffet', 'Dinner', 'person', 100000);

-- Insert reservation (assuming user_id = 1 exists)
INSERT INTO reservations (
    user_id, contact_name, contact_phone, contact_company, 
    duration_minute, total_participants, add_snack, subtotal_snack, 
    subtotal_room, total, note, status_reservation
)
VALUES 
(1, 'John Doe', '081234567890', 'TechnoHub', 
 120, 10, TRUE, 750000, 300000, 1050000, 'First booking', 'booked');

-- Insert reservation details (assume reservation_id = 1)
INSERT INTO reservation_details (
    reservation_id, room_id, room_name, room_price,
    snack_id, snack_name, snack_price, duration_minute,
    total_participants, start_at, end_at,
    total_snack, total_room
)
VALUES
(1, 2, 'Room Fuji', 300000, 2, 'Lunch Bento', 75000, 120, 10,
 NOW(), NOW() + INTERVAL '2 hours', 750000, 300000);
