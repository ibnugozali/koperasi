package models

import "time"

// LoginHistory merepresentasikan struktur data untuk riwayat login
type LoginHistory struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	LoginTime time.Time `json:"login_time"`
	IPAddress string    `json:"ip_address"`
	Status    string    `json:"status"`
}
