# SIMBA - System Update Roadmap

Dokumen ini berisi daftar pembaruan sistem yang direncanakan, dikategorikan berdasarkan skala perubahan.

---

## Minor Updates (Bug Fix & Small Improvements)

### Security Hardening

- [ ] **Rate Limiting pada Auth Endpoints** — Tambahkan rate limiter (misal: 5 request/menit) pada `/api/auth/login` untuk mencegah brute-force attack
- [ ] **CSRF Token Protection** — Implementasikan CSRF token pada semua state-changing request (POST/PUT/DELETE) menggunakan Fiber CSRF middleware
- [ ] **Password Complexity Rules** — Tambahkan validasi kompleksitas password (minimal 1 huruf besar, 1 angka, 1 karakter spesial, minimum 8 karakter)
- [x] **Token TTL & Rotation** — Implementasikan expiry time pada token di database, bukan hanya di cookie. Tambahkan mekanisme token refresh otomatis

### Data & Validation

- [x] **CSV Import Row Limit** — Dibatasi max 500 baris untuk mencegah DoS (v1.0.0)
- [ ] **CSV Import Validation** — Perkuat validasi: deteksi duplikat nama bahan, validasi format angka
- [ ] **Date Format Validation** — Standarisasi validasi format tanggal pada semua endpoint yang menerima parameter date (gunakan `datetime=2006-01-02T15:04:05Z07:00`)
- [ ] **Pagination Default & Guard** — Pastikan semua endpoint list memiliki default `page=1, size=10` dan max `size=100` untuk mencegah query berlebihan

### UX Improvements

- [ ] **Loading State pada Stock Opname** — Tambahkan skeleton loader saat data stock opname sedang di-fetch
- [ ] **Error Toast yang Informatif** — Ganti pesan error generik dengan pesan yang lebih spesifik dari backend response
- [ ] **Konfirmasi Sebelum Navigasi** — Tambahkan prompt konfirmasi jika user meninggalkan form yang belum disimpan (unsaved changes)
- [ ] **Responsive Table** — Optimasi tabel pada tampilan mobile agar bisa di-scroll horizontal dengan indikator

---

## Major Updates (Significant Feature Enhancements)

### Audit & Logging

- [ ] **Full Audit Trail** — Ganti sistem activity log (yang saat ini hanya simpan 4 record terakhir) dengan audit log permanen yang mencatat semua operasi CRUD beserta user, timestamp, dan detail perubahan (before/after)
- [ ] **Login History** — Catat riwayat login/logout per user (IP address, user agent, timestamp) untuk keperluan monitoring keamanan

### Reporting & Export

- [ ] **Laporan Keuangan PDF** — Generate laporan keuangan harian, per 10 hari, dan bulanan dalam format PDF dengan breakdown per kategori
- [ ] **Laporan Stock Opname PDF** — Export hasil stock opname ke PDF termasuk selisih stok fisik vs sistem
- [ ] **Dashboard Export** — Kemampuan export seluruh data dashboard (chart + statistik) ke PDF atau Excel

### Inventory Enhancement

- [ ] **Low Stock Alert** — Sistem peringatan otomatis ketika stok bahan di bawah threshold minimum (konfigurasikan per item)
- [ ] **Stock History Graph** — Visualisasi grafik pergerakan stok per item (trend line stok masuk/keluar per waktu) di halaman detail item
- [ ] **Batch & Expiry Tracking** — Tambahkan tracking nomor batch dan tanggal kadaluarsa untuk setiap stok masuk, dengan peringatan bahan mendekati expired
- [ ] **Unit Conversion** — Dukungan konversi antar satuan (kg <-> gram, liter <-> ml) untuk fleksibilitas pencatatan

### Finance Enhancement

- [ ] **Budget Planning** — Fitur perencanaan anggaran bulanan dengan perbandingan realisasi vs rencana
- [ ] **Multi-proof Upload** — Izinkan upload lebih dari satu bukti transaksi per record keuangan
- [ ] **Approval Workflow** — Transaksi keuangan di atas nominal tertentu memerlukan approval dari Super Admin sebelum tercatat

