package main

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/controllers"
	"koperasi-simpan-pinjam/repository"
	"koperasi-simpan-pinjam/routes"
	"log"
	"time"
)

func main() {
	// Inisialisasi koneksi database saat aplikasi dimulai
	config.InitDB()
	startOverdueLoanNotificationWorker()

	// Menjalankan router
	router := routes.SetupRouter()
	router.Run(":8080")
}

func startOverdueLoanNotificationWorker() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			created, err := repository.CreateOverdueLoanNotifications()
			if err != nil {
				log.Printf("Gagal memeriksa notifikasi pinjaman terlambat: %v", err)
			} else if created > 0 {
				log.Printf("Berhasil membuat %d notifikasi pinjaman terlambat", created)
			}

			createdInstallments, err := repository.CreateOverdueInstallmentNotifications()
			if err != nil {
				log.Printf("Gagal memeriksa notifikasi cicilan bulanan terlambat: %v", err)
			} else if createdInstallments > 0 {
				log.Printf("Berhasil membuat %d notifikasi cicilan bulanan terlambat", createdInstallments)
			}
			if err := controllers.SendOverdueInstallmentWhatsAppNotifications(); err != nil {
				log.Printf("Gagal mengirim notifikasi WhatsApp cicilan bulanan terlambat: %v", err)
			}

			<-ticker.C
		}
	}()
}
