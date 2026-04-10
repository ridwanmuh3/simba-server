# AI Assistant Instructions for SPPG MBG Project

## Project Context

Proyek ini adalah Sistem Informasi Manajemen Anggaran dan Barang untuk Satuan Pelayanan Pemenuhan Gizi (SPPG) Program Makan Bergizi Gratis (MBG).
Fokus utama: Manajemen keuangan (harian, per 10 hari, bulanan), manajemen pengguna (RBAC/Super Admin), kelola bahan baku (stok masuk/keluar, opname), dan visualisasi dashboard analitik.

## Global Rules

- Berikan respons yang ringkas, langsung pada solusi, dan hindari kata-kata penghibur.
- Tulis kode produksi yang bersih, efisien, dan memiliki penanganan error yang jelas.
- Asumsikan arsitektur monorepo dengan pemisahan direktori `/client` (Frontend) dan `/server` (Backend).
- Sistem harus terhindar dari serangan XSS, SQL Injection, dan serangan berbahaya lainnya

## Backend Guidelines (Golang `/server`)

- **Language & Framework:** Golang 1.25.5 dengan Fiber V2.
- **JSON Processing:** Wajib menggunakan `github.com/bytedance/sonic` untuk serialisasi/deserialisasi JSON, jangan gunakan `encoding/json` standar.
- **Database & ORM:** PostgreSQL 18 dengan GORM. Gunakan praktik terbaik GORM untuk relasi dan _eager loading_.
- **Migrations:** Gunakan AtlasHCL untuk skema dan migrasi database.
- **Validation:** Gunakan `github.com/go-playground/validator/v10` pada level struct (DTO/Request payloads).
- **Configuration:** Gunakan Viper untuk membaca variabel lingkungan dan konfigurasi.
- **Logging:** Implementasikan Zap Logger terstruktur di seluruh _layer_ aplikasi (Handler, Usecase, Repository).
- **Architecture:** Gunakan pola desain _Clean Architecture_ (Handler/Controller -> Service/Usecase -> Repository).

## Frontend Guidelines (React `/client`)

- **Framework:** React (Vite/CRA) dengan React Router untuk navigasi.
- **Styling & UI Components:** Gunakan TailwindCSS. Prioritaskan penggunaan komponen Shadcn UI yang dapat dikustomisasi, dan ikon dari Lucide React.
- **State Management & Data Fetching:** Gunakan Tanstack React Query terintegrasi dengan Axios untuk semua panggilan API. Pisahkan logika _fetching_ ke dalam _custom hooks_ (misal: `useGetIngredients`, `useCreateTransaction`).
- **Data Visualization:** Gunakan Recharts untuk komponen Dashboard (Pie chart alokasi pembelanjaan, Bar/Line chart grafik pemasukan/pengeluaran).
- **Component Structure:** Gunakan _Functional Components_ dengan pola desain _composition_. Pisahkan _smart components_ (yang mengelola data/state) dan _dumb components_ (hanya untuk presentasi UI).

## Specific Feature Implementations

- **Manajemen Keuangan:** Pastikan kalkulasi agregasi harian, 10-harian, dan bulanan ditangani secara presisi di level backend (menggunakan fungsi agregasi SQL GORM) untuk mengurangi beban klien.
- **Kelola Bahan:** Logika _inventory_ harus menerapkan _transaction database_ saat melakukan update stok masuk/keluar dan _stock opname_ untuk mencegah _race conditions_.

## Formatting & Naming Conventions

- **Go:** Gunakan _camelCase_ untuk variabel lokal, _PascalCase_ untuk struct/method yang diekspor. Nama file gunakan _snake_case_.
- **React:** Gunakan _PascalCase_ untuk nama file komponen (`Dashboard.jsx/tsx`), _camelCase_ untuk fungsi dan _hooks_ (`useAuth.js`).
