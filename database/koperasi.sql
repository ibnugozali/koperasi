-- Hapus tabel jika sudah ada untuk memastikan skrip bisa dijalankan ulang
DROP TABLE IF EXISTS angsuran CASCADE;
DROP TABLE IF EXISTS pinjaman CASCADE;
DROP TABLE IF EXISTS detail CASCADE;
DROP TABLE IF EXISTS simpanan CASCADE;
DROP TABLE IF EXISTS pengelola CASCADE;
DROP TABLE IF EXISTS anggota CASCADE;
DROP TABLE IF EXISTS halaman CASCADE;

-- =================================================================
-- BAGIAN 1: PEMBUATAN STRUKTUR TABEL
-- =================================================================

CREATE TABLE anggota (
    id_anggota SERIAL PRIMARY KEY,
    nama_anggota VARCHAR(50) NOT NULL,
    username VARCHAR(25) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    tgl_lahir DATE,
    nik_ktp VARCHAR(25),
    no_telepon VARCHAR(20),
    tgl_gabung DATE DEFAULT CURRENT_DATE,
    provinsi VARCHAR(50),
    jenis_kelamin VARCHAR(12),
    status VARCHAR(25) DEFAULT 'pending' CHECK (status IN ('aktif', 'nonaktif', 'pending')),
    kode_anggota VARCHAR(50) UNIQUE
);

CREATE TABLE pengelola (
    id_pengelola SERIAL PRIMARY KEY,
    nama_pengelola VARCHAR(50) NOT NULL,
    username VARCHAR(25) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    jabatan_koperasi VARCHAR(25),
    no_telepon VARCHAR(20),
    email VARCHAR(50),
    tgl_gabung DATE DEFAULT CURRENT_DATE,
    level VARCHAR(25) CHECK (level IN ('admin', 'bendahara', 'ketua', 'pembina')),
    status VARCHAR(25) DEFAULT 'aktif' CHECK (status IN ('aktif', 'nonaktif'))
);

CREATE TABLE simpanan (
    id_simpanan SERIAL PRIMARY KEY,
    jenis_simpanan VARCHAR(50) UNIQUE NOT NULL
);

-- PERBAIKAN: Menggunakan ON DELETE SET NULL untuk id_pengelola agar riwayat transaksi tidak hilang
-- jika seorang pengelola dihapus dari sistem.
CREATE TABLE detail (
    id_detail SERIAL PRIMARY KEY,
    id_anggota INT REFERENCES anggota(id_anggota) ON DELETE CASCADE,
    id_simpanan INT REFERENCES simpanan(id_simpanan) ON DELETE CASCADE,
    id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL, -- Diubah
    tgl_transaksi DATE DEFAULT CURRENT_DATE,
    jumlah_simpanan NUMERIC(15,2),
    total_simpanan NUMERIC(15,2)
);

CREATE TABLE pinjaman (
    id_pinjaman SERIAL PRIMARY KEY,
    id_anggota INT REFERENCES anggota(id_anggota) ON DELETE CASCADE,
    id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL, -- Diubah
    tgl_pinjaman DATE DEFAULT CURRENT_DATE,
    jumlah_pinjaman NUMERIC(15,2),
    jangka_waktu INT, -- Dalam bulan
    bunga NUMERIC(5,2) DEFAULT 2.0,
    status VARCHAR(25) CHECK (status IN ('proses', 'aktif', 'lunas', 'gagal'))
);

CREATE TABLE angsuran (
    id_angsuran SERIAL PRIMARY KEY,
    id_pinjaman INT REFERENCES pinjaman(id_pinjaman) ON DELETE CASCADE,
    id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL, -- Diubah
    tgl_bayar DATE DEFAULT CURRENT_DATE,
    sisa_pinjaman NUMERIC(15,2),
    status_angsuran VARCHAR(25) CHECK (status_angsuran IN ('belum_lunas', 'lunas', 'terlambat')),
    bukti_angsuran BYTEA,
    status VARCHAR(25) DEFAULT 'valid' CHECK (status IN ('valid', 'invalid'))
);

