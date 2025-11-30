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

-- Tabel anggota koperasi
-- Format ID Anggota: {unit_kerja}{fakultas_code}{tahun}{nomor_urut}
-- Contoh: 010120250001
-- - 01: Unit Kerja (01=Dosen, 02=Karyawan/Staff, 03=Mahasiswa)
-- - 01: Fakultas (01=FAI, 02=FE, 03=FH, 04=FISIP, 05=FKIP, 06=FKM, 07=FAPERTA, 08=FT, 09=Rektorat/Yayasan/Staff)
-- - 2025: Tahun konfirmasi oleh bendahara
-- - 0001: Nomor urut anggota (4 digit)
CREATE TABLE anggota (
    id_anggota VARCHAR(50) PRIMARY KEY,
    nama_anggota VARCHAR(50) NOT NULL,
    username VARCHAR(25) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    tgl_lahir DATE,
    nik_ktp VARCHAR(25),
    no_telepon VARCHAR(20),
    tgl_gabung DATE DEFAULT CURRENT_DATE,
    alamat VARCHAR(50),
    provinsi VARCHAR(50),
    jenis_kelamin VARCHAR(12),
    status VARCHAR(25) DEFAULT 'pending' CHECK (status IN ('aktif', 'nonaktif', 'pending')),
    status_anggota VARCHAR(50),
    fakultas VARCHAR(100),
    unit_kerja VARCHAR(2),  -- 01=Dosen, 02=Karyawan/Staff, 03=Mahasiswa
    fakultas_code VARCHAR(2),  -- 01=FAI, 02=FE, 03=FH, 04=FISIP, 05=FKIP, 06=FKM, 07=FAPERTA, 08=FT, 09=Rektorat/Yayasan/Staff
    tahun VARCHAR(4),  -- Tahun konfirmasi
    nomor_urut VARCHAR(4),  -- Nomor urut 4 digit
    bukti_transfer VARCHAR(255)
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
    id_anggota VARCHAR(50) REFERENCES anggota(id_anggota) ON DELETE CASCADE,
    id_simpanan INT REFERENCES simpanan(id_simpanan) ON DELETE CASCADE,
    id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL, -- Diubah
    tgl_transaksi DATE DEFAULT CURRENT_DATE,
    jumlah_simpanan NUMERIC(15,2),
    total_simpanan NUMERIC(15,2)
);

CREATE TABLE pinjaman (
    id_pinjaman SERIAL PRIMARY KEY,
    id_anggota VARCHAR(50) REFERENCES anggota(id_anggota) ON DELETE CASCADE,
    id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL, -- Diubah
    tgl_pinjaman DATE DEFAULT CURRENT_DATE,
    jumlah_pinjaman NUMERIC(15,2),
    jangka_waktu INT, -- Dalam bulan
    bunga NUMERIC(5,2) DEFAULT 2.0,
    status VARCHAR(25) CHECK (status IN ('proses', 'aktif', 'lunas', 'gagal'))
);

