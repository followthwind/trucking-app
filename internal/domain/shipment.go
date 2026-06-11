package domain

import (
	"context"
	"io"
	"time"
)

type Shipment struct {
	ID               string         `json:"id"`
	ItemDescription  string         `json:"item_description"`
	Origin           string         `json:"origin"`
	Destination      string         `json:"destination"`
	Qty              int            `json:"qty"`
	Rate             int64          `json:"rate"`
	Amount           int64          `json:"amount"`
	BuyingPrice      int64          `json:"buying_price"`
	GrossProfit      int64          `json:"gross_profit"`
	ProfitPercentage int            `json:"profit_percentage"`
	Remark           string         `json:"remark"`
	DocumentPath     string         `json:"document_path"`
	CreatedAt        time.Time      `json:"created_at"`
	LoloRate         int64          `json:"lolo_rate"`
	ReturnTo         string         `json:"return_to"`
	Status           ShipmentStatus `json:"status"`
	BLNumber         string         `json:"bl_number"`
	ContainerNumber  string         `json:"container_number"`

	IsDelete bool `json:"is_delete"`
}

type ShipmentStatus string

const (
	StatusPending ShipmentStatus = "PENDING"
	StatusProcess ShipmentStatus = "INVOICING"
	StatusDone    ShipmentStatus = "SUBMIT OA DONE"
)

type ShipmentDocument struct {
	ID           string    `json:"id"`
	ShipmentID   string    `json:"shipment_id"`
	FileName     string    `json:"file_name"`
	DocumentPath string    `json:"document_path"`
	CreatedAt    time.Time `json:"created_at"`
}

type ShipmentRepository interface {
	Create(ctx context.Context, shipment *Shipment) error
	FetchAll(ctx context.Context, startDate, endDate time.Time) ([]Shipment, error)
	FindByID(ctx context.Context, id string) (*Shipment, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, shipment *Shipment) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

type ShipmentUsecase interface {
	// Pastikan parameternya sama persis dengan yang ada di Usecase Anda saat ini
	InsertShipment(ctx context.Context, shipment *Shipment) error
	GetAllShipments(ctx context.Context, startDate, endDate time.Time) ([]Shipment, error)
	GetShipmentByID(ctx context.Context, id string) (*Shipment, error) // <-- Tambahkan ini
	DeleteShipment(ctx context.Context, id string) error
	UpdateShipmentStatus(ctx context.Context, id string, status string) error
	UploadDocument(ctx context.Context, shipmentID, fileName, contentType string, fileData []byte) (*ShipmentDocument, error)
	GetShipmentDocuments(ctx context.Context, shipmentID string) ([]ShipmentDocument, error)
	UpdateShipmentData(ctx context.Context, s *Shipment) error
	DeleteShipmentDocument(ctx context.Context, docID string) error
}

type MinioRepository interface {
	Upload(ctx context.Context, fileName string, fileData []byte, contentType string) (string, error)
	GetObject(ctx context.Context, fileName string) (io.ReadCloser, error) // Tambahkan baris ini
	DeleteFile(ctx context.Context, fileName string) error
}

type DocumentRepository interface {
	Save(ctx context.Context, doc *ShipmentDocument) error
	FindByShipmentID(ctx context.Context, shipmentID string) ([]ShipmentDocument, error)
	DeleteDoc(ctx context.Context, id string) error // Nama ini yang dipakai sesuai error compiler tadi
	FindDocByID(ctx context.Context, id string) (*ShipmentDocument, error)
}

// fileData []byte, contentType string
