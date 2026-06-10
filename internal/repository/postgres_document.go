package repository

import (
	"context"
	"database/sql"
	"trucking-app/internal/domain"
)

type postgresDocumentRepository struct {
	db *sql.DB
}

// Ini fungsi yang dicari-cari oleh main.go kamu tadi!
func NewPostgresDocumentRepository(db *sql.DB) domain.DocumentRepository {
	return &postgresDocumentRepository{db: db}
}

func (r *postgresDocumentRepository) Save(ctx context.Context, doc *domain.ShipmentDocument) error {
	query := `INSERT INTO shipment_documents (id, shipment_id, file_name, document_path) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, doc.ID, doc.ShipmentID, doc.FileName, doc.DocumentPath)
	return err
}

func (r *postgresDocumentRepository) FindByShipmentID(ctx context.Context, shipmentID string) ([]domain.ShipmentDocument, error) {
	query := `SELECT id, shipment_id, file_name, document_path, created_at FROM shipment_documents WHERE shipment_id = $1`
	rows, err := r.db.QueryContext(ctx, query, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []domain.ShipmentDocument
	for rows.Next() {
		var doc domain.ShipmentDocument
		if err := rows.Scan(&doc.ID, &doc.ShipmentID, &doc.FileName, &doc.DocumentPath, &doc.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func (r *postgresDocumentRepository) FindDocByID(ctx context.Context, id string) (*domain.ShipmentDocument, error) {
	// Pastikan nama kolom (id, shipment_id, file_name, document_path) sesuai dengan isi tabel DB kamu
	query := `SELECT id, shipment_id, file_name, document_path FROM shipment_documents WHERE id = $1`

	var doc domain.ShipmentDocument
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&doc.ShipmentID,
		&doc.FileName,
		&doc.DocumentPath,
	)

	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *postgresDocumentRepository) DeleteDoc(ctx context.Context, id string) error {
	query := `DELETE FROM shipment_documents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
