package fuzz

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync"

	"piko.sh/piko"
	"piko.sh/piko/wdk/storage"
	"piko.sh/piko/wdk/storage/storage_provider_disk"
)

type StoreAction struct{ piko.ActionMetadata }
type StoredFile struct {
	Name        string
	Media       string
	Size        int64
	Digest      string
	RangeDigest string
}

var uploadService = sync.OnceValues(func() (storage.Service, error) {
	service, err := storage.GetDefaultService()
	if err != nil {
		return nil, err
	}
	provider, err := storage_provider_disk.NewDiskProvider(storage_provider_disk.Config{BaseDirectory: os.Getenv("PIKO_FUZZ_UPLOAD_DIR")})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	if err = service.RegisterProvider(ctx, "fuzz-upload", provider); err != nil {
		_ = provider.Close(ctx)
		return nil, err
	}
	service.RegisterPublicRepository("uploads", "public, max-age=60")
	return service, nil
})

func (a *StoreAction) Call(file piko.FileUpload) (StoredFile, error) {
	if file.Size < 4 || file.Size > 4096 {
		return StoredFile{}, fmt.Errorf("expected 4 to 4096 bytes")
	}
	ctx := a.Ctx()
	service, err := uploadService()
	if err != nil {
		return StoredFile{}, err
	}
	source, err := file.Open()
	if err != nil {
		return StoredFile{}, err
	}
	defer source.Close()
	if err = service.PutObject(ctx, "fuzz-upload", &storage.PutParams{
		Repository: "uploads", Key: "payload.bin", Reader: io.LimitReader(source, 4097),
		ContentType: file.ContentType, Size: file.Size,
	}); err != nil {
		return StoredFile{}, err
	}
	read := func(r *storage.ByteRange) (string, error) {
		saved, err := service.GetObject(ctx, "fuzz-upload", storage.GetParams{Repository: "uploads", Key: "payload.bin", ByteRange: r})
		if err != nil {
			return "", err
		}
		defer saved.Close()
		data, err := io.ReadAll(io.LimitReader(saved, 4097))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", sha256.Sum256(data)), nil
	}
	digest, err := read(nil)
	if err != nil {
		return StoredFile{}, err
	}
	ranged, err := read(&storage.ByteRange{Start: 2, End: file.Size - 2})
	if err != nil {
		return StoredFile{}, err
	}
	return StoredFile{Name: file.Name, Media: file.ContentType, Size: file.Size, Digest: digest, RangeDigest: ranged}, nil
}
