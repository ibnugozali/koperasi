package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

// GetPesanByAnggotaID mengambil semua pesan untuk anggota tertentu
func GetPesanByAnggotaID(idAnggota string) ([]models.Pesan, error) {
	db := config.GetDB()
	var pesans []models.Pesan

	query := `
		SELECT id_pesan, id_anggota, judul, isi, tgl_kirim, status
		FROM pesan
		WHERE id_anggota = $1
		ORDER BY tgl_kirim DESC`

	rows, err := db.Query(query, idAnggota)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Pesan
		err := rows.Scan(&p.IDPesan, &p.IDAnggota, &p.Judul, &p.Isi, &p.TglKirim, &p.Status)
		if err != nil {
			return nil, err
		}
		pesans = append(pesans, p)
	}

	return pesans, nil
}

// CreatePesan membuat pesan baru
func CreatePesan(pesan models.Pesan) error {
	db := config.GetDB()
	query := `
		INSERT INTO pesan (id_anggota, judul, isi, status, tgl_kirim)
		VALUES ($1, $2, $3, $4, NOW())`
	_, err := db.Exec(query, pesan.IDAnggota, pesan.Judul, pesan.Isi, pesan.Status)
	return err
}

// MarkPesanAsRead menandai pesan sebagai sudah dibaca
func MarkPesanAsRead(idPesan int) error {
	db := config.GetDB()
	query := "UPDATE pesan SET status = 'read' WHERE id_pesan = $1"
	_, err := db.Exec(query, idPesan)
	return err
}

// MarkAllPesanAsReadByAnggotaID menandai semua pesan anggota sebagai sudah dibaca.
func MarkAllPesanAsReadByAnggotaID(idAnggota string) error {
	db := config.GetDB()
	query := "UPDATE pesan SET status = 'read' WHERE id_anggota = $1 AND status = 'unread'"
	_, err := db.Exec(query, idAnggota)
	return err
}

// GetPesanNotifSummaryByAnggotaID mengambil ringkasan notifikasi pesan anggota.
func GetPesanNotifSummaryByAnggotaID(idAnggota string) (latestID int, unreadCount int, err error) {
	db := config.GetDB()
	query := `
		SELECT
			COALESCE(MAX(id_pesan), 0) AS latest_id,
			COALESCE(SUM(CASE WHEN status = 'unread' THEN 1 ELSE 0 END), 0) AS unread_count
		FROM pesan
		WHERE id_anggota = $1
	`
	err = db.QueryRow(query, idAnggota).Scan(&latestID, &unreadCount)
	return
}