-- PERBAIKAN: Menambahkan kolom 'kategori' agar mudah memfilter halaman
-- antara 'tentang' dan 'pelayanan' di dalam kode Go.
CREATE TABLE halaman (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(50) UNIQUE NOT NULL,
    judul VARCHAR(100) NOT NULL,
    kategori VARCHAR(50), -- Ditambahkan
    konten TEXT, -- Akan diisi dengan format JSON
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =================================================================
-- BAGIAN 2: PENGISIAN DATA AWAL (SEEDING)
-- =================================================================

-- Menambahkan data admin awal (password: admin12)
INSERT INTO pengelola (nama_pengelola, username, password, jabatan_koperasi, level, status)
VALUES (
    'Administrator Utama',
    'admin',
    '$2a$12$u7jMyMxTcDvt7hfzRTbK9eO9VRfxJHL1ztd0AWSr/p5HLuuB89hMG',
    'Admin',
    'admin',
    'aktif'
);

-- Mengisi data awal untuk jenis simpanan
INSERT INTO simpanan (jenis_simpanan) VALUES
('Simpanan');

-- Mengisi data awal untuk halaman statis dengan format JSON
INSERT INTO halaman (slug, judul, kategori, konten) VALUES
('sejarah', 'Sejarah Koperasi', 'tentang',
  '{
    "teks": "Tulis konten lengkap sejarah di sini...",
    "gambar": "/static/images/placeholder.png"
  }'
),
('visi-misi', 'Visi & Misi', 'tentang',
  '{
    "visi": "Menjadi koperasi yang mandiri, andal, dan terpercaya.",
    "misi": "Meningkatkan kesejahteraan anggota melalui usaha yang inovatif.",
    "gambar1": "/static/images/placeholder.png",
    "gambar2": "/static/images/placeholder.png"
  }'
),
('struktur', 'Struktur Organisasi', 'tentang',
  '{
    "deskripsi": "Berikut adalah struktur organisasi Koperasi Wirya periode 2024-2026.",
    "gambar_struktur": "/static/images/placeholder.png"
  }'
),
('pinjaman', 'Pinjaman Anggota', 'pelayanan',
  '{
    "teks": "Kami menyediakan layanan pinjaman untuk anggota dengan bunga yang kompetitif dan proses yang mudah. Layanan ini bertujuan untuk membantu anggota dalam memenuhi kebutuhan finansial mendesak maupun untuk modal usaha.",
    "gambar": "/static/images/placeholder.png"
  }'
),
('simpanan', 'Simpanan', 'pelayanan',
  '{
    "teks": "Simpanan adalah dana yang disetor oleh anggota kepada koperasi sebagai bentuk partisipasi dan investasi. Simpanan ini dapat digunakan untuk berbagai keperluan anggota dan akan mendapatkan bagi hasil sesuai dengan ketentuan koperasi.",
    "gambar": "/static/images/placeholder.png"
  }'
),
('angsuran', 'Pembayaran Angsuran', 'pelayanan',
  '{
    "teks": "Halaman ini berisi informasi mengenai tata cara pembayaran angsuran pinjaman. Pastikan untuk melakukan pembayaran tepat waktu untuk menghindari denda dan menjaga riwayat kredit Anda tetap baik di koperasi.",
    "gambar": "/static/images/placeholder.png"
  }'
),
('dashboard_anggota', 'Dashboard Anggota', 'dashboard',
  '{
    "teks": "Selamat datang di dashboard anggota Koperasi Wirya. Di sini Anda dapat mengakses berbagai layanan simpanan dan pinjaman yang disediakan oleh koperasi.",
    "gambar": "/static/images/placeholder.png",
    "welcome": "Selamat Datang di Koperasi Wirya",
    "slogan": "Dari Anggota, Oleh Anggota, dan Untuk Anggota"
  }'
);

-- =================================================================
-- BAGIAN 3: PEMBUATAN INDEX UNTUK OPTIMASI PERFORMA
-- =================================================================

CREATE INDEX idx_anggota_username ON anggota(username);
CREATE INDEX idx_pengelola_username ON pengelola(username);
CREATE INDEX idx_pinjaman_anggota ON pinjaman(id_anggota);
CREATE INDEX idx_angsuran_pinjaman ON angsuran(id_pinjaman);
CREATE INDEX idx_detail_anggota ON detail(id_anggota);
CREATE INDEX idx_detail_simpanan ON detail(id_simpanan);
CREATE INDEX idx_halaman_slug ON halaman(slug);

-- Add new columns for status_anggota and fakultas
ALTER TABLE anggota ADD COLUMN status_anggota VARCHAR(50);
ALTER TABLE anggota ADD COLUMN fakultas VARCHAR(100);

-- Update sejarah content
UPDATE halaman SET konten = '{
  "teks": "Koperasi adalah bentuk organisasi ekonomi yang didirikan oleh masyarakat untuk memenuhi kebutuhan bersama. Gerakan koperasi modern dimulai di Eropa pada abad ke-19, dipelopori oleh tokoh-tokoh seperti Robert Owen di Inggris dan Charles Fourier di Prancis. Mereka melihat koperasi sebagai alternatif terhadap kapitalisme yang eksploitatif.\n\nDi Indonesia, perkembangan koperasi dimulai pada masa kolonial Belanda. Pada tahun 1896, Raden Aria Wiriatmadja mendirikan Koperasi Kredit pertama di Purwokerto, Jawa Tengah. Gerakan ini semakin berkembang setelah Indonesia merdeka, dengan dukungan pemerintah untuk membangun ekonomi rakyat.\n\nPada tahun 1967, pemerintah mengeluarkan Undang-Undang No. 12 Tahun 1967 tentang Pokok-Pokok Perkoperasian. Undang-undang ini kemudian diganti dengan Undang-Undang No. 25 Tahun 1992 tentang Perkoperasian, yang menjadi dasar hukum koperasi di Indonesia hingga saat ini.\n\nKoperasi di Indonesia berperan penting dalam berbagai sektor, termasuk simpan pinjam, pertanian, konsumsi, dan produksi. Prinsip-prinsip koperasi seperti keanggotaan sukarela, pengelolaan demokratis, dan pembagian hasil secara adil menjadi landasan operasional koperasi.",
  "gambar": "/static/images/placeholder.png"
}' WHERE slug = 'sejarah';
