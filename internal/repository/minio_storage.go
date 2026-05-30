package repository

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"trucking-app/internal/domain"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioRepository struct {
	client     *minio.Client
	bucketName string
}

// NewMinioRepository menginisialisasi koneksi awal ke MinIO Server
func NewMinioRepository() (domain.DocumentRepository, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucketName := os.Getenv("MINIO_BUCKET_NAME")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	// 1. Konek ke MinIO
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	// 2. Buat bucket otomatis jika belum ada di MinIO
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
		log.Printf("🎉 Bucket '%s' berhasil dibuat otomatis di MinIO!", bucketName)
	}

	return &minioRepository{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// Upload menerima file berupa byte data lalu mengirimkannya ke bucket MinIO
func (m *minioRepository) Upload(ctx context.Context, fileName string, fileData []byte, contentType string) (string, error) {
	reader := bytes.NewReader(fileData)
	size := int64(len(fileData))

	_, err := m.client.PutObject(ctx, m.bucketName, fileName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	// Mengembalikan nama file yang sukses disimpan untuk dicatat di Postgres
	return fileName, nil
}

// GetObject mengambil file dari MinIO berbentuk stream data reader
func (m *minioRepository) GetObject(ctx context.Context, fileName string) (io.ReadCloser, error) {
	return m.client.GetObject(ctx, m.bucketName, fileName, minio.GetObjectOptions{})
}