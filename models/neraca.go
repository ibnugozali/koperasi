package models

import "time"

// Neraca represents the balance sheet data
type Neraca struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`          // User ID yang memiliki neraca
	Data2024       string    `json:"data_2024"`        // JSON string of neracaData
	Data2023       string    `json:"data_2023"`        // JSON string of neracaData2023
	Labels         string    `json:"labels"`           // JSON string of neracaLabels
	NoPerkiraan    string    `json:"no_perkiraan"`     // JSON string of neracaNoPerkiraan
	CustomItems    string    `json:"custom_items"`     // JSON string of customItems
	ItemCounter    string    `json:"item_counter"`     // JSON string of itemCounter
	DeletedItems   string    `json:"deleted_items"`    // JSON string of deletedItems
	CreatedBy      int       `json:"created_by"`       // User ID yang membuat
	LastModifiedBy int       `json:"last_modified_by"` // User ID yang terakhir edit
	CreatedAt      time.Time `json:"created_at"`
	LastModifiedAt time.Time `json:"last_modified_at"`
}

// NeracaRequest represents the request body for saving neraca
type NeracaRequest struct {
	Data2024     map[string]interface{} `json:"data_2024"`
	Data2023     map[string]interface{} `json:"data_2023"`
	Labels       map[string]string      `json:"labels"`
	NoPerkiraan  map[string]string      `json:"no_perkiraan"`
	CustomItems  map[string]interface{} `json:"custom_items"`
	ItemCounter  map[string]int         `json:"item_counter"`
	DeletedItems []string               `json:"deleted_items"`
}