CREATE TABLE IF NOT EXISTS angsuran (
  id_angsuran SERIAL PRIMARY KEY,
  id_pinjaman INT REFERENCES pinjaman(id_pinjaman) ON DELETE CASCADE,
  id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL,
  tgl_bayar TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  sisa_pinjaman NUMERIC(15,2),
  status_angsuran VARCHAR(25) CHECK (status_angsuran IN ('belum_lunas', 'lunas', 'terlambat')),
  bukti_angsuran VARCHAR(255),
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

-- Tabel pesan untuk komunikasi dengan anggota
DROP TABLE IF EXISTS pesan CASCADE;
CREATE TABLE pesan (
    id_pesan SERIAL PRIMARY KEY,
    id_anggota VARCHAR(50) REFERENCES anggota(id_anggota) ON DELETE CASCADE,
    judul VARCHAR(100) NOT NULL,
    isi TEXT NOT NULL,
    tgl_kirim TIMESTAMPTZ DEFAULT NOW(),
    status VARCHAR(25) DEFAULT 'unread' CHECK (status IN ('read', 'unread'))
);

-- Tabel pengambilan simpanan untuk pengajuan penarikan simpanan oleh anggota
DROP TABLE IF EXISTS pengambilan_simpanan CASCADE;
CREATE TABLE pengambilan_simpanan (
    id_pengambilan SERIAL PRIMARY KEY,
    id_anggota VARCHAR(50) REFERENCES anggota(id_anggota) ON DELETE CASCADE,
    id_simpanan INT REFERENCES simpanan(id_simpanan) ON DELETE CASCADE,
    jumlah NUMERIC(15,2) NOT NULL,
    alasan TEXT,
    tgl_pengajuan TIMESTAMPTZ DEFAULT NOW(),
    tgl_proses TIMESTAMPTZ,
    status VARCHAR(25) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    catatan_bendahara TEXT,
    id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL
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

-- Menambahkan data bendahara awal (password: bendahara12)
INSERT INTO pengelola (nama_pengelola, username, password, jabatan_koperasi, level, status)
VALUES (
    'Bendahara Utama',
    'bendahara',
    '$2a$10$jemm9k/ZA0AEUt48CuwFU.7uePc6e.Gk3PYeDxTlgpKijh40Z.71m',
    'Bendahara',
    'bendahara',
    'aktif'
);

-- Menambahkan data ketua awal (password: ketua12)
INSERT INTO pengelola (nama_pengelola, username, password, jabatan_koperasi, level, status)
VALUES (
    'Ketua Utama',
    'ketua',
    '$2a$10$S50Wpfxf1Rq4gha9tqCxk.C6yMaASJERd9HgMdwvYKLAAZH0tBgbS',
    'Ketua',
    'ketua',
    'aktif'
);

-- Mengisi data awal untuk jenis simpanan
INSERT INTO simpanan (jenis_simpanan) VALUES
('pokok'),
('wajib'),
('sukarela'),
('hari_raya');

-- Mengisi data awal untuk halaman statis dengan format JSON
INSERT INTO halaman (slug, judul, kategori, konten) VALUES
('simpanan', 'Simpanan', 'pelayanan',
  '{
    "judul": "Simpanan",
    "jenis_simpanan": [
      {
        "nama": "Simpanan Pokok",
        "deskripsi": "Simpanan wajib yang dibayarkan saat menjadi anggota"
      },
      {
        "nama": "Simpanan Wajib",
        "deskripsi": "Simpanan rutin bulanan yang harus dibayarkan anggota"
      },
      {
        "nama": "Simpanan Sukarela",
        "deskripsi": "Simpanan tambahan yang dapat dibayarkan kapan saja"
      },
      {
        "nama": "Simpanan Hari Raya",
        "deskripsi": "Simpanan yang ditujukan untuk mempersiapkan kebutuhan pada saat hari raya"
      }
    ],
    "manfaat": [
      "Mendapatkan bagi hasil sesuai ketentuan koperasi",
      "Dapat digunakan sebagai modal pinjaman",
      "Meningkatkan kesejahteraan anggota",
      "Investasi yang aman dan terpercaya"
    ]
  }'
),
('dashboard_anggota', 'Dashboard Anggota', 'dashboard',
  '{
    "teks": "Selamat datang di dashboard anggota Koperasi Wirya. Di sini Anda dapat mengakses berbagai layanan simpanan dan pinjaman yang disediakan oleh koperasi.",
    "gambar": "/static/images/placeholder.png",
    "welcome": "Selamat Datang di Koperasi Wirya",
    "slogan": "Dari Anggota, Oleh Anggota, dan Untuk Anggota"
  }'
),
('pinjaman', 'Pinjaman', 'pelayanan',
  '{
    "judul": "Pinjaman",
    "deskripsi": "Layanan pinjaman untuk membantu memenuhi kebutuhan finansial anggota dengan bunga yang kompetitif",
    "syarat": [
      "Sudah menjadi anggota aktif koperasi",
      "Memiliki simpanan pokok dan wajib",
      "Menyerahkan fotokopi KTP dan KK",
      "Mengisi formulir pengajuan pinjaman",
      "Melampirkan slip gaji atau surat keterangan penghasilan"
    ],
    "manfaat": [
      "Proses pengajuan yang mudah dan cepat",
      "Bunga kompetitif",
      "Jangka waktu pembayaran yang fleksibel",
      "Tanpa agunan untuk nominal tertentu"
    ]
  }'
),
('angsuran', 'Angsuran', 'pelayanan',
  '{
    "judul": "Angsuran",
    "deskripsi": "Sistem pembayaran angsuran pinjaman yang fleksibel dan mudah",
    "cara_bayar": [
      "Transfer ke rekening koperasi",
      "Pembayaran langsung di kantor koperasi",
      "Potong gaji (untuk yang bekerja di institusi kerjasama)"
    ],
    "ketentuan": [
      "Angsuran dibayarkan setiap bulan",
      "Denda keterlambatan 0.5% per hari",
      "Dapat melakukan pelunasan dipercepat tanpa penalti",
      "Pembayaran dapat dilakukan melalui berbagai metode"
    ]
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
ALTER TABLE anggota ADD COLUMN IF NOT EXISTS status_anggota VARCHAR(50);
ALTER TABLE anggota ADD COLUMN IF NOT EXISTS fakultas VARCHAR(100);
-- Note: provinsi column already exists in the CREATE TABLE statement above

-- Mengisi data awal untuk halaman struktur
INSERT INTO halaman (slug, judul, kategori, konten) VALUES
('struktur', 'Struktur Organisasi', 'tentang',
  '{
    "deskripsi": "Koperasi Wirya memiliki struktur organisasi yang terdiri dari berbagai jabatan penting yang saling mendukung untuk mencapai tujuan bersama. Struktur ini memastikan pengelolaan yang efektif dan demokratis sesuai dengan prinsip-prinsip koperasi.",
    "gambar_struktur": "/static/images/placeholder.png"
  }'
),
('sejarah', 'Sejarah Koperasi', 'tentang',
  '{
    "timeline": [
      {
        "title": "Pendirian Koperasi (2020)",
        "marker": "bg-primary",
        "text": "Koperasi Wirya didirikan pada tahun 2020 oleh sekelompok masyarakat yang peduli dengan pemberdayaan ekonomi lokal. Dengan semangat kebersamaan, koperasi ini mulai beroperasi dengan modal awal dari kontribusi anggota pendiri."
      },
      {
        "title": "Pengembangan Layanan (2021)",
        "marker": "bg-success",
        "text": "Pada tahun 2021, koperasi mulai mengembangkan layanan simpan pinjam yang lebih komprehensif. Sistem manajemen yang lebih baik diimplementasikan untuk meningkatkan efisiensi dan transparansi operasional."
      },
      {
        "title": "Digitalisasi Layanan (2022)",
        "marker": "bg-warning",
        "text": "Tahun 2022 menandai era digitalisasi koperasi. Platform online dikembangkan untuk memudahkan anggota mengakses layanan dari mana saja, kapan saja, melalui aplikasi web modern."
      },
      {
        "title": "Pertumbuhan dan Inovasi (2023-Sekarang)",
        "marker": "bg-info",
        "text": "Saat ini, Koperasi Wirya terus berkembang dengan fokus pada inovasi layanan dan peningkatan kesejahteraan anggota. Kami berkomitmen untuk menjadi mitra terpercaya dalam membangun ekonomi bersama."
      }
    ],
    "komitmen_title": "Komitmen Kami",
    "komitmen_text": "Koperasi Wirya berkomitmen untuk terus berkembang dan berinovasi dalam memberikan layanan terbaik kepada anggota. Dengan prinsip kebersamaan dan demokrasi, kami berusaha menciptakan dampak positif bagi masyarakat dan ekonomi lokal."
  }'
),
('visi-misi', 'Visi dan Misi', 'tentang',
  '{
    "visi": "Menjadi koperasi simpan pinjam terdepan yang mampu mendorong pemberdayaan ekonomi anggota dan masyarakat melalui layanan keuangan yang inovatif, transparan, dan berkelanjutan.",
    "misi": [
      "Meningkatkan kesejahteraan anggota melalui usaha yang inovatif dan berkelanjutan",
      "Menyediakan layanan simpan pinjam yang mudah, cepat, dan transparan",
      "Mengembangkan program pendidikan dan pelatihan koperasi",
      "Membangun kemitraan dengan stakeholder untuk kemajuan bersama"
    ]
  }'
);

