package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models" // Kita akan buat model Halaman sebentar lagi
)

func GetHalamanBySlug(slug string) (models.Halaman, error) {
	db := config.GetDB()
	var h models.Halaman
	query := "SELECT slug, judul, konten FROM halaman WHERE slug = $1"
	err := db.QueryRow(query, slug).Scan(&h.Slug, &h.Judul, &h.Konten)
	return h, err
}

func GetAllHalaman() ([]models.Halaman, error) {
	db := config.GetDB()
	var allHalaman []models.Halaman
	query := "SELECT id, slug, judul FROM halaman"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var h models.Halaman
		if err := rows.Scan(&h.ID, &h.Slug, &h.Judul); err != nil {
			return nil, err
		}
		allHalaman = append(allHalaman, h)
	}
	return allHalaman, nil
}

func UpdateHalaman(h models.Halaman) error {
	db := config.GetDB()
	query := "UPDATE halaman SET judul = $1, konten = $2, updated_at = NOW() WHERE slug = $3"
	_, err := db.Exec(query, h.Judul, h.Konten, h.Slug)
	return err
}
