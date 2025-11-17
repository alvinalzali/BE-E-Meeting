# 📌 E-Meeting API

Backend API untuk sistem pemesanan ruang meeting menggunakan **Golang (Echo Framework)**, **PostgreSQL**, dan **JWT Authentication**.
Mendukung reservasi ruangan, snack, manajemen user, upload gambar, serta Swagger Documentation.

---

## ✨ Features

### 🔐 Authentication

* Register user
* Login (JWT access + refresh token)
* Reset password (request token + update via token)

### 👥 Users

* Get user by ID
* Update user (with avatar upload & validation)

### 🏢 Rooms

* Create room (with image validation)
* Update room
* Delete room
* Search + Pagination
* Room schedule listing

### 🍽 Snacks

* List all snacks

### 📅 Reservations

* Reservation calculation
* Create reservation
* Reservation history (filter + pagination)
* Get reservation detail
* Schedule listing

### 📸 File Upload

* Upload image (temp folder)
* Auto-move image to final folder on update

---

## 🛠 Tech Stack

| Tech                    | Description         |
| ----------------------- | ------------------- |
| **Go 1.22+**            | Backend             |
| **Echo v4**             | Web Framework       |
| **PostgreSQL**          | Database            |
| **golang-migrate**      | Database Migration  |
| **JWT (golang-jwt v5)** | Auth                |
| **bcrypt**              | Password Encryption |
| **Swagger**             | API Documentation   |

---

## 📂 Project Structure

```
├── main.go
├── migrations/
│   ├── 1_users.up.sql
│   ├── 1_users.down.sql
│   └── ...
├── app/
│   └── entities/
├── assets/
│   ├── temp/
│   ├── rooms/
│   └── default/
├── .env
└── go.mod
```

---

## ⚙️ Environment Variables (.env)

```
db_host=localhost
db_port=5432
db_user=postgres
db_password=yourpassword
db_name=e_meeting_db

jwt_secret=yourJWTsecret
secret_key=yourJWTsecret
```

---

## 🚀 How to Run

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Run Database Migration

Saat program dijalankan, akan muncul:

```
Enter 1 for migrate up, 2 for migrate down, 3 for continue:
```

Pilih sesuai kebutuhan:

* **1** → migrate up
* **2** → migrate down
* **3** → lanjut menjalankan API

### 3. Start Server

```bash
go run main.go
```

Server berjalan di:

```
http://localhost:8080
```

---

## 📚 Swagger Docs

Akses dokumentasi API lengkap di:

```
/swagger/index.html
```

---

## 🔑 Authentication

Gunakan JWT:

```
Authorization: Bearer <token>
```

Role:

* `admin`
* `user`

Contoh penggunaan middleware:

```go
roleAuthMiddleware("admin", "user")
```

---

## 📸 Upload Image

**Endpoint**

```
POST /uploads
```

**Form File**

```
image: <file>
```

**Validasi**

* JPEG / PNG
* Max size 1MB
* Disimpan sementara di `/assets/temp`

---

## 🏢 Rooms API

### Create Room

```
POST /rooms
```

### Get Rooms

```
GET /rooms?name=&type=&capacity=&page=&pageSize=
```

### Update Room

```
PUT /rooms/:id
```

### Delete Room

```
DELETE /rooms/:id
```

---

## 📅 Reservation API

### Calculate Reservation

```
GET /reservation/calculation
```

### Create Reservation

```
POST /reservation
```

### Reservation History

```
GET /reservation/history?startDate=&endDate=&type=&status=&page=&pageSize=
```

### Get Reservation Detail

```
GET /reservation/:id
```

---

## 🧩 Deployment Notes

Pastikan folder berikut memiliki akses yang benar:

```
assets/temp/
assets/rooms/
assets/image/users/
assets/default/
```

Jika menggunakan nginx:

```
proxy_pass http://localhost:8080;
```

---

## 🤝 Contribution

Open untuk Pull Request dan Issue.

---

## 📄 License

MIT License © 2025 — E-Meeting API
