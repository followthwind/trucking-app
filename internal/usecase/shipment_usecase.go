package usecase

import (
	"context"
	"time"
	"trucking-app/internal/domain"

	"github.com/google/uuid"
)

type shipmentUsecase struct {
	shipmentRepo domain.ShipmentRepository
	docRepo      domain.DocumentRepository
}

// NewShipmentUsecase melahirkan objek usecase baru
func NewShipmentUsecase(repo domain.ShipmentRepository, docRepo domain.DocumentRepository) domain.ShipmentUsecase {
	return &shipmentUsecase{
		shipmentRepo: repo,
		docRepo:      docRepo,
	}
}

// InsertShipment memproses kalkulasi dan menyimpan data + file dokumen
func (u *shipmentUsecase) InsertShipment(ctx context.Context, s *domain.Shipment, fileData []byte, contentType string) error {
	// 1. Generate ID unik otomatis menggunakan UUID
	s.ID = uuid.New().String()

	// 2. PROSES UPLOAD KE MINIO (Jika manajer mengunggah file)
	if len(fileData) > 0 {
		uniqueFileName := s.ID + "-document"
		
		uploadedName, err := u.docRepo.Upload(ctx, uniqueFileName, fileData, contentType)
		if err != nil {
			return err
		}
		
		s.DocumentPath = uploadedName
	}

	// 3. RUMUS FINANSIAL OTOMATIS
	s.Amount = int64(s.Qty) * s.Rate
	s.GrossProfit = s.Amount - s.BuyingPrice
	
	if s.Amount > 0 {
		s.ProfitPercentage = int((s.GrossProfit * 100) / s.Amount)
	} else {
		s.ProfitPercentage = 0
	}

	// 4. Catat waktu pembuatan
	s.CreatedAt = time.Now()

	// 5. Kirim ke PostgreSQL repo (Hanya mengirim 2 argumen: ctx dan s)
	return u.shipmentRepo.Create(ctx, s)
}

// GetAllShipments menarik daftar semua data dari database
func (u *shipmentUsecase) GetAllShipments(ctx context.Context) ([]domain.Shipment, error) {
	return u.shipmentRepo.FetchAll(ctx)
}

func (u *shipmentUsecase) GetShipmentByID(ctx context.Context, id string) (*domain.Shipment, error) {
	return u.shipmentRepo.FindByID(ctx, id)
}