-- Update sejarah content
UPDATE halaman SET konten = '{
  "teks": "Koperasi adalah bentuk organisasi ekonomi yang didirikan oleh masyarakat untuk memenuhi kebutuhan bersama. Gerakan koperasi modern dimulai di Eropa pada abad ke-19, dipelopori oleh tokoh-tokoh seperti Robert Owen di Inggris dan Charles Fourier di Prancis. Mereka melihat koperasi sebagai alternatif terhadap kapitalisme yang eksploitatif.\n\nDi Indonesia, perkembangan koperasi dimulai pada masa kolonial Belanda. Pada tahun 1896, Raden Aria Wiriatmadja mendirikan Koperasi Kredit pertama di Purwokerto, Jawa Tengah. Gerakan ini semakin berkembang setelah Indonesia merdeka, dengan dukungan pemerintah untuk membangun ekonomi rakyat.\n\nPada tahun 1967, pemerintah mengeluarkan Undang-Undang No. 12 Tahun 1967 tentang Pokok-Pokok Perkoperasian. Undang-undang ini kemudian diganti dengan Undang-Undang No. 25 Tahun 1992 tentang Perkoperasian, yang menjadi dasar hukum koperasi di Indonesia hingga saat ini.\n\nKoperasi di Indonesia berperan penting dalam berbagai sektor, termasuk simpan pinjam, pertanian, konsumsi, dan produksi. Prinsip-prinsip koperasi seperti keanggotaan sukarela, pengelolaan demokratis, dan pembagian hasil secara adil menjadi landasan operasional koperasi.",
  "gambar": "/static/images/placeholder.png"
}' WHERE slug = 'sejarah';

