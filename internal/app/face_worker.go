package app

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/media"
)

type faceJobQueue struct{ database *databaseService }

func (q faceJobQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return q.database.RecoverExpiredFaceJobs(ctx, time.Now().UTC())
}
func (q faceJobQueue) Claim(ctx context.Context, lease time.Duration) (face.AnalysisJob, bool, error) {
	return q.database.ClaimFaceJob(ctx, time.Now().UTC(), lease)
}
func (q faceJobQueue) RefreshLease(ctx context.Context, job face.AnalysisJob, lease time.Duration) (bool, error) {
	return q.database.RefreshFaceJobLease(ctx, job, time.Now().UTC(), lease)
}

type faceClearQueue struct{ database *databaseService }

func (q faceClearQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return q.database.RecoverExpiredFaceClears(ctx, time.Now().UTC())
}
func (q faceClearQueue) Claim(ctx context.Context, lease time.Duration) (face.ClearJob, bool, error) {
	return q.database.ClaimFaceClear(ctx, time.Now().UTC(), lease)
}
func (q faceClearQueue) RefreshLease(ctx context.Context, job face.ClearJob, lease time.Duration) (bool, error) {
	return q.database.RefreshFaceClearLease(ctx, job, time.Now().UTC(), lease)
}

type faceContentSource struct{ content *media.ContentService }

func (s faceContentSource) OpenFaceAsset(ctx context.Context, libraryID, assetID int64) (face.Asset, error) {
	if s.content == nil || libraryID < 1 || assetID < 1 {
		return face.Asset{}, face.ErrInvalidInput
	}
	content, err := s.content.Open(ctx, assetID)
	if err != nil {
		if errors.Is(err, media.ErrContentSourceChanged) || errors.Is(err, media.ErrContentAssetNotFound) {
			return face.Asset{}, face.ErrSourceChanged
		}
		return face.Asset{}, err
	}
	if content.LibraryID != libraryID {
		_ = content.File.Close()
		return face.Asset{}, face.ErrSourceChanged
	}
	_, format, _, ok := media.ClassifyPath(content.Name)
	if !ok {
		_ = content.File.Close()
		return face.Asset{}, face.ErrInvalidInput
	}
	return face.Asset{File: content.File, Format: format, SourceFingerprint: content.Fingerprint.String()}, nil
}

var _ jobs.LeaseQueue[face.AnalysisJob] = faceJobQueue{}
var _ jobs.LeaseQueue[face.ClearJob] = faceClearQueue{}
var _ face.AssetSource = faceContentSource{}
