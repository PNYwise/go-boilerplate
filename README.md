# Go Boilerplate - Simplified & Seamless

## Deskripsi Singkat

Template boilerplate untuk layanan backend menggunakan Go dengan arsitektur **seamless dan ultra-sederhana**. Fokus ke single transport (HTTP by default) dengan mudah switching ke gRPC/RabbitMQ saat dibutuhkan. 95% development terjadi di satu file: `internal/apps/wire.go`.

---

## Struktur Proyek

### Root Files
- **go.mod, go.sum**: Dependency dan versi module
- **Makefile**: Shortcut tugas developer (build, run, test, migrate)
- **README.md**: Dokumentasi proyek

### Direktori Utama
- **cmd/**: Entrypoint aplikasi (DO NOT MODIFY). Bootstrap generik yang handle semua transport modes.
- **internal/**: Kode aplikasi yang tidak diekspor ke package lain. Struktur internal merepresentasikan boundary arsitektur aplikasi.
  - **apps/**: 
    - **wire.go** - 🎯 **PRIMARY FILE** - Semua customization development di sini
    - **app.go** - Core lifecycle management (DO NOT MODIFY)
  - **configs/**: Definisi struct konfigurasi, loader seamless tanpa mode switching
  - **dbs/**: Inisialisasi koneksi database, pool, helper transaksi, dan health checks DB
  - **dtos/**: Transport-level data shapes (request/response DTO), dipisah dari domain entities
  - **entities/**: Domain models dan value objects
  - **repositories/**: Definisi interface repository dan implementasi penyimpanan/data access (SQL/NoSQL/cache/RestApi). Pisahkan interface dan implementasi untuk memudahkan mocking
  - **services/**: Use-cases / business logic yang mengorkestrasi repositori dan external clients
  - **transports/**: Adapter transport (HTTP by default, easily switchable)
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
- **wire.go**: 🎯 **DEVELOPER ZONE** - Semua customization (repositories, services, handlers) diatur di sini
- **app.go**: Core lifecycle management, panic recovery, resource cleanup (DO NOT MODIFY)
- Mengatur dependency injection dengan Wire
- Aliran: wire.go configuration → automatic dependency injection → application ready

### Seamless Development Workflow
**95% development terjadi di `wire.go`:**
```go
// Tambah repository
var RepositoryProviders = wire.NewSet(
    repositories.NewUserRepository,
    repositories.NewProductRepository,  // ← Tambah di sini
)

// Tambah service  
var ServiceProviders = wire.NewSet(
    services.NewUserService,
    services.NewProductService,  // ← Tambah di sini
)

// Tambah handler
var HandlerProviders = wire.NewSet(
    httphandlers.NewUserHandler,
    httphandlers.NewProductHandler,  // ← Tambah di sini
)
```

### Transports (HTTP) — Router & Server (`transports/http`)
- Menerima koneksi masuk, routing, dan lifecycle server
- Aliran request: masuk → middleware → handler

---

## Environment & Stage Configuration (Seamless)

### File .env
.env files menyimpan variabel lingkungan; **jangan commit file yang berisi kredensial**. Simpan contoh variabel di `.env.example` atau `.env.stage.example`.

#### File env per stage
- Gunakan file terpisah per stage untuk memudahkan deployment dan pengujian:
  - `.env.stage.local`
  - `.env.stage.development` 
  - `.env.stage.staging`
  - `.env.stage.production`
- Pilih konvensi nama yang konsisten di seluruh tim

#### Flag --stage
- Memilih konfigurasi environment (local, development, staging, production)
- Menentukan file env yang akan dimuat, level logging default, koneksi ke resource (DB/queue), dan perilaku fitur

**Running aplikasi:**
```bash
# Default (no config needed)
go run ./cmd

# With stage
go run ./cmd -stage=dev
go run ./cmd -stage=staging 
go run ./cmd -stage=prod
```

#### Transport Switching (Wire-based, No Mode Flags)
**Untuk switch transport (HTTP → gRPC), hanya edit `wire.go`:**
```go
// Step 1: Ganti handlers
var HandlerProviders = wire.NewSet(
    grpchandlers.NewUserHandler,     // HTTP → gRPC
)

// Step 2: Ganti transport
var TransportProvider = grpctransport.NewGRPCServer  // HTTP → gRPC

// Step 3: Update Application struct  
type Application struct {
    Server *grpctransport.Server  // Change type
}
```

#### Loader & Precedence
- Precedence konfigurasi: flag CLI (`--stage`) > environment variables > file konfigurasi  
- Loader membentuk nama file berdasarkan nilai `--stage` (misal: `.env.stage.production`) dan memuatnya pada startup

#### Rekomendasi Operasional
- Pastikan `Makefile` dan dokumentasi memakai nama stage yang sama dengan file `.env.stage.*`
- Transport mode ditentukan otomatis dari wire.go configuration - seamless switching
- Jangan commit `.env.stage.production` yang berisi secret; gunakan secret manager untuk produksi
- Log level harus sesuai: `Info` untuk alur normal, `Error` untuk kegagalan

---

## Dependency Injection dengan Wire (Architecture)

Proyek ini menggunakan [Google Wire](https://github.com/google/wire) untuk dependency injection yang aman dan performant.

### Keuntungan Wire
- **Compile-time safety**: Semua dependensi diresolve saat compile time
- **Zero runtime overhead**: Tidak ada reflection atau service locator
- **Explicit dependencies**: Dependency chain yang jelas dan mudah di-trace
- **Easy testing**: Mudah untuk inject mock dependencies

### Wire Structure (Simplified)
```
internal/apps/
├── wire.go          # PRIMARY FILE untuk semua provider configuration
└── wire_gen.go      # Kode yang di-generate otomatis (JANGAN EDIT)
```

### Provider Sets (Seamless)
- **InfrastructureProviders**: Database, logging, utilities (foundation)
- **RepositoryProviders**: All repository implementations
- **ServiceProviders**: Business logic services
- **HandlerProviders**: Handler layer (http/grpc/etc)
- **TransportProvider**: Single variable untuk switch transport

### Menambah Fitur - Edit wire.go Saja

#### 1. Tambah Repository:
```go
var RepositoryProviders = wire.NewSet(
    repositories.NewExampleRepository,
    repositories.NewUserRepository,
    repositories.NewProductRepository,  // ← Add this
)
```

#### 2. Tambah Service:
```go
var ServiceProviders = wire.NewSet(
    services.NewExampleService,
    services.NewHealthService,
    services.NewProductService,  // ← Add this
)
```

#### 3. Tambah Handler:
```go
var HandlerProviders = wire.NewSet(
    handlers.NewExampleHandler,
    handlers.NewHealthHandler,
    handlers.NewProductHandler,  // ← Add this  
)
```

#### 4. Generate Wire:
```bash
make wire  # Single command untuk regenerate semua
```

### Transport Switching (No Config Changes!)
```go
// Edit hanya wire.go untuk switch HTTP → gRPC
var TransportProvider = grpctransport.NewGRPCServer  // was: httptransport.NewGinServer
```

---

## Quick Commands

```bash
# Development
go run ./cmd                    # Default
go run ./cmd -stage=dev         # With stage

# Testing
go test ./...                   # All tests
go test -cover ./...            # With coverage

# Build & Deploy
make build                      # Compile
./tmp/main -stage=prod          # Run production

# Wire generation
make wire                       # After adding features
```

### Testing Guidelines
- File test: `xxx_test.go`
- Mock external resources (DB, network)
- Table-driven tests preferred

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

## Development & Deployment

### Quick Start
```bash
# Default development
go run ./cmd

# Production stage
go run ./cmd -stage=prod
```

### Transport Switching (Wire-based)
**Ubah transport hanya di `internal/apps/wire.go`:**
```go
// HTTP (default)  
var TransportProvider = httptransport.NewGinServer

// gRPC
var TransportProvider = grpctransport.NewGRPCServer  

// Worker/Queue  
var TransportProvider = workertransport.NewWorker
```

**Jangan commit secrets!**

