<div align="center">

# ⚡ Netcore

### Network Management API

**Backend API untuk membangun sistem manajemen jaringan yang terintegrasi.**

[![Go](https://img.shields.io/badge/Go-Backend-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![API](https://img.shields.io/badge/REST-API-2563EB?style=for-the-badge&logo=fastapi&logoColor=white)](https://en.wikipedia.org/wiki/REST)
[![License](https://img.shields.io/badge/License-TBD-gray?style=for-the-badge)](#license)

</div>

---

## 📖 About

**Netcore** adalah backend API berbasis Go yang dirancang sebagai fondasi untuk aplikasi manajemen jaringan.

Project ini dikembangkan dengan struktur modular agar mudah dikembangkan, dipelihara, dan diintegrasikan dengan frontend maupun layanan jaringan lainnya.

## ✨ Features

- 🔌 **Network Management** — Fondasi API untuk pengelolaan perangkat jaringan.
- 📡 **GenieACS Integration** — Integrasi dengan GenieACS untuk manajemen perangkat yang mendukung TR-069.
- 🌐 **MikroTik Integration** — Mendukung pengembangan fitur manajemen router MikroTik.
- 🔐 **Authentication** — Fondasi autentikasi untuk API.
- 🗄️ **Database Integration** — Mendukung penyimpanan data aplikasi.
- 🧩 **Modular Architecture** — Struktur kode yang memudahkan pengembangan fitur baru.

> Status fitur mengikuti implementasi aktual dalam source code.

## 🛠️ Tech Stack

| Technology | Purpose                       |
| ---------- | ----------------------------- |
| Go         | Backend application           |
| Gin        | HTTP routing                  |
| PostgreSQL | Database                      |
| GORM       | ORM                           |
| JWT        | Token-based authentication    |
| GenieACS   | Device management integration |

## 📂 Project Structure

```text
netcore/
├── cmd/
│   └── ...
├── internal/
│   ├── controllers/
│   ├── models/
│   ├── routes/
│   └── services/
├── tests/
├── .air.toml
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## 🚀 Getting Started

### Prerequisites

Pastikan perangkat kamu sudah memiliki:

- Go
- PostgreSQL
- Git

### 1. Clone Repository

```bash
git clone https://github.com/asepcepot6140-glitch/netcore.git
cd netcore
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Configure Environment

Buat file `.env` berdasarkan konfigurasi yang dibutuhkan aplikasi.

Jangan commit file `.env` karena dapat berisi kredensial dan konfigurasi sensitif.

### 4. Run Application

Jalankan aplikasi menggunakan entry point yang sesuai dengan struktur project:

```bash
go run ./cmd/...
```

Sesuaikan perintah tersebut jika project memiliki entry point tertentu, misalnya `cmd/server/main.go`.

## 🧪 Testing

Jalankan unit test dengan:

```bash
go test ./...
```

Periksa kompilasi seluruh package dengan:

```bash
go build ./...
```

## 🗺️ Roadmap

- [ ] Pengembangan API manajemen perangkat jaringan.
- [ ] Integrasi manajemen MikroTik.
- [ ] Pengembangan integrasi GenieACS.
- [ ] Pengelolaan pelanggan dan layanan jaringan.
- [ ] Pengembangan monitoring dan pelaporan.
- [ ] Peningkatan keamanan dan dokumentasi API.

## 🔒 Security

Jika menemukan masalah keamanan, hindari memublikasikan kredensial atau informasi sensitif dalam issue maupun log.

Pastikan password database, JWT secret, dan konfigurasi privat disimpan melalui environment variables.

## 📄 License

Lisensi project belum ditentukan. Tambahkan file `LICENSE` sebelum mendistribusikan project dengan ketentuan lisensi tertentu.

---

<div align="center">

**Netcore** · Built for Network Management

</div>
