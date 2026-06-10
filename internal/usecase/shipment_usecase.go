package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	"trucking-app/internal/domain"

	"github.com/google/uuid"
)

type shipmentUsecase struct {
	shipmentRepo domain.ShipmentRepository
	minioRepo    domain.MinioRepository
	docRepo      domain.DocumentRepository
}

// NewShipmentUsecase melahirkan objek usecase baru
func NewShipmentUsecase(repo domain.ShipmentRepository, minioRepo domain.MinioRepository, docRepo domain.DocumentRepository) domain.ShipmentUsecase {
	return &shipmentUsecase{
		shipmentRepo: repo,
		minioRepo:    minioRepo,
		docRepo:      docRepo, // Daftarkan di sini
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

func (u *shipmentUsecase) UpdateShipmentData(ctx context.Context, s *domain.Shipment) error {
	// 1. Ambil data kargo asli dari DB terlebih dahulu untuk tahu "Selling Rate" / "Amount" awal
	existingShipment, err := u.shipmentRepo.FindByID(ctx, s.ID)
	if err != nil {
		return fmt.Errorf("kargo tidak ditemukan: %w", err)
	}

	s.ItemDescription = existingShipment.ItemDescription
	s.Origin = existingShipment.Origin
	s.Destination = existingShipment.Destination
	s.Rate = existingShipment.Rate
	s.Qty = existingShipment.Qty
	s.Amount = existingShipment.Amount

	// 2. Hitung Ulang Rumus Keuangan Logistik
	// Gross Profit = Total Pendapatan - (Modal Vendor + Rate LOLO)
	s.GrossProfit = existingShipment.Amount - s.BuyingPrice

	// Profit Percentage = (Gross Profit / Total Pendapatan) * 100
	if existingShipment.Amount > 0 {
		s.ProfitPercentage = int((float64(s.GrossProfit) * 100) / float64(existingShipment.Amount))
	}

	// Pertahankan data esensial yang tidak diubah di modal finance

	// 3. Tembak perintah update ke repository
	return u.shipmentRepo.Update(ctx, s)
}

func (u *shipmentUsecase) UpdateShipmentStatus(ctx context.Context, id string, status string) error {
	// 1. Validasi bisnis tambahan (Opsional)
	if id == "" {
		return errors.New("id tidak boleh kosong")
	}

	// 2. Teruskan data ke layer repository
	err := u.shipmentRepo.UpdateStatus(ctx, id, status)
	if err != nil {
		return err
	}

	return nil
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

func (u *shipmentUsecase) UploadDocument(ctx context.Context, shipmentID, fileName, contentType string, fileData []byte) (*domain.ShipmentDocument, error) {
	if shipmentID == "" {
		return nil, errors.New("shipment ID tidak boleh kosong")
	}

	uniqueName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileName)

	objectPath, err := u.minioRepo.Upload(ctx, uniqueName, fileData, contentType)
	if err != nil {
		return nil, fmt.Errorf("gagal upload ke minio storage: %w", err)
	}

	newDoc := &domain.ShipmentDocument{
		ID:           uuid.New().String(),
		ShipmentID:   shipmentID,
		FileName:     fileName,
		DocumentPath: objectPath,
		CreatedAt:    time.Now(),
	}

	// Memanggil u.docRepo.Save (bukan SaveDoc lagi) sesuai interface baru
	if err := u.docRepo.Save(ctx, newDoc); err != nil {
		_ = u.minioRepo.DeleteFile(ctx, uniqueName)
		return nil, fmt.Errorf("gagal menyimpan data dokumen ke database: %w", err)
	}

	return newDoc, nil
}

func (u *shipmentUsecase) GetShipmentDocuments(ctx context.Context, shipmentID string) ([]domain.ShipmentDocument, error) {
	if shipmentID == "" {
		return nil, errors.New("shipment ID tidak boleh kosong")
	}

	// Memanggil u.docRepo.FindByShipmentID (bukan FindByShipmentIDDoc lagi)
	return u.docRepo.FindByShipmentID(ctx, shipmentID)
}

func (u *shipmentUsecase) DeleteShipmentDocument(ctx context.Context, docID string) error {
	// 1. Ambil data SATU dokumen tunggal berdasarkan ID-nya
	doc, err := u.docRepo.FindDocByID(ctx, docID) // <-- Sekarang memanggil fungsi baru
	if err != nil {
		return fmt.Errorf("dokumen tidak ditemukan: %w", err)
	}

	// 2. Hapus file fisiknya dari MinIO (Sekarang doc.DocumentPath sudah aman dan tidak error lagi)
	err = u.minioRepo.DeleteFile(ctx, doc.DocumentPath)
	if err != nil {
		log.Printf("[Warning] Gagal menghapus file di MinIO: %v", err)
	}

	// 3. Hapus record baris data dari Postgres menggunakan fungsi DeleteDoc milikmu
	return u.docRepo.DeleteDoc(ctx, docID)
}
