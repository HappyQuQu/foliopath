package app

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"time"

	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

var errDatabaseCheckPlatform = errors.New("database check must run inside the Linux application container")

func checkDatabase(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return errDatabaseCheckPlatform
	}
	bounded, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return sqlitestore.CheckIntegrity(bounded, filepath.Join(persistentDataRoot, databaseFilename))
}
