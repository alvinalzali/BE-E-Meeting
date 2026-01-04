# 📌 E-Hotel API

Backend API untuk sistem pemesanan ruang meeting menggunakan **Golang (Echo Framework)**, **PostgreSQL**, dan **JWT Authentication**.
Mendukung reservasi ruangan, snack, manajemen user, upload gambar, serta Swagger Documentation.

---

## ✨ Features

### 🔐 Authentication & User
* Register & Login (JWT access token)
* **Password Reset** (Request token via email simulation + Reset password)
* Get User Profile
* Update User (with avatar upload validation)

### 🏢 Rooms (Admin)
* Create room (with image validation)
* Update room details
* Delete room
* Get all rooms (Search + Pagination + Filter by type/capacity)
* Get specific room detail

### 🍽 Snacks
* List all snacks available

### 📅 Reservations
* **Check Availability** (Mencegah bentrok jadwal)
* **Calculation** (Estimasi harga sebelum booking)
* Create reservation (Booking ruangan + Snack)
* Reservation history (Filter by date, status, room type)
* Update Reservation Status (Admin: `booked` -> `paid`/`cancel`)
* Get Reservation Detail
* Room Schedule Listing

### 📊 Dashboard (Admin)
* View Total Omzet, Total Visitor, Total Reservations
* Room usage percentage statistics

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
├── app/
│   ├── entities/       # Definisi Struct (Model Data & DTO)
│   ├── handler/        # HTTP Handlers (Controller)
│   ├── middleware/     # Auth & Role Middleware
│   ├── repositories/   # Layer Akses Data (Query SQL)
│   └── usecases/       # Layer Bisnis Logic & Validasi
├── assets/
│   ├── default/
│   └── image/users/
├── database/           # Konfigurasi DB & Helper Migrasi
├── docs/
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── migrations/
│   ├── 1_users.up.sql
│   ├── 1_users.down.sql
│   └── ...
├── .env
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── LICENSE
├── main.go
└── ReadMe.md
```

---

## ⚙️ Environment Variables (.env)

```
db_host=localhost
db_port=5432
db_user=postgres
db_password=yourpassword
db_name=e_meeting_db

secret_key=yourJWTsecret
SKIP_MIGRATION=false # Kalau sudah berikan "True"
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

### 🔐 Auth
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/login` | Authenticate user & get token | No |
| `POST` | `/register` | Register a new user | No |
| `POST` | `/password/reset_request` | Request password reset token (via email) | No |
| `PUT` | `/password/reset/:token` | Reset password using valid token | No |

### 🏢 Rooms
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `GET` | `/rooms` | List all rooms (Search & Filter) | Yes |
| `POST` | `/rooms` | Create a new room | **Admin** |
| `GET` | `/rooms/:id/reservation` | Check specific room schedule | Yes |

#### 🔹 Detail: Get Rooms
**Endpoint:** `GET /rooms`
Filter rooms based on criteria.

| Query Param | Type | Description | Example |
| :--- | :--- | :--- | :--- |
| `name` | string | Filter by room name (partial match) | `Sakura` |
| `type` | string | Filter by room type (`small`, `medium`, `large`) | `medium` |
| `capacity` | int | Filter by minimum capacity | `10` |
| `page` | int | Page number (default: 1) | `1` |
| `pageSize` | int | Items per page (default: 10) | `10` |

### 📅 Reservation
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `GET` | `/reservation/calculation` | Calculate total price before booking | Yes |
| `POST` | `/reservation` | Create a new reservation (Booking) | Yes |
| `GET` | `/reservation/history` | View reservation history | Yes |
| `PUT` | `/reservation/status` | Update reservation status | **Admin** |

#### 🔹 Detail: Reservation History
**Endpoint:** `GET /reservation/history`
Retrieve booking history. Users see their own data; Admins see all data.

| Query Param | Type | Description | Example |
| :--- | :--- | :--- | :--- |
| `startDate` | string | Filter start date (YYYY-MM-DD) | `2024-01-01` |
| `endDate` | string | Filter end date (YYYY-MM-DD) | `2024-12-31` |
| `type` | string | Filter by room type | `large` |
| `status` | string | Filter by status (`booked`, `paid`, `cancel`) | `paid` |
| `page` | int | Page number | `1` |
| `pageSize` | int | Items per page | `10` |

### 📊 Dashboard
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `GET` | `/dashboard` | View analytics & statistics | **Admin** |

**Required Query Params:**
* `startDate` (YYYY-MM-DD)
* `endDate` (YYYY-MM-DD)

> **Catatan:**
> * Endpoint dengan Auth **Yes** membutuhkan header `Authorization: Bearer <token>`.
> * Endpoint Dashboard wajib menyertakan query param `startDate` dan `endDate` (Format: `YYYY-MM-DD`).
---

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
