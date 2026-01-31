package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"koperasi-simpan-pinjam/models"
)

type NeracaRepository struct {
	DB *sql.DB
}

func NewNeracaRepository(db *sql.DB) *NeracaRepository {
	return &NeracaRepository{DB: db}
}

// SaveNeraca saves or updates neraca data
func (r *NeracaRepository) SaveNeraca(req *models.NeracaRequest, userID int) error {
	// Convert maps to JSON strings
	data2024, _ := json.Marshal(req.Data2024)
	data2023, _ := json.Marshal(req.Data2023)
	labels, _ := json.Marshal(req.Labels)
	noPerkiraan, _ := json.Marshal(req.NoPerkiraan)
	customItems, _ := json.Marshal(req.CustomItems)
	itemCounter, _ := json.Marshal(req.ItemCounter)
	deletedItems, _ := json.Marshal(req.DeletedItems)

	// Check if neraca already exists for this user
	var exists bool
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM neraca WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Update existing neraca
		query := `UPDATE neraca SET
			data_2024 = $1,
			data_2023 = $2,
			labels = $3,
			no_perkiraan = $4,
			custom_items = $5,
			item_counter = $6,
			deleted_items = $7,
			last_modified_by = $8,
			last_modified_at = $9
			WHERE user_id = $10`

		_, err = r.DB.Exec(query,
			string(data2024),
			string(data2023),
			string(labels),
			string(noPerkiraan),
			string(customItems),
			string(itemCounter),
			string(deletedItems),
			userID,
			time.Now(),
			userID,
		)
		return err
	}

	// Insert new neraca
	query := `INSERT INTO neraca
		(user_id, data_2024, data_2023, labels, no_perkiraan, custom_items, item_counter, deleted_items, created_by, last_modified_by, created_at, last_modified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = r.DB.Exec(query,
		userID,
		string(data2024),
		string(data2023),
		string(labels),
		string(noPerkiraan),
		string(customItems),
		string(itemCounter),
		string(deletedItems),
		userID,
		userID,
		time.Now(),
		time.Now(),
	)
	return err
}

// GetNeraca retrieves neraca data for a specific user
func (r *NeracaRepository) GetNeraca(userID int) (*models.Neraca, error) {
	query := `SELECT id, user_id, data_2024, data_2023, labels, no_perkiraan, custom_items, item_counter, deleted_items, 
		created_by, last_modified_by, created_at, last_modified_at 
		FROM neraca 
		WHERE user_id = $1
		ORDER BY last_modified_at DESC 
		LIMIT 1`

	neraca := &models.Neraca{}
	err := r.DB.QueryRow(query, userID).Scan(
		&neraca.ID,
		&neraca.UserID,
		&neraca.Data2024,
		&neraca.Data2023,
		&neraca.Labels,
		&neraca.NoPerkiraan,
		&neraca.CustomItems,
		&neraca.ItemCounter,
		&neraca.DeletedItems,
		&neraca.CreatedBy,
		&neraca.LastModifiedBy,
		&neraca.CreatedAt,
		&neraca.LastModifiedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No data found
	}

	return neraca, err
}
