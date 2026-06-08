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
func (u *shipmentUsecase) InsertShipment(ctx context.Context, s *domain.Shipment) error {
	// 1. Generate ID unik otomatis menggunakan UUID
	s.ID = uuid.New().String()

	// // 2. PROSES UPLOAD KE MINIO (Jika manajer mengunggah file)
	// if len(fileData) > 0 {
	// 	uniqueFileName := s.ID + "-document"

	// 	uploadedName, err := u.docRepo.Upload(ctx, uniqueFileName, fileData, contentType)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	s.DocumentPath = uploadedName
	// }

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

func (u *shipmentUsecase) UpdateShipment(ctx context.Context, s *domain.Shipment, fileData []byte, contentType string) error {

	existing, err := u.shipmentRepo.FindByID(ctx, s.ID)
	if err != nil {
		return err
	}

	// 2. Hitung ulang finansial secara lengkap
	existing.BuyingPrice = s.BuyingPrice
	existing.LoloRate = s.LoloRate
	existing.ReturnTo = s.ReturnTo
	existing.BLNumber = s.BLNumber
	existing.ContainerNumber = s.ContainerNumber

	// Rumus Finansial
	existing.GrossProfit = existing.Amount - s.BuyingPrice
	if existing.Amount > 0 {
		existing.ProfitPercentage = int((existing.GrossProfit * 100) / existing.Amount)
	}

	// Ubah status menjadi INVOICED karena data sudah lengkap
	existing.Status = domain.StatusProcess

	// 3. Jika ada upload file invoice/bukti baru dari finance
	if len(fileData) > 0 {
		uniqueFileName := s.ID + "-invoice"
		uploadedName, err := u.docRepo.Upload(ctx, uniqueFileName, fileData, contentType)
		if err != nil {
			return err
		}
		existing.DocumentPath = uploadedName
	}

	return u.shipmentRepo.Update(ctx, existing)
}

// GetAllShipments menarik daftar semua data dari database
func (u *shipmentUsecase) GetAllShipments(ctx context.Context) ([]domain.Shipment, error) {
	return u.shipmentRepo.FetchAll(ctx)
}

func (u *shipmentUsecase) GetShipmentByID(ctx context.Context, id string) (*domain.Shipment, error) {
	return u.shipmentRepo.FindByID(ctx, id)
}

func (u *shipmentUsecase) DeleteShipment(ctx context.Context, id string) error {
	return u.shipmentRepo.Delete(ctx, id)
}