-- Update dashboard content
UPDATE halaman SET konten = '{"welcome":"Selamat Datang di Koperasi KOPMA","slogan":"Dari Anggota, Oleh Anggota, dan Untuk Anggota","teks":"Selamat datang di dashboard anggota.","gambar":"/static/images/placeholder.png"}' WHERE slug = 'dashboard_anggota';

-- PERBAIKAN: Ubah kolom tanggal dari DATE ke TIMESTAMP agar menyimpan waktu
ALTER TABLE detail ALTER COLUMN tgl_transaksi TYPE TIMESTAMP WITHOUT TIME ZONE;
ALTER TABLE detail ALTER COLUMN tgl_transaksi SET DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE pinjaman ALTER COLUMN tgl_pinjaman TYPE TIMESTAMP WITHOUT TIME ZONE;
ALTER TABLE pinjaman ALTER COLUMN tgl_pinjaman SET DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE angsuran ALTER COLUMN tgl_bayar TYPE TIMESTAMP WITHOUT TIME ZONE;
ALTER TABLE angsuran ALTER COLUMN tgl_bayar SET DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE pinjaman ADD COLUMN IF NOT EXISTS metode_pencairan VARCHAR(20);
ALTER TABLE pinjaman ADD COLUMN IF NOT EXISTS nomor_rekening VARCHAR(50);
ALTER TABLE pinjaman ADD COLUMN IF NOT EXISTS gaji_bulanan NUMERIC(15,2) DEFAULT 0;
ALTER TABLE pinjaman ADD COLUMN IF NOT EXISTS tujuan_pinjaman TEXT;
ALTER TABLE detail ADD COLUMN IF NOT EXISTS status VARCHAR(25) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'rejected'));
ALTER TABLE detail ADD COLUMN IF NOT EXISTS bukti_pembayaran VARCHAR(255);

-- Update angsuran status constraint to support pending workflow
ALTER TABLE angsuran DROP CONSTRAINT IF EXISTS angsuran_status_check;
ALTER TABLE angsuran ADD CONSTRAINT angsuran_status_check CHECK (status IN ('pending', 'confirmed', 'rejected', 'valid', 'invalid'));
ALTER TABLE angsuran ALTER COLUMN status SET DEFAULT 'pending';
