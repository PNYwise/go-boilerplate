# Go Boilerplate

## Deskripsi Singkat

Template boilerplate untuk layanan backend menggunakan Go. Berfungsi sebagai panduan dan standar struktur kode, konvensi penamaan, serta contoh implementasi utilitas, repository, service, dan handler HTTP (Gin).

---

## Struktur Proyek

### Root Files
- **go.mod, go.sum**: Dependency dan versi module
- **Makefile**: Shortcut tugas developer (build, run, test, migrate)
- **README.md**: Dokumentasi proyek

### Direktori Utama
- **cmd/**: Entrypoint aplikasi. Setiap subfolder mewakili satu binary/entrypoint (misal: `main`). Di dalamnya diparsing flag, inisialisasi konfigurasi, dan memanggil wiring aplikasi.
- **internal/**: Kode aplikasi yang tidak diekspor ke package lain. Struktur internal merepresentasikan boundary arsitektur aplikasi.
  - **apps/**: Wiring dan bootstrap aplikasi (menghubungkan repo, service, transport, migration, health checks)
  - **configs/**: Definisi struct konfigurasi, loader, dan validasi konfigurasi (file/env precedence)
  - **dbs/**: Inisialisasi koneksi database, pool, helper transaksi, dan health checks DB
  - **dtos/**: Transport-level data shapes (request/response DTO), dipisah dari domain entities
  - **entities/**: Domain models dan value objects
  - **repositories/**: Definisi interface repository dan implementasi penyimpanan/data access (SQL/NoSQL/cache/RestApi). Pisahkan interface dan implementasi untuk memudahkan mocking
  - **services/**: Use-cases / business logic yang mengorkestrasi repositori dan external clients
  - **transports/**: Adapter transport (HTTP, gRPC, RMQ)
    - Untuk HTTP: `transports/http/router.go`, `handlers/`, `middlewares/`
  - **utils/**
    - **validation/**: Aturan validasi input (dipanggil sebelum masuk ke service)
    - **logs/**: Konfigurasi logger dan helper structured logging

- **internal/repositories/_mock/** dan test files: Mock otomatis untuk interfaces (digenerate dalam tests) dan test-related helpers
- **tmp/**: Artifacts lokal atau binary hasil build selama development; tidak untuk file penting

### Pedoman Tambahan
- Handler HTTP dan middleware diletakkan di `transports/http/handlers` dan `transports/http/middlewares`
- Tests mengikuti struktur yang sama dengan file yang dites (misal: `services/example_service_test.go`)
- File konfigurasi contoh (misal: `.env.stage.example`) disimpan di root, jangan commit file dengan secret

> **Tujuan struktur ini:**
> - Menjaga separation of concerns
> - Memudahkan testing
> - Membuat dependency wiring eksplisit

---

## Arsitektur & Aliran Data (Layer-by-Layer)

Penjelasan tanggung jawab tiap layer dan aliran data antar layer:

### Entrypoint / CLI (`cmd`)
- Parsing flag/argument, memilih `stage`, inisialisasi konfigurasi, logger, dan koneksi infrastruktur (DB, broker)
- Bootstrap aplikasi dan menyerahkan kontrol ke layer `apps`
- Aliran: menentukan stage → memuat konfigurasi → inisialisasi infrastruktur → memanggil wiring aplikasi

### Configs (`internal/configs`)
- Memuat, memvalidasi, dan menyediakan konfigurasi read-only ke seluruh aplikasi
- Aliran: nilai config diinject ke konstruktor (dependency injection) untuk digunakan oleh apps, repositori, dan services

### Apps / Wiring (`internal/apps`)
- Menyusun dan menghubungkan semua dependensi (repo, service, handler)
- Mengatur lifecycle (migrations, health checks)
- Memulai transport
- Aliran: menerima config & infra → instantiate komponen → registrasi route & middleware → mulai server

### Transports (HTTP) — Router & Server (`transports/http`)
- Menerima koneksi masuk, routing, dan lifecycle server
- Aliran request: masuk → middleware → handler

---

## Environment, Stage & Mode

### File .env
.env files menyimpan variabel lingkungan; **jangan commit file yang berisi kredensial**. Simpan contoh variabel di `.env.example` atau `.env.stage.example`.

#### File env per stage
- Gunakan file terpisah per stage untuk memudahkan deployment dan pengujian:
  - `.env.stage.local`
  - `.env.stage.development`
  - `.env.stage.staging`
  - `.env.stage.production`
- Pilih konvensi nama yang konsisten di seluruh tim (misal: gunakan `production` bukan `prod`, kecuali jika ada mapping jelas)

#### Flag --stage
- Memilih konfigurasi environment (lokal, development, staging, production)
- Menentukan file env yang akan dimuat, level logging default, koneksi ke resource (DB/queue), dan perilaku fitur

#### Flag --mode
- Menentukan subsistem yang dijalankan oleh binary
- Nilai umum: `http`, `grpc`, `worker`, `migrate`, `cli`
- Setiap mode hanya inisialisasi komponen yang diperlukan (misal: `migrate` hanya menyiapkan koneksi DB dan menjalankan migrasi)

#### Loader & Precedence
- Precedence konfigurasi: flag CLI (`--stage`) > environment variables > file konfigurasi
- Loader membentuk nama file berdasarkan nilai `--stage` (misal: `.env.stage.production`) dan memuatnya pada startup

#### Rekomendasi Operasional
- Pastikan `Makefile` dan dokumentasi memakai nama stage yang sama dengan file `.env.stage.*`
- Mode dan stage bersifat orthogonal: panggil `--mode` untuk menentukan subsistem dan `--stage` untuk menentukan konfigurasi environment
- Jangan commit `.env.stage.production` yang berisi secret; gunakan secret manager untuk produksi
- Log level harus sesuai: `Info` untuk alur normal, `Error` untuk kegagalan

---

## Dependency Injection dengan Wire

Proyek ini menggunakan [Google Wire](https://github.com/google/wire) untuk dependency injection yang aman dan performant.

### Keuntungan Wire
- **Compile-time safety**: Semua dependensi diresolve saat compile time
- **Zero runtime overhead**: Tidak ada reflection atau service locator
- **Explicit dependencies**: Dependency chain yang jelas dan mudah di-trace
- **Easy testing**: Mudah untuk inject mock dependencies

### Wire Structure
```
internal/wire/
├── wire.go          # Definisi providers dan injectors
└── wire_gen.go      # Kode yang di-generate otomatis (JANGAN EDIT)
```

### Provider Sets
- **DatabaseSet**: Koneksi database MySQL
- **ValidationSet**: Input validation
- **LoggerSet**: Structured logging dengan Elasticsearch
- **RepositorySet**: Semua repository implementations
- **ServiceSet**: Business logic services
- **HTTPSet**: HTTP server dengan routing

### Menggunakan Wire

#### Generate kode Wire:
```bash
make wire
```

#### Menambah dependency baru:
1. Buat provider function di package yang sesuai
2. Tambahkan ke provider set di `internal/wire/wire.go`  
3. Update service register jika perlu
4. Jalankan `make wire` untuk regenerate

### Detail Documentation
Lihat [WIRE.md](WIRE.md) untuk dokumentasi lengkap tentang penggunaan Wire.

---

## Testing

- Nama file test: `xxx_test.go`
- Gunakan table-driven tests dan mock untuk dependencies
- Untuk unit tests, mock external resources (DB, network) dan hanya tes logika bisnis

---

## Aturan Standarisasi Kode

### Git Standar Commit Message & Branching

#### Penamaan Branch
- Gunakan kode Jira card sebagai nama branch baru
  - Contoh: `feature/ABC-123-add-login`, `bugfix/XYZ-456-fix-auth`

#### Format Commit Message
Gunakan format berikut untuk commit message:

- `feat (nama_feature): message`  → Untuk penambahan fitur baru
- `fix (nama_feature): message`   → Untuk perbaikan bug
- `refactor (nama_feature): message` → Untuk perubahan/perbaikan kode tanpa menambah fitur
- `style (nama_feature): message` → Untuk perubahan style/kosmetik (indentasi, format, dll)
- `remove (nama_feature): message` → Untuk penghapusan fitur/kode

**Contoh:**
- `feat (auth): implementasi login JWT`
- `fix (order): perbaiki validasi input order`
- `refactor (db): optimasi query user`
- `style (ui): update warna tombol login`
- `remove (payment): hapus metode pembayaran lama`

#### Catatan
Pastikan setiap commit dan branch mengikuti standar di atas untuk memudahkan tracking dan kolaborasi

### Standar Kode Go

1. **Tangani semua error secara eksplisit**
   - Jangan mengabaikan error dengan blank identifier `_` tanpa alasan kuat
   - Selalu propagasi atau wrap error menggunakan `fmt.Errorf("...: %w", err)` agar konteks tidak hilang
2. **Jangan gunakan `_` untuk mengabaikan hasil penting**
   - Gunakan `_` hanya untuk nilai yang benar-benar tidak relevan
3. **Gunakan gofmt / go fmt / gofumpt untuk formatting**
   - Terapkan formatting otomatis sebagai pre-commit hook atau di CI
   - Gunakan `goimports` untuk merapikan imports
4. **Gunakan static analysis dan linters**
   - Jalankan `go vet`, `staticcheck`, dan `golangci-lint` di CI
   - Anggap temuan linter sebagai aturan, bukan rekomendasi
5. **Hindari `panic` di library**
   - Untuk paket yang bisa di-reuse, kembalikan error, jangan panggil `panic`
6. **Gunakan `context.Context` pada boundary transport/IO**
   - Terima `context.Context` sebagai parameter pertama pada fungsi yang melakukan I/O, request handling, atau operasi panjang
7. **Batasi penggunaan variabel global yang mutable**
   - Favor dependency injection dan konstruktor (`NewX`) untuk membuat instance
   - Global hanya untuk konstanta atau konfigurasi immutable
8. **Timeout dan cancellation eksplisit**
   - Untuk request jaringan atau DB, pastikan ada timeout atau gunakan context dengan deadline
9. **Dokumentasikan semua simbol yang diekspor**
   - Gunakan komentar di atas tipe, fungsi, dan variabel yang diekspor agar `godoc` dan tim lain paham intent
10. **Hindari magic numbers dan string**
    - Tempatkan konfigurasi dan konstanta di satu file konstanta atau di package `configs`
11. **Desain untuk testabilitas**
    - Program harus mudah diuji: gunakan interface untuk dependency (repo, clients) dan sediakan cara untuk inject mock pada tests
12. **Praktek keamanan dasar**
    - Jangan commit secrets
    - Validasi input, lakukan escaping output, dan gunakan library yang dipelihara untuk enkripsi/crypto
13. **CI wajib: build, test, lint, vet**
    - Pastikan pipeline CI melakukan `go test -race`, linter, dan build untuk mencegah regresi
14. **Kode harus idiomatik Go**
    - Ikuti Effective Go dan Go Code Review Comments
    - Nama pendek, error-first returns, tidak ada setter/getter berlebihan, gunakan slices/maps idiomatik
15. **Konsistensi penamaan**
    - Hindari underscore di identifier publik
    - Gunakan `CamelCase` untuk exported names dan `camelCase` untuk private names

---

## Penjelasan Makefile

Makefile memudahkan developer menjalankan tugas umum. Target yang biasa ada:

- `make build` : Compile binary (`go build ./...`)
- `make run` : Jalankan aplikasi (`go run ./cmd/main`)
- `make test` : Jalankan unit test (`go test ./...`)

---

## Penjelasan File .env

.env menyimpan variabel lingkungan lokal (development). Contoh:

```env
APP_ENV=development
PORT=8080
DB_DSN=user:pass@tcp(localhost:3306)/dbname
```

> **Jangan commit file `.env` berisi kredensial. Gunakan `.env.example` untuk contoh.**

Ketika menjalankan aplikasi, flag `--stage` digunakan untuk memilih file yang sesuai. Contoh Makefile menjalankan aplikasi dengan flag:

```bash
./cmd/$(BINARY_NAME) --mode http --stage prod
```

### Tips
- Buat file `.env.example` atau `.env.stage.example` yang tidak berisi kredensial tapi menunjukkan variabel yang diperlukan
- Jangan commit file `.env.stage.production` yang berisi secret; gunakan secret manager untuk produksi
- Pastikan nilai `--stage` pada Makefile konsisten dengan nama file `.env.stage.*` yang Anda pakai

---

## Mode dan Stage

`--mode` dan `--stage` adalah flag runtime yang umum dipakai untuk memilih cara aplikasi berjalan dan konfigurasi environment.

### 1. --mode
- Menentukan subsistem yang dijalankan oleh binary
- Nilai umum:
  - `http` : Jalankan HTTP server (biasanya Gin). Ekspos REST/HTTP API
  - `grpc` : Jalankan gRPC server (jika aplikasi mendukung gRPC)
  - `worker` : Jalankan worker/consumer untuk background jobs (queue consumer, worker loop)
  - `migrate` : Jalankan proses migrasi database dan keluar
  - `cli` / `task` : Jalankan tugas CLI tertentu atau runner task
- Setiap mode hanya menginisialisasi komponen yang diperlukan untuk fungsinya

### 2. --stage
- Memilih konfigurasi environment (lokal, development, staging, production)
- Contoh nilai: `local`, `development`, `staging`, `production` (atau singkatan jika konsisten, misal `prod`)
- Pengaruh: penentuan file env yang akan dimuat (`.env.stage.<stage>`), level logging default, koneksi ke resource berbeda (DB, queue), dan perilaku fitur (misal: feature flags)

### 3. Kombinasi & Precedence
- Mode menentukan subsistem; stage menentukan konfigurasi untuk subsistem tersebut
- Precedence konfigurasi: flag CLI (`--stage`) > environment variables > file konfigurasi

### 4. Praktik dan Rekomendasi
- Gunakan nama stage yang konsisten dan deskriptif (`production` lebih disarankan daripada `prod` kecuali ada mapping jelas)
- Pastikan Makefile dan dokumentasi memakai nama stage yang sama dengan file `.env.stage.*` di repo
- Mode harus orthogonal terhadap stage: bisa menjalankan `--mode http --stage development` maupun `--mode worker --stage production`

### 5. Contoh Perintah

```bash
# Jalankan HTTP server pada stage production
./cmd/main --mode=http --stage=production

# Jalankan migrasi pada stage staging
./cmd/main --mode=migrate --stage=staging

# Jalankan worker di environment development
./cmd/main --mode=worker --stage=development
```

> **Catatan:** Pastikan loader environment Anda mencari file `.env.stage.<stage>` atau memetakan singkatan stage ke nama file yang sesuai

---

## Penjelasan Config

Config didefinisikan sebagai struct (misal: `configs.Config`) dan di-load dari file + environment variables (prioritas env > file)

Contoh singkat:

```go
type Config struct {
  AppName string
  Port    int
  DBDSN   string
}
```

---

## Langkah Menjalankan Aplikasi Lokal

1. Copy file environment contoh dan sesuaikan:
   ```bash
   cp .env.example .env.stage.local
   # edit .env sesuai kebutuhan
   ```
2. Install dependensi dan build (Go modules):
   ```bash
   go mod tidy
   make build
   ```
3. Jalankan aplikasi:
   ```bash
   make run
   # atau
   go run ./cmd/main --mode=http --stage=local
   ```
4. Menjalankan unit test:
   ```bash
   make test
   # atau
   go test ./... -v -coverprofile=coverage.out
   ```


