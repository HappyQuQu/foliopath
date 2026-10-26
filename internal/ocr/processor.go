package ocr

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

var (
	ErrSourceChanged      = errors.New("OCR source changed")
	ErrSourceUnavailable  = errors.New("OCR source unavailable")
	ErrRuntimeUnavailable = errors.New("OCR runtime unavailable")
	ErrRuntimeBusy        = errors.New("OCR runtime busy")
	ErrRuntimeIdentity    = errors.New("OCR runtime identity mismatch")
	ErrRuntimeOutput      = errors.New("OCR runtime output invalid")
	ErrRuntimeOutputLimit = errors.New("OCR runtime output budget exceeded")
)

type ContentOpener interface {
	Open(context.Context, int64) (media.Content, error)
}
type Runtime interface {
	Identity() Identity
	Recognize(context.Context, []byte) (Result, error)
}

type Analyzer struct {
	content     ContentOpener
	images      ImagePreprocessor
	runtime     Runtime
	fingerprint string
}

func NewAnalyzer(content ContentOpener, images ImagePreprocessor, runtime Runtime) (*Analyzer, error) {
	if content == nil || images == nil || runtime == nil {
		return nil, ErrInvalidPersistence
	}
	identity := runtime.Identity()
	fp, err := identity.Fingerprint()
	if err != nil || identity.InputContractSHA256 != InputContractSHA256() || identity.OutputContractSHA256 != OutputContractSHA256() {
		return nil, ErrInvalidIdentity
	}
	return &Analyzer{content: content, images: images, runtime: runtime, fingerprint: fp}, nil
}
func sourceError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, media.ErrContentAssetNotFound) || errors.Is(err, media.ErrContentSourceChanged) {
		return ErrSourceChanged
	}
	return ErrSourceUnavailable
}
func (a *Analyzer) open(ctx context.Context, libraryID int64, c Candidate) (media.Content, media.Format, error) {
	source, err := a.content.Open(ctx, c.AssetID)
	if err != nil {
		return source, "", sourceError(ctx, err)
	}
	kind, format, _, ok := media.ClassifyPath(source.Name)
	if source.File == nil || source.LibraryID != libraryID || source.Fingerprint.String() != c.SourceFingerprint || !ok || (kind != media.KindImage && kind != media.KindAnimated) {
		if source.File != nil {
			_ = source.File.Close()
		}
		return media.Content{}, "", ErrSourceChanged
	}
	return source, format, nil
}

// Analyze keeps original access in the existing content/files boundary and
// closes all files before returning. Even definitive input failures are checked
// against the current source before they can become durable asset failures.
func (a *Analyzer) Analyze(ctx context.Context, libraryID int64, c Candidate) (StoredResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	value := StoredResult{Candidate: c, Fingerprint: a.fingerprint}
	source, format, err := a.open(ctx, libraryID, c)
	if err != nil {
		return value, err
	}
	prepared, err := a.images.PrepareOCRImage(ctx, source.File, format)
	fromPreprocessor := err != nil
	closeErr := source.File.Close()
	if ctx.Err() != nil {
		return value, ctx.Err()
	}
	if closeErr != nil {
		return value, ErrSourceUnavailable
	}
	if err != nil && !errors.Is(err, ErrInvalidMedia) && !errors.Is(err, ErrInputLimit) {
		err = ErrPreprocessFailed
	}
	if err == nil {
		if prepared.ContractSHA256 != InputContractSHA256() || len(prepared.PNG) == 0 || len(prepared.PNG) > MaxImageBytes {
			return value, ErrPreprocessFailed
		}
		value.Width, value.Height = prepared.Width, prepared.Height
		// Validate dimensions independently before any inference request.
		check := value
		check.Outcome = "ready"
		check.Result.Lines = []Line{}
		if _, invalid := ValidateStoredResult(check); invalid != nil {
			return value, ErrPreprocessFailed
		}
		value.Result, err = a.runtime.Recognize(ctx, prepared.PNG)
	}
	if ctx.Err() != nil {
		return value, ctx.Err()
	}
	// Source mismatch/IO outranks model output and input-failure classification.
	current, _, checkErr := a.open(ctx, libraryID, c)
	if checkErr != nil {
		return value, checkErr
	}
	if closeErr := current.File.Close(); closeErr != nil {
		return value, ErrSourceUnavailable
	}
	switch {
	case err == nil:
		value.Outcome = "ready"
	case fromPreprocessor && errors.Is(err, ErrInvalidMedia):
		value.Outcome, value.ErrorCode = "failed", "invalid_media"
		value.Width, value.Height = 0, 0
		value.Result.Lines = nil
	case fromPreprocessor && errors.Is(err, ErrInputLimit):
		value.Outcome, value.ErrorCode = "failed", "input_limit"
		value.Width, value.Height = 0, 0
		value.Result.Lines = nil
	default:
		return value, err
	}
	normalized, err := ValidateStoredResult(value)
	if err != nil {
		return StoredResult{}, ErrRuntimeOutput
	}
	return normalized, nil
}
