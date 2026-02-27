package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

// GetLoginHistory mengambil semua riwayat login dari database
func GetLoginHistory() ([]models.LoginHistory, error) {
	db := config.GetDB()
	var loginHistories []models.LoginHistory

	query := `
		SELECT
			id,
			username,
			CASE
				WHEN LOWER(TRIM(role)) IN ('admin', 'bendahara', 'ketua', 'anggota') THEN LOWER(TRIM(role))
				WHEN LOWER(TRIM(role)) = 'member' THEN 'anggota'
				ELSE LOWER(TRIM(role))
			END AS role,
			login_time,
			ip_address,
			CASE
				WHEN LOWER(TRIM(status)) IN ('success', 'berhasil', 'sukses') THEN 'success'
				WHEN LOWER(TRIM(status)) IN ('failed', 'gagal') THEN 'failed'
				ELSE LOWER(TRIM(status))
			END AS status
		FROM login_history
		ORDER BY login_time DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lh models.LoginHistory
		if err := rows.Scan(&lh.ID, &lh.Username, &lh.Role, &lh.LoginTime, &lh.IPAddress, &lh.Status); err != nil {
			return nil, err
		}
		loginHistories = append(loginHistories, lh)
	}

	return loginHistories, nil
}

// CreateLoginHistory menyimpan riwayat login ke database
func CreateLoginHistory(loginHistory models.LoginHistory) error {
	db := config.GetDB()
	query := `
		INSERT INTO login_history (username, role, login_time, ip_address, status)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(query, loginHistory.Username, loginHistory.Role, loginHistory.LoginTime, loginHistory.IPAddress, loginHistory.Status)
	return err
}

// DeleteLoginHistory menghapus riwayat login berdasarkan ID
func DeleteLoginHistory(id int) error {
	db := config.GetDB()
	query := `DELETE FROM login_history WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}

// DeleteAllLoginHistory menghapus semua riwayat login
func DeleteAllLoginHistory() error {
	db := config.GetDB()
	query := `DELETE FROM login_history`
	_, err := db.Exec(query)
	return err
}
