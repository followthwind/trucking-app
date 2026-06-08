package repository

import (
	"context"
	"database/sql"
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
			amount, buying_price, gross_profit, profit_percentage, 
			remark, document_path, created_at, is_delete, lolo_rate, return_to, bl_number, container_number
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.db.ExecContext(ctx, query,
		s.ID, s.ItemDescription, s.Origin, s.Destination, s.Qty, s.Rate,
		s.Amount, s.BuyingPrice, s.GrossProfit, s.ProfitPercentage,
		s.Remark, s.DocumentPath, s.CreatedAt, s.IsDelete, s.LoloRate, s.ReturnTo, s.BLNumber, s.ContainerNumber,
	)
	return err
}

// FetchAll bertugas menjalankan query SELECT untuk mengambil semua data dari PostgreSQL
func (r *postgresShipmentRepository) FetchAll(ctx context.Context) ([]domain.Shipment, error) {
	query := `
        SELECT id, item_description, origin, destination, qty, rate, 
        amount, buying_price, gross_profit, profit_percentage, 
        remark, document_path, created_at, lolo_rate, return_to, bl_number, container_number
        FROM shipments 
        WHERE is_delete = false 
        ORDER BY created_at DESC
    `
	rows, err := r.db.QueryContext(ctx, query)
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
			&s.Remark, &s.DocumentPath, &s.CreatedAt, &s.LoloRate, &s.ReturnTo, &s.BLNumber, &s.ContainerNumber,
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