### User Management

- [ ] **Profile & Avatar** — Halaman profil user dengan kemampuan upload foto dan ubah data pribadi
- [ ] **Session Management** — User bisa melihat dan menghentikan sesi aktif di perangkat lain
- [ ] **Granular Permissions** — RBAC yang lebih detail: kontrol akses per fitur (bukan hanya per role), misal Admin A hanya bisa akses keuangan, Admin B hanya kelola bahan

---

## New Features (Tambahan Fungsionalitas Baru)

### Supplier Management Module

- [ ] **Master Data Supplier** — CRUD data supplier (nama, alamat, kontak, kategori barang, rating)
- [ ] **Supplier Linkage** — Hubungkan setiap transaksi stok masuk dengan data supplier yang terdaftar
- [ ] **Purchase Order** — Generate PO otomatis dari sistem ke supplier terdaftar berdasarkan kebutuhan stok

### Menu Planning Module

- [ ] **Master Menu/Resep** — Database resep makanan dengan daftar bahan dan takaran per porsi
- [ ] **Kalkulasi Kebutuhan** — Hitung otomatis kebutuhan bahan baku berdasarkan jumlah porsi yang direncanakan
- [ ] **Cost per Serving** — Kalkulasi biaya per porsi makanan berdasarkan harga bahan aktual

### Notification System

- [ ] **In-App Notifications** — Notifikasi real-time dalam aplikasi (stok rendah, transaksi baru, approval pending)
- [ ] **Email/Webhook Integration** — Kirim notifikasi ke email atau webhook untuk alert kritis (stok habis, anggaran melebihi batas)

### Analytics & Forecasting

- [ ] **Trend Analysis** — Analisis tren pengeluaran dan konsumsi bahan per periode (mingguan, bulanan, kuartalan)
- [ ] **Demand Forecasting** — Prediksi kebutuhan bahan baku berdasarkan data historis konsumsi
- [ ] **Cost Optimization** — Rekomendasi penghematan berdasarkan perbandingan harga antar supplier dan pola pembelian
- [ ] **Custom Dashboard Widgets** — User bisa mengatur widget dashboard sesuai kebutuhan (drag & drop)

### Multi-Location Support

- [ ] **Multi-SPPG** — Dukungan pengelolaan beberapa unit SPPG dalam satu sistem
- [ ] **Inter-Location Transfer** — Fitur transfer stok antar lokasi SPPG
- [ ] **Consolidated Report** — Laporan gabungan dari seluruh lokasi untuk level manajemen

### Data Integrity & Backup

- [ ] **Database Backup Scheduler** — Backup otomatis database secara berkala (harian/mingguan) dengan retensi yang dapat dikonfigurasi
- [ ] **Data Import/Export Full** — Export dan import seluruh data sistem untuk migrasi atau backup manual
- [ ] **Recycle Bin** — Soft-deleted items bisa dipulihkan dalam jangka waktu tertentu (misal: 30 hari) sebelum dihapus permanen

---

## Priority Matrix

| Priority | Update                  | Category | Impact       |
| -------- | ----------------------- | -------- | ------------ |
| P0       | Rate Limiting Auth      | Minor    | Security     |
| P0       | CSRF Protection         | Minor    | Security     |
| P0       | Password Complexity     | Minor    | Security     |
| P0       | Token TTL               | Minor    | Security     |
| P1       | Full Audit Trail        | Major    | Compliance   |
| P1       | Laporan Keuangan PDF    | Major    | Business     |
| P1       | Low Stock Alert         | Major    | Operations   |
| P1       | Budget Planning         | Major    | Business     |
| P2       | Supplier Management     | New      | Operations   |
| P2       | Menu Planning           | New      | Operations   |
| P2       | Notification System     | New      | UX           |
| P3       | Analytics & Forecasting | New      | Intelligence |
| P3       | Multi-Location          | New      | Scalability  |
