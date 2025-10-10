package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

func GetRiwayatSimpananByAnggotaID(id int) ([]models.Detail, error) {
	db := config.GetDB()
	var details []models.Detail
	query := `
		SELECT d.id_detail, d.id_anggota, d.id_simpanan, d.id_pengelola, d.tgl_transaksi, d.jumlah_simpanan, d.total_simpanan,
		       s.jenis_simpanan
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE d.id_anggota = $1
		ORDER BY d.tgl_transaksi DESC
	`
	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.Detail
		var s models.Simpanan
		if err := rows.Scan(&d.IDDetail, &d.IDAnggota, &d.IDSimpanan, &d.IDPengelola, &d.TglTransaksi, &d.JumlahSimpanan, &d.TotalSimpanan, &s.JenisSimpanan); err != nil {
			return nil, err
		}
		d.Simpanan = s
		details = append(details, d)
	}
	return details, nil
}

func GetRiwayatPinjamanByAnggotaID(id int) ([]models.Pinjaman, error) {
	db := config.GetDB()
	var pinjamans []models.Pinjaman
	query := `
		SELECT id_pinjaman, id_anggota, id_pengelola, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status
		FROM pinjaman
		WHERE id_anggota = $1
		ORDER BY tgl_pinjaman DESC
	`
	rows, err := db.Query(query, id)
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

func GetRiwayatAngsuranByAnggotaID(id int) ([]models.Angsuran, error) {
	db := config.GetDB()
	var angsurans []models.Angsuran
	query := `
		SELECT a.id_angsuran, a.id_pinjaman, a.id_pengelola, a.tgl_bayar, a.sisa_pinjaman, a.status_angsuran, a.status
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		WHERE p.id_anggota = $1
		ORDER BY a.tgl_bayar DESC
	`
	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Angsuran
		if err := rows.Scan(&a.IDAngsuran, &a.IDPinjaman, &a.IDPengelola, &a.TglBayar, &a.SisaPinjaman, &a.StatusAngsuran, &a.Status); err != nil {
			return nil, err
		}
		angsurans = append(angsurans, a)
	}
	return angsurans, nil
}
