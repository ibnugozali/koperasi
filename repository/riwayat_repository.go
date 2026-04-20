package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

func GetRiwayatSimpananByAnggotaID(id string, search string) ([]models.Detail, error) {
	db := config.GetDB()
	var details []models.Detail
	query := `
        SELECT d.id_detail, d.id_anggota, d.id_simpanan, d.id_pengelola, d.tgl_transaksi, d.jumlah_simpanan, COALESCE(d.total_simpanan, 0),
               s.jenis_simpanan, COALESCE(d.status, 'confirmed') as status
        FROM detail d
        JOIN simpanan s ON d.id_simpanan = s.id_simpanan
        WHERE d.id_anggota = $1
    `
	args := []interface{}{id}
	if search != "" {
		query += ` AND (s.jenis_simpanan ILIKE $2 OR CAST(d.jumlah_simpanan AS TEXT) ILIKE $2 OR CAST(d.total_simpanan AS TEXT) ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY d.tgl_transaksi DESC, d.id_detail DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.Detail
		var s models.Simpanan
		if err := rows.Scan(&d.IDDetail, &d.IDAnggota, &d.IDSimpanan, &d.IDPengelola, &d.TglTransaksi, &d.JumlahSimpanan, &d.TotalSimpanan, &s.JenisSimpanan, &d.Status); err != nil {
			return nil, err
		}
		d.Simpanan = s
		details = append(details, d)
	}
	return details, nil
}

func GetRiwayatPinjamanByAnggotaID(id string, search string) ([]models.Pinjaman, error) {
	db := config.GetDB()
	var pinjamans []models.Pinjaman
	query := `
		SELECT id_pinjaman, id_anggota, id_pengelola, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status
		FROM pinjaman
		WHERE id_anggota = $1
	`
	args := []interface{}{id}
	if search != "" {
		query += ` AND (status ILIKE $2 OR CAST(jumlah_pinjaman AS TEXT) ILIKE $2 OR CAST(jangka_waktu AS TEXT) ILIKE $2 OR CAST(bunga AS TEXT) ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY tgl_pinjaman DESC, id_pinjaman DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Pinjaman
		if err := rows.Scan(&p.IDPinjaman, &p.IDAnggota, &p.IDPengelola, &p.TglPinjaman, &p.JumlahPinjaman, &p.JangkaWaktu, &p.Bunga, &p.Status); err != nil {
			return nil, err
		}
		pinjamans = append(pinjamans, p)
	}
	return pinjamans, nil
}
func GetRiwayatAngsuranByAnggotaID(id string, search string) ([]models.Angsuran, error) {
	db := config.GetDB()
	var angsurans []models.Angsuran
	query := `
        SELECT a.id_angsuran, a.id_pinjaman, a.id_pengelola, a.tgl_bayar, a.jumlah_angsuran, a.sisa_pinjaman, a.bukti_angsuran, 
               a.status, ang.nama_anggota
        FROM angsuran a
        JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
        JOIN anggota ang ON p.id_anggota = ang.id_anggota
        WHERE p.id_anggota = $1
    `
	args := []interface{}{id}
	if search != "" {
		query += ` AND (a.status ILIKE $2 OR CAST(a.sisa_pinjaman AS TEXT) ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY a.tgl_bayar DESC, a.id_angsuran DESC`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Angsuran
		if err := rows.Scan(&a.IDAngsuran, &a.IDPinjaman, &a.IDPengelola, &a.TglBayar, &a.JumlahAngsuran, &a.SisaPinjaman, &a.BuktiAngsuran, &a.Status, &a.NamaAnggota); err != nil {
			return nil, err
		}
		angsurans = append(angsurans, a)
	}
	return angsurans, nil
}

// GetRiwayatTransaksiByAnggotaID mengambil riwayat transaksi gabungan (simpanan & pinjaman) untuk anggota tertentu
func GetRiwayatTransaksiByAnggotaID(id string) ([]models.Riwayat, error) {
	db := config.GetDB()
	var riwayats []models.Riwayat
	query := `
		SELECT d.id_detail AS id, d.tgl_transaksi AS tanggal, 'Simpanan' AS jenis, d.jumlah_simpanan AS jumlah, d.status AS status, a.nama_anggota, a.id_anggota, a.no_telepon, a.gaji_bulanan
		FROM detail d
		JOIN anggota a ON d.id_anggota = a.id_anggota
		WHERE d.id_anggota = $1
		UNION ALL
		SELECT p.id_pinjaman AS id, p.tgl_pinjaman AS tanggal, 'Pinjaman' AS jenis, p.jumlah_pinjaman AS jumlah, p.status AS status, a.nama_anggota, a.id_anggota, a.no_telepon, a.gaji_bulanan
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota
		WHERE p.id_anggota = $1
		UNION ALL
		SELECT ag.id_angsuran AS id, ag.tgl_bayar AS tanggal, 'Angsuran' AS jenis, ag.jumlah_angsuran AS jumlah, ag.status AS status, a.nama_anggota, a.id_anggota, a.no_telepon, a.gaji_bulanan
		FROM angsuran ag
		JOIN pinjaman p ON ag.id_pinjaman = p.id_pinjaman
		JOIN anggota a ON p.id_anggota = a.id_anggota
		WHERE p.id_anggota = $1
		ORDER BY tanggal DESC, id DESC
	`
	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r models.Riwayat
		if err := rows.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota, &r.IDAnggota, &r.NoTelepon, &r.GajiBulanan); err != nil {
			return nil, err
		}
		riwayats = append(riwayats, r)
	}
	return riwayats, nil
}
