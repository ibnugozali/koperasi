package repository

import (
	"database/sql"
	"fmt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

type overdueLoanNotification struct {
	idPinjaman  int
	idAnggota   string
	jangkaWaktu int
	jatuhTempo  string
}

type overdueInstallmentNotification struct {
	idPinjaman int
	idAnggota  string
	angsuranKe int
	jatuhTempo string
}

type OverdueInstallmentWhatsAppNotification struct {
	IDPinjaman  int
	AngsuranKe  int
	IDAnggota   string
	NamaAnggota string
	NoTelepon   string
	JatuhTempo  string
}

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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pesans, nil
}

// EnsureOverdueInstallmentPesanByAnggotaID memulihkan pesan aplikasi dari
// catatan pengingat cicilan yang sudah ada, termasuk yang sebelumnya hanya
// terkirim melalui WhatsApp.
func EnsureOverdueInstallmentPesanByAnggotaID(idAnggota string) error {
	db := config.GetDB()
	_, err := db.Exec(`
		INSERT INTO pesan (id_anggota, judul, isi, status, tgl_kirim)
		SELECT p.id_anggota,
		       'Pengingat Keterlambatan Cicilan',
		       'Cicilan ke-' || n.angsuran_ke || ' pinjaman #' || n.id_pinjaman ||
		       ' telah melewati tanggal jatuh tempo ' ||
		       TO_CHAR((p.tgl_pinjaman + (n.angsuran_ke * INTERVAL '1 month'))::date, 'DD-MM-YYYY') ||
		       ' dan belum tercatat dibayar. Silakan segera lakukan pembayaran atau hubungi bendahara jika Anda sudah membayar.',
		       'unread',
		       NOW()
		FROM notifikasi_angsuran_terlambat n
		JOIN pinjaman p ON p.id_pinjaman = n.id_pinjaman
		WHERE p.id_anggota = $1
		  AND (p.tgl_pinjaman + (n.angsuran_ke * INTERVAL '1 month'))::date < CURRENT_DATE
		  AND NOT EXISTS (
		      SELECT 1
		      FROM pesan m
		      WHERE m.id_anggota = p.id_anggota
		        AND m.judul = 'Pengingat Keterlambatan Cicilan'
		        AND m.isi = 'Cicilan ke-' || n.angsuran_ke || ' pinjaman #' || n.id_pinjaman ||
		            ' telah melewati tanggal jatuh tempo ' ||
		            TO_CHAR((p.tgl_pinjaman + (n.angsuran_ke * INTERVAL '1 month'))::date, 'DD-MM-YYYY') ||
		            ' dan belum tercatat dibayar. Silakan segera lakukan pembayaran atau hubungi bendahara jika Anda sudah membayar.'
		  )
	`, idAnggota)
	if err != nil {
		return fmt.Errorf("gagal memulihkan pesan pengingat cicilan anggota %s: %w", idAnggota, err)
	}
	return nil
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

// CreateOverdueLoanNotifications membuat satu pesan untuk setiap pinjaman aktif yang
// sudah melewati akhir jangka waktu dan belum pernah diberi pengingat.
func CreateOverdueLoanNotifications() (int, error) {
	db := config.GetDB()
	rows, err := db.Query(`
		SELECT p.id_pinjaman, p.id_anggota, p.jangka_waktu,
		       (p.tgl_pinjaman + INTERVAL '1 month' * p.jangka_waktu)::date
		FROM pinjaman p
		JOIN anggota a ON a.id_anggota = p.id_anggota
		WHERE p.status = 'aktif'
		  AND p.tgl_pinjaman + INTERVAL '1 month' * p.jangka_waktu < CURRENT_DATE
	`)
	if err != nil {
		return 0, fmt.Errorf("gagal mencari pinjaman terlambat: %w", err)
	}

	var loans []overdueLoanNotification
	for rows.Next() {
		var loan overdueLoanNotification
		var jatuhTempo sql.NullTime
		if err := rows.Scan(&loan.idPinjaman, &loan.idAnggota, &loan.jangkaWaktu, &jatuhTempo); err != nil {
			rows.Close()
			return 0, fmt.Errorf("gagal membaca data pinjaman terlambat: %w", err)
		}
		if !jatuhTempo.Valid {
			rows.Close()
			return 0, fmt.Errorf("tanggal jatuh tempo pinjaman %d tidak valid", loan.idPinjaman)
		}
		loan.jatuhTempo = jatuhTempo.Time.Format("02-01-2006")
		loans = append(loans, loan)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("gagal membaca daftar pinjaman terlambat: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("gagal menutup hasil pencarian pinjaman terlambat: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("gagal memulai transaksi notifikasi pinjaman terlambat: %w", err)
	}
	defer tx.Rollback()

	created := 0
	for _, loan := range loans {
		_, err := tx.Exec(`
			INSERT INTO notifikasi_pinjaman_terlambat (id_pinjaman, id_anggota)
			VALUES ($1, $2)
			ON CONFLICT (id_pinjaman) DO UPDATE
			SET id_anggota = EXCLUDED.id_anggota
		`, loan.idPinjaman, loan.idAnggota)
		if err != nil {
			return 0, fmt.Errorf("gagal mencatat pengingat pinjaman %d: %w", loan.idPinjaman, err)
		}

		isi := fmt.Sprintf(
			"Pinjaman #%d Anda telah melewati tanggal jatuh tempo %s (jangka waktu %d bulan). Silakan hubungi bendahara untuk informasi pembayaran.",
			loan.idPinjaman, loan.jatuhTempo, loan.jangkaWaktu,
		)
		result, err := tx.Exec(`
			INSERT INTO pesan (id_anggota, judul, isi, status, tgl_kirim)
			SELECT $1::VARCHAR(50), $2::VARCHAR(100), $3::TEXT, 'unread', NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM pesan
				WHERE id_anggota = $1::VARCHAR(50)
				  AND judul = $2::VARCHAR(100)
				  AND isi = $3::TEXT
			)
		`, loan.idAnggota, "Pengingat Keterlambatan Pembayaran", isi)
		if err != nil {
			return 0, fmt.Errorf("gagal membuat pesan pengingat pinjaman %d: %w", loan.idPinjaman, err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("gagal memeriksa pesan pengingat pinjaman %d: %w", loan.idPinjaman, err)
		}
		created += int(rowsAffected)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("gagal menyimpan pengingat pinjaman terlambat: %w", err)
	}
	return created, nil
}

// CreateOverdueInstallmentNotifications membuat satu pesan untuk setiap cicilan
// bulanan yang sudah jatuh tempo dan belum dibayar atau diajukan untuk diverifikasi.
func CreateOverdueInstallmentNotifications() (int, error) {
	db := config.GetDB()
	rows, err := db.Query(`
		WITH pinjaman_aktif AS (
			SELECT p.id_pinjaman,
			       p.id_anggota,
			       p.tgl_pinjaman::date AS tgl_pinjaman,
			       p.jangka_waktu,
			       (
			           SELECT COUNT(*)
			           FROM angsuran a
			           WHERE a.id_pinjaman = p.id_pinjaman
			             AND (
			                 LOWER(COALESCE(a.status, '')) IN ('confirmed', 'diterima', 'lunas')
			                 OR (
			                     LOWER(COALESCE(a.status, '')) = 'pending'
			                     AND NULLIF(TRIM(COALESCE(a.bukti_angsuran, '')), '') IS NOT NULL
			                     AND UPPER(COALESCE(a.bukti_angsuran, '')) NOT LIKE '%AUTO_%'
			                 )
			             )
			       ) AS cicilan_tercatat
			FROM pinjaman p
			WHERE LOWER(TRIM(COALESCE(p.status, ''))) = 'aktif'
			  AND p.jangka_waktu > 0
		),
		cicilan_terlambat AS (
			SELECT p.id_pinjaman,
			       p.id_anggota,
			       gs.angsuran_ke,
			       (p.tgl_pinjaman + (gs.angsuran_ke * INTERVAL '1 month'))::date AS jatuh_tempo
			FROM pinjaman_aktif p
			CROSS JOIN LATERAL generate_series(1, p.jangka_waktu) AS gs(angsuran_ke)
			WHERE gs.angsuran_ke > p.cicilan_tercatat
			  AND (p.tgl_pinjaman + (gs.angsuran_ke * INTERVAL '1 month'))::date < CURRENT_DATE
		)
		SELECT id_pinjaman, id_anggota, angsuran_ke, jatuh_tempo
		FROM cicilan_terlambat
		ORDER BY id_pinjaman, angsuran_ke
	`)
	if err != nil {
		return 0, fmt.Errorf("gagal mencari cicilan bulanan terlambat: %w", err)
	}

	var installments []overdueInstallmentNotification
	for rows.Next() {
		var installment overdueInstallmentNotification
		var jatuhTempo sql.NullTime
		if err := rows.Scan(&installment.idPinjaman, &installment.idAnggota, &installment.angsuranKe, &jatuhTempo); err != nil {
			rows.Close()
			return 0, fmt.Errorf("gagal membaca cicilan bulanan terlambat: %w", err)
		}
		if !jatuhTempo.Valid {
			rows.Close()
			return 0, fmt.Errorf("tanggal jatuh tempo cicilan ke-%d pinjaman %d tidak valid", installment.angsuranKe, installment.idPinjaman)
		}
		installment.jatuhTempo = jatuhTempo.Time.Format("02-01-2006")
		installments = append(installments, installment)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("gagal membaca daftar cicilan bulanan terlambat: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("gagal menutup hasil pencarian cicilan bulanan terlambat: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("gagal memulai transaksi pengingat cicilan terlambat: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM pesan m
		USING notifikasi_angsuran_terlambat n, pinjaman p
		WHERE m.id_anggota = n.id_anggota
		  AND m.judul = 'Pengingat Keterlambatan Cicilan'
		  AND m.isi = 'Cicilan ke-' || n.angsuran_ke || ' pinjaman #' || n.id_pinjaman ||
		      ' telah melewati tanggal jatuh tempo ' ||
		      TO_CHAR((p.tgl_pinjaman + ((n.angsuran_ke - 1) * INTERVAL '1 month'))::date, 'DD-MM-YYYY') ||
		      ' dan belum tercatat dibayar. Silakan segera lakukan pembayaran atau hubungi bendahara jika Anda sudah membayar.'
		  AND p.id_pinjaman = n.id_pinjaman
		  AND n.wa_sent_at IS NULL
		  AND (p.tgl_pinjaman + (n.angsuran_ke * INTERVAL '1 month'))::date >= CURRENT_DATE
	`)
	if err != nil {
		return 0, fmt.Errorf("gagal membersihkan pesan pengingat cicilan yang belum jatuh tempo: %w", err)
	}
	_, err = tx.Exec(`
		DELETE FROM notifikasi_angsuran_terlambat n
		USING pinjaman p
		WHERE p.id_pinjaman = n.id_pinjaman
		  AND n.wa_sent_at IS NULL
		  AND (p.tgl_pinjaman + (n.angsuran_ke * INTERVAL '1 month'))::date >= CURRENT_DATE
	`)
	if err != nil {
		return 0, fmt.Errorf("gagal membersihkan antrean cicilan yang belum jatuh tempo: %w", err)
	}

	created := 0
	for _, installment := range installments {
		_, err := tx.Exec(`
			INSERT INTO notifikasi_angsuran_terlambat AS existing (id_pinjaman, angsuran_ke, id_anggota)
			VALUES ($1, $2, $3)
			ON CONFLICT (id_pinjaman, angsuran_ke) DO UPDATE
			SET id_anggota = EXCLUDED.id_anggota,
			    wa_sent_at = CASE
			        WHEN existing.id_anggota IS DISTINCT FROM EXCLUDED.id_anggota THEN NULL
			        ELSE existing.wa_sent_at
			    END
		`, installment.idPinjaman, installment.angsuranKe, installment.idAnggota)
		if err != nil {
			return 0, fmt.Errorf(
				"gagal mencatat pengingat cicilan ke-%d pinjaman %d: %w",
				installment.angsuranKe, installment.idPinjaman, err,
			)
		}

		isi := fmt.Sprintf(
			"Cicilan ke-%d pinjaman #%d telah melewati tanggal jatuh tempo %s dan belum tercatat dibayar. Silakan segera lakukan pembayaran atau hubungi bendahara jika Anda sudah membayar.",
			installment.angsuranKe, installment.idPinjaman, installment.jatuhTempo,
		)
		result, err := tx.Exec(`
			INSERT INTO pesan (id_anggota, judul, isi, status, tgl_kirim)
			SELECT $1::VARCHAR(50), $2::VARCHAR(100), $3::TEXT, 'unread', NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM pesan
				WHERE id_anggota = $1::VARCHAR(50)
				  AND judul = $2::VARCHAR(100)
				  AND isi = $3::TEXT
			)
		`, installment.idAnggota, "Pengingat Keterlambatan Cicilan", isi)
		if err != nil {
			return 0, fmt.Errorf(
				"gagal membuat pengingat cicilan ke-%d pinjaman %d: %w",
				installment.angsuranKe, installment.idPinjaman, err,
			)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf(
				"gagal memeriksa pesan pengingat cicilan ke-%d pinjaman %d: %w",
				installment.angsuranKe, installment.idPinjaman, err,
			)
		}
		created += int(rowsAffected)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("gagal menyimpan pengingat cicilan terlambat: %w", err)
	}
	return created, nil
}

// GetUnsentOverdueInstallmentWhatsAppNotifications mengambil pengingat cicilan
// yang belum berhasil dikirim ke WhatsApp.
func GetUnsentOverdueInstallmentWhatsAppNotifications() ([]OverdueInstallmentWhatsAppNotification, error) {
	db := config.GetDB()
	rows, err := db.Query(`
		SELECT n.id_pinjaman, n.angsuran_ke, p.id_anggota,
		       COALESCE(a.nama_anggota, ''), COALESCE(a.no_telepon, ''),
		       (p.tgl_pinjaman + (n.angsuran_ke * INTERVAL '1 month'))::date
		FROM notifikasi_angsuran_terlambat n
		JOIN pinjaman p ON p.id_pinjaman = n.id_pinjaman
		JOIN anggota a ON a.id_anggota = p.id_anggota
		WHERE n.wa_sent_at IS NULL
		  AND (p.tgl_pinjaman + (n.angsuran_ke * INTERVAL '1 month'))::date < CURRENT_DATE
		ORDER BY n.dibuat_pada, n.id_pinjaman, n.angsuran_ke
	`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil antrean notifikasi WhatsApp cicilan: %w", err)
	}
	defer rows.Close()

	var notifications []OverdueInstallmentWhatsAppNotification
	for rows.Next() {
		var notification OverdueInstallmentWhatsAppNotification
		var jatuhTempo sql.NullTime
		if err := rows.Scan(
			&notification.IDPinjaman,
			&notification.AngsuranKe,
			&notification.IDAnggota,
			&notification.NamaAnggota,
			&notification.NoTelepon,
			&jatuhTempo,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca antrean notifikasi WhatsApp cicilan: %w", err)
		}
		if !jatuhTempo.Valid {
			return nil, fmt.Errorf(
				"tanggal jatuh tempo cicilan ke-%d pinjaman %d tidak valid",
				notification.AngsuranKe, notification.IDPinjaman,
			)
		}
		notification.JatuhTempo = jatuhTempo.Time.Format("02-01-2006")
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca antrean notifikasi WhatsApp cicilan: %w", err)
	}
	return notifications, nil
}

// MarkOverdueInstallmentWhatsAppNotificationSent menandai pengingat sebagai terkirim.
func MarkOverdueInstallmentWhatsAppNotificationSent(idPinjaman, angsuranKe int) error {
	db := config.GetDB()
	_, err := db.Exec(`
		UPDATE notifikasi_angsuran_terlambat
		SET wa_sent_at = NOW()
		WHERE id_pinjaman = $1 AND angsuran_ke = $2 AND wa_sent_at IS NULL
	`, idPinjaman, angsuranKe)
	if err != nil {
		return fmt.Errorf(
			"gagal menandai notifikasi WhatsApp cicilan ke-%d pinjaman %d terkirim: %w",
			angsuranKe, idPinjaman, err,
		)
	}
	return nil
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
