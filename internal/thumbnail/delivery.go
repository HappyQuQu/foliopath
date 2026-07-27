package thumbnail

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const ThumbnailRetryAfterMS = 1000

var ErrCacheEntryMissing = errors.New("thumbnail cache entry missing")

type DeliveryStatus string

const (
	DeliveryQueued  DeliveryStatus = "queued"
	DeliveryRunning DeliveryStatus = "running"
	DeliveryReady   DeliveryStatus = "ready"
	DeliveryFailed  DeliveryStatus = "failed"
	DeliveryOffline DeliveryStatus = "offline"
)

type DeliveryState struct {
	AssetID           int64
	SourceFingerprint media.SourceFingerprint
	Status            DeliveryStatus
	ErrorCode         media.ProcessingErrorCode
	CacheRelativePath string
	ByteSize          int64
}

type DeliveryRepository interface {
	GetThumbnailDelivery(context.Context, int64) (DeliveryState, error)
	TouchThumbnail(context.Context, int64, media.SourceFingerprint, string) error
	RequeueMissingThumbnail(context.Context, DeliveryState) error
}

type CacheContent struct {
	Reader   io.ReadSeekCloser
	ByteSize int64
}

type CacheReader interface {
	Open(context.Context, string) (CacheContent, error)
}

type DeliveryWaker interface {
	Wake()
}

type Delivery struct {
	Status       DeliveryStatus
	ErrorCode    media.ProcessingErrorCode
	Content      io.ReadSeekCloser
	ContentBytes int64
	ETag         string
	RetryAfterMS int
}

type DeliveryService struct {
	repository DeliveryRepository
	cache      CacheReader
	waker      DeliveryWaker
}

func NewDeliveryService(
	repository DeliveryRepository,
	cache CacheReader,
	waker DeliveryWaker,
) (*DeliveryService, error) {
	if repository == nil || cache == nil || waker == nil {
		return nil, errors.New("thumbnail delivery dependencies are required")
	}
	return &DeliveryService{repository: repository, cache: cache, waker: waker}, nil
}

func (service *DeliveryService) Get(ctx context.Context, assetID int64) (Delivery, error) {
	if err := ctx.Err(); err != nil {
		return Delivery{}, err
	}
	if assetID <= 0 {
		return Delivery{}, ErrAssetNotFound
	}
	state, err := service.repository.GetThumbnailDelivery(ctx, assetID)
	if err != nil {
		return Delivery{}, err
	}
	switch state.Status {
	case DeliveryOffline, DeliveryFailed:
		if state.ErrorCode == "" {
			return Delivery{}, ErrInvalidState
		}
		return Delivery{Status: state.Status, ErrorCode: state.ErrorCode}, nil
	case DeliveryQueued, DeliveryRunning:
		return Delivery{Status: state.Status, RetryAfterMS: ThumbnailRetryAfterMS}, nil
	case DeliveryReady:
	default:
		return Delivery{}, ErrInvalidState
	}
	if state.CacheRelativePath == "" || state.ByteSize <= 0 ||
		!state.SourceFingerprint.Valid() {
		return Delivery{}, ErrInvalidState
	}
	content, err := service.cache.Open(ctx, state.CacheRelativePath)
	if errors.Is(err, ErrCacheEntryMissing) {
		if repairErr := service.repository.RequeueMissingThumbnail(ctx, state); repairErr != nil {
			return Delivery{}, repairErr
		}
		service.waker.Wake()
		return Delivery{Status: DeliveryQueued, RetryAfterMS: ThumbnailRetryAfterMS}, nil
	}
	if err != nil {
		return Delivery{}, err
	}
	if content.Reader == nil {
		return Delivery{}, ErrInvalidState
	}
	if content.ByteSize != state.ByteSize {
		_ = content.Reader.Close()
		if repairErr := service.repository.RequeueMissingThumbnail(ctx, state); repairErr != nil {
			return Delivery{}, repairErr
		}
		service.waker.Wake()
		return Delivery{Status: DeliveryQueued, RetryAfterMS: ThumbnailRetryAfterMS}, nil
	}
	if err := service.repository.TouchThumbnail(
		ctx, state.AssetID, state.SourceFingerprint, state.CacheRelativePath,
	); err != nil {
		_ = content.Reader.Close()
		return Delivery{}, err
	}
	validator := sha256.Sum256([]byte(
		state.SourceFingerprint.String() + "\x00" + state.CacheRelativePath,
	))
	return Delivery{
		Status: DeliveryReady, Content: content.Reader,
		ContentBytes: content.ByteSize,
		ETag:         fmt.Sprintf(`"thumb-%x"`, validator[:16]),
	}, nil
}
