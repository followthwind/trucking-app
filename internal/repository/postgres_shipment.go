package repository

import (
	"context"
	"database/sql"
	"time"
	"trucking-app/internal/domain"
)

// postgresShipmentRepository adalah struct yang memegang koneksi ke database
type postgresShipmentRepository struct {
	db *sql.DB
}

// NewPostgresShipmentRepository adalah fungsi untuk melahirkan repository ini
func NewPostgresShipmentRepository(db *sql.DB) domain.ShipmentRepository {
	return &postgresShipmentRepository{db: db}
}

// Create bertugas menjalankan query INSERT ke PostgreSQL
func (r *postgresShipmentRepository) Create(ctx context.Context, s *domain.Shipment) error {
	query := `
		INSERT INTO shipments (
			id, item_description, origin, destination, qty, rate, 
			amount, remark, created_at, is_delete
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		s.ID, s.ItemDescription, s.Origin, s.Destination, s.Qty, s.Rate,
		s.Amount,
		s.Remark, s.CreatedAt, s.IsDelete,
	)
	return err
}

func (r *postgresShipmentRepository) Update(ctx context.Context, s *domain.Shipment) error {
	// HAPUS kolom document_path dari query update
	query := `
		UPDATE shipments 
		SET buying_price = $1, 
		    gross_profit = $2, 
		    profit_percentage = $3, 
		    lolo_rate = $4, 
		    return_to = $5, 
		    bl_number = $6, 
		    container_number = $7, 
		    status = $8
		WHERE id = $9
	`

	// Sesuaikan urutan parameternya (sekarang cuma sampai $9)
	_, err := r.db.ExecContext(ctx, query,
		s.BuyingPrice,
		s.GrossProfit,
		s.ProfitPercentage,
		s.LoloRate,
		s.ReturnTo,
		s.BLNumber,
		s.ContainerNumber,
		s.Status,
		s.ID,
	)

	return err
}

func (r *postgresShipmentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	// Hapus updated_at karena kolomnya tidak ada di database kamu
	query := `
		UPDATE shipments 
		SET status = $1 
		WHERE id = $2
	`

	// Eksekusi query ke database
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	// Pastikan bahwa memang ada baris data yang ter-update
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// FetchAll bertugas menjalankan query SELECT untuk mengambil semua data dari PostgreSQL
func (r *postgresShipmentRepository) FetchAll(ctx context.Context, startDate, endDate time.Time) ([]domain.Shipment, error) {
	// Tambahkan kondisi BETWEEN untuk menyaring berdasarkan tanggal kalender bulanan
	query := `
        SELECT id, item_description, origin, destination, qty, rate, 
               amount, buying_price, gross_profit, profit_percentage, 
               remark, document_path, created_at, lolo_rate, return_to, bl_number, container_number, status
        FROM shipments 
        WHERE is_delete = false 
          AND created_at BETWEEN $1 AND $2
        ORDER BY created_at DESC
    `

	// Masukkan startDate ($1) dan endDate ($2) ke dalam QueryContext
	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []domain.Shipment
	for rows.Next() {
		var s domain.Shipment
		err := rows.Scan(
			&s.ID, &s.ItemDescription, &s.Origin, &s.Destination, &s.Qty, &s.Rate,
			&s.Amount, &s.BuyingPrice, &s.GrossProfit, &s.ProfitPercentage,
			&s.Remark, &s.DocumentPath, &s.CreatedAt, &s.LoloRate, &s.ReturnTo, &s.BLNumber, &s.ContainerNumber, &s.Status,
		)
		if err != nil {
			return nil, err
		}
		shipments = append(shipments, s)
	}

	return shipments, nil
}
func (r *postgresShipmentRepository) FindByID(ctx context.Context, id string) (*domain.Shipment, error) {
	query := `SELECT id, item_description, origin, destination, qty, rate, amount, buying_price, gross_profit, profit_percentage, remark, document_path, created_at FROM shipments WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var s domain.Shipment
	err := row.Scan(&s.ID, &s.ItemDescription, &s.Origin, &s.Destination, &s.Qty, &s.Rate, &s.Amount, &s.BuyingPrice, &s.GrossProfit, &s.ProfitPercentage, &s.Remark, &s.DocumentPath, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *postgresShipmentRepository) Delete(ctx context.Context, id string) error {
	// Mengubah status is_delete menjadi true
	query := `UPDATE shipments SET is_delete = true WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
