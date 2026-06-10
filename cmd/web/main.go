package main

import (
	"fmt"
	"log"
	netHttp "net/http"
	"trucking-app/internal/delivery/http"
	"trucking-app/internal/repository"
	"trucking-app/internal/usecase"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load file .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Peringatan: File .env tidak ditemukan, menggunakan env sistem")
	}

	// 2. Konek ke Database
	db, err := repository.InitDB()
	if err != nil {
		log.Fatalf("Gagal mengoneksikan database: %v", err)
	}
	defer db.Close()

	minioRepo, err := repository.NewMinioRepository()
	if err != nil {
		log.Fatalf("Gagal mengoneksikan MinIO: %v", err)
	}

	// 3. Rakit Clean Architecture (Dependency Injection)
	// Hubungkan DB -> Repository -> Usecase
	shipmentRepo := repository.NewPostgresShipmentRepository(db)
	docRepo := repository.NewPostgresDocumentRepository(db)
	shipmentUsecase := usecase.NewShipmentUsecase(shipmentRepo, minioRepo, docRepo)

	// Inisialisasi HTTP Handler
	webHandler := http.NewShipmentHandler(shipmentUsecase, minioRepo)

	// Gunakan multiplexer bawaan Go sebagai engine router
	mux := netHttp.NewServeMux()
	webHandler.RegisterRoutes(mux)

	port := ":8080"
	fmt.Printf("🚀 Server Web Trucking Menyala Tampan di http://localhost%s\n", port)
	log.Fatal(netHttp.ListenAndServe(port, mux))

}
