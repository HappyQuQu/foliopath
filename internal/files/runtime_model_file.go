package files

import (
	"os"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

type runtimeModelFile struct {
	file *os.File
	size int64
	path string
}

func (file *runtimeModelFile) Close() error        { return file.file.Close() }
func (file *runtimeModelFile) RuntimePath() string { return file.path }
func (file *runtimeModelFile) Size() int64         { return file.size }

func newRuntimeModelFile(file *os.File, size int64) (aimodel.RuntimeModelFile, error) {
	path, err := runtimeFilePath(file)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &runtimeModelFile{file: file, size: size, path: path}, nil
}

var _ aimodel.RuntimeModelFile = (*runtimeModelFile)(nil)
