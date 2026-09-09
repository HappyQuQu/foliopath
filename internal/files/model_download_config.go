package files

import (
	"io"
	"io/fs"
	"os"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

// ReadModelDownloadCatalog reads the fixed deployment-owned config, never media.
func ReadModelDownloadCatalog(name string) ([]byte, error) {
	file, err := os.Open(name)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > aimodel.MaxDownloadCatalogBytes {
		return nil, fs.ErrInvalid
	}
	data, err := io.ReadAll(io.LimitReader(file, aimodel.MaxDownloadCatalogBytes+1))
	if len(data) > aimodel.MaxDownloadCatalogBytes {
		return nil, fs.ErrInvalid
	}
	return data, err
}
