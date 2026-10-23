package app

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

// The app adapter owns the cross-capability composition. No path or catalog
// access is delegated to the inference container or to the autotag capability.
type automaticTagRuntime struct {
	content      *media.ContentService
	preprocessor semantic.EncodedImagePreprocessor
	client       *remoteml.TagClient
	model        *autotag.Model
}

func (r *automaticTagRuntime) Model() *autotag.Model {
	if r.model == nil {
		return nil
	}
	value := *r.model
	return &value
}
func (r *automaticTagRuntime) Verify(ctx context.Context) error {
	if r.client == nil {
		return autotag.ErrUnavailable
	}
	return automaticTagInferenceError(r.client.VerifyIdentity(ctx))
}
func (r *automaticTagRuntime) Analyze(ctx context.Context, asset autotag.Candidate) ([]curation.AutomaticLabel, error) {
	if r.client == nil || r.preprocessor == nil {
		return nil, autotag.ErrUnavailable
	}
	source, err := r.content.Open(ctx, asset.ID)
	if err != nil {
		return nil, automaticTagSourceError(err)
	}
	if source.LibraryID != asset.LibraryID || source.Fingerprint.String() != asset.SourceFingerprint {
		source.File.Close()
		return nil, autotag.ErrSourceChanged
	}
	kind, format, _, supported := media.ClassifyPath(source.Name)
	if !supported || (kind != media.KindImage && kind != media.KindAnimated) {
		source.File.Close()
		return nil, autotag.ErrSourceChanged
	}
	encoded, err := r.preprocessor.PrepareRemoteSemanticImage(ctx, source.File, format)
	closeErr := source.File.Close()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || closeErr != nil {
		return nil, autotag.ErrImage
	}
	values, err := r.client.Analyze(ctx, encoded)
	if err != nil {
		return nil, automaticTagInferenceError(err)
	}
	// Reopen through the anchored content owner after inference; checking only
	// SQLite would allow a replaced on-disk source to receive the old result.
	current, err := r.content.Open(ctx, asset.ID)
	if err != nil {
		return nil, automaticTagSourceError(err)
	}
	defer current.File.Close()
	if current.LibraryID != asset.LibraryID || current.Fingerprint.String() != asset.SourceFingerprint {
		return nil, autotag.ErrSourceChanged
	}
	labels := make([]curation.AutomaticLabel, len(values))
	for i, v := range values {
		labels[i] = curation.AutomaticLabel{Name: v.Name, Confidence: v.Confidence}
	}
	return labels, nil
}
func automaticTagInferenceError(err error) error {
	if errors.Is(err, remoteml.ErrBusy) {
		return autotag.ErrBusy
	}
	return err
}
func automaticTagSourceError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, media.ErrContentSourceChanged), errors.Is(err, media.ErrContentAssetNotFound):
		return autotag.ErrSourceChanged
	case errors.Is(err, media.ErrContentSourceOffline):
		return autotag.ErrOffline
	default:
		return autotag.ErrImage
	}
}
