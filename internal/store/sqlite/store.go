// Package sqlite implements FolioPath capability repositories with SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HappyQuQu/foliopath/internal/store/sqlite/dbgen"
	_ "modernc.org/sqlite"
)

var storeOpenMu sync.Mutex

const (
	defaultBusyTimeout = 5 * time.Second
	defaultBatchLimit  = 500
)

type Options struct {
	BusyTimeout        time.Duration
	MaxOpenConnections int
	MaxBatchSize       int
	Now                func() time.Time
}

type Store struct {
	db           *sql.DB
	queries      *dbgen.Queries
	writeGate    chan struct{}
	maxBatchSize int
	now          func() time.Time
}

func Open(ctx context.Context, filename string, options Options) (*Store, error) {
	if filename == "" || filename == ":memory:" {
		return nil, errors.New("sqlite requires a file-backed database")
	}
	if options.BusyTimeout == 0 {
		options.BusyTimeout = defaultBusyTimeout
	}
	if options.BusyTimeout < time.Millisecond {
		return nil, errors.New("sqlite busy timeout must be at least one millisecond")
	}
	if options.MaxOpenConnections == 0 {
		options.MaxOpenConnections = 4
	}
	if options.MaxOpenConnections < 1 {
		return nil, errors.New("sqlite max open connections must be positive")
	}
	if options.MaxBatchSize == 0 {
		options.MaxBatchSize = defaultBatchLimit
	}
	if options.MaxBatchSize < 1 || options.MaxBatchSize > 1000 {
		return nil, errors.New("sqlite max batch size must be between 1 and 1000")
	}
	if options.Now == nil {
		options.Now = time.Now
	}

	// Startup may be invoked concurrently by tests or accidental duplicate app
	// composition. Connection PRAGMAs and the first migration both take SQLite
	// locks, so serialize initialization within this single-instance process.
	storeOpenMu.Lock()
	defer storeOpenMu.Unlock()

	db, err := sql.Open("sqlite", buildDSN(filename, options.BusyTimeout))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(options.MaxOpenConnections)
	db.SetMaxIdleConns(options.MaxOpenConnections)

	closeOnError := func(err error) (*Store, error) {
		_ = db.Close()
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return closeOnError(fmt.Errorf("ping sqlite: %w", err))
	}
	if err := verifyPragmas(ctx, db, options.BusyTimeout); err != nil {
		return closeOnError(err)
	}
	if err := applyMigrations(ctx, db); err != nil {
		return closeOnError(err)
	}

	return &Store{
		db:           db,
		queries:      dbgen.New(db),
		writeGate:    make(chan struct{}, 1),
		maxBatchSize: options.MaxBatchSize,
		now:          options.Now,
	}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) withWriteTx(ctx context.Context, operation func(*sql.Tx) error) error {
	select {
	case s.writeGate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.writeGate }()

	// _txlock=immediate in the DSN makes this BEGIN IMMEDIATE. The database
	// lock and schema constraints protect generation allocation across
	// processes; writeGate only provides fair, context-aware local queuing.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sqlite write transaction: %w", err)
	}
	if err := operation(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback sqlite transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite write transaction: %w", err)
	}
	return nil
}

func (s *Store) nowMS() int64 {
	return s.now().UTC().UnixMilli()
}

func buildDSN(filename string, busyTimeout time.Duration) string {
	dsn := &url.URL{Scheme: "file", Path: filename}
	query := dsn.Query()
	query.Add("_pragma", "busy_timeout("+strconv.FormatInt(busyTimeout.Milliseconds(), 10)+")")
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(NORMAL)")
	query.Set("_txlock", "immediate")
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func verifyPragmas(ctx context.Context, db *sql.DB, busyTimeout time.Duration) error {
	var sqliteVersion string
	if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&sqliteVersion); err != nil {
		return fmt.Errorf("read sqlite version: %w", err)
	}
	if !hasWALResetFix(sqliteVersion) {
		return fmt.Errorf("sqlite version %q lacks the required WAL-reset fix", sqliteVersion)
	}

	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("read sqlite journal_mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("sqlite journal_mode is %q, want wal", journalMode)
	}

	var foreignKeys int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return fmt.Errorf("read sqlite foreign_keys: %w", err)
	}
	if foreignKeys != 1 {
		return errors.New("sqlite foreign_keys pragma is disabled")
	}

	var actualBusyTimeout int64
	if err := db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&actualBusyTimeout); err != nil {
		return fmt.Errorf("read sqlite busy_timeout: %w", err)
	}
	if actualBusyTimeout != busyTimeout.Milliseconds() {
		return fmt.Errorf("sqlite busy_timeout is %dms, want %dms", actualBusyTimeout, busyTimeout.Milliseconds())
	}

	var synchronous int
	if err := db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous); err != nil {
		return fmt.Errorf("read sqlite synchronous: %w", err)
	}
	if synchronous != 1 {
		return fmt.Errorf("sqlite synchronous is %d, want NORMAL (1)", synchronous)
	}
	return nil
}

// hasWALResetFix accepts the mainline fix and the two official backports.
// SQLite versions 3.7.0 through 3.51.2 otherwise contain the WAL-reset bug.
func hasWALResetFix(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil || major != 3 {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return false
	}

	switch {
	case minor > 51:
		return true
	case minor == 51:
		return patch >= 3
	case minor == 50:
		return patch >= 7
	case minor == 44:
		return patch >= 6
	default:
		return false
	}
}
