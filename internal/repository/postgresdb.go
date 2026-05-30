package repository

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // Driver PostgreSQL
)


func InitDB() (*sql.DB, error) {
	// Ambil data konfigurasi dari environment variable
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	// Susun string koneksi (DSN)
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// Buka koneksi ke Postgres
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Tes apakah koneksi benar-benar tersambung dan password-nya benar
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("---Berhasil terhubung ke database PostgreSQL!")
	return db, nil
}