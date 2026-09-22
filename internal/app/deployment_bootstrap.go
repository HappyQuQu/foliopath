package app

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
)

type deploymentBootstrap struct {
	client     *remoteml.Client
	models     *aimodel.Service
	admission  *aimodel.ActivationAdmissionService
	config     deploymentConfiguration
	faceClient *remoteml.FaceClient
}

func (b *deploymentBootstrap) Run(ctx context.Context) error {
	if b.faceClient == nil {
		defer b.client.Close()
	}
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}
		// Failure is confined to AI. Durable activation failures remain in jobs;
		// unavailable models retain their generation and embeddings.
		_ = b.reconcile(ctx)
		timer.Reset(30 * time.Second)
	}
}

func (b *deploymentBootstrap) reconcile(ctx context.Context) error {
	pkg := b.config.model()
	verify := b.client.VerifyIdentity
	register := b.models.RegisterDeployment
	if b.faceClient != nil {
		pkg = b.config.Face.model()
		verify = b.faceClient.VerifyIdentity
		register = b.models.RegisterFaceDeployment
	}
	snapshot, err := b.models.List(ctx)
	if err != nil {
		return err
	}
	var current *aimodel.Model
	for i := range snapshot.Items {
		model := &snapshot.Items[i]
		if model.StorageMode != aimodel.StorageService || model.Package.Purpose != pkg.Purpose {
			continue
		}
		if model.Package.ContentHash == pkg.ContentHash {
			current = model
			continue
		}
		// Only the configured identity has a runtime in this process. A previous
		// deployment must not remain advertised as available if the replacement
		// fails verification; retain its generation and all derived/manual data.
		if model.State == aimodel.StateAvailable {
			if _, err := b.models.SetAvailability(ctx, model.ID, model.AvailabilityRevision, false); err != nil {
				return err
			}
		}
	}
	if err := verify(ctx); err != nil {
		// A saturated local inference budget is not evidence of service failure.
		if errors.Is(err, remoteml.ErrBusy) {
			return err
		}
		if ctx.Err() == nil && current != nil && current.State == aimodel.StateAvailable {
			_, markErr := b.models.SetAvailability(ctx, current.ID, current.AvailabilityRevision, false)
			return errors.Join(err, markErr)
		}
		return err
	}
	model, _, err := register(ctx, pkg)
	if err != nil {
		return err
	}
	if model.State != aimodel.StateAvailable {
		model, err = b.models.SetAvailability(ctx, model.ID, model.AvailabilityRevision, true)
		if err != nil {
			return err
		}
	}
	activeID := snapshot.ActiveModelID
	if b.faceClient != nil {
		activeID = snapshot.ActiveFaceModelID
	}
	if activeID == model.ID {
		return nil
	}
	// Three durable attempts per model availability revision, including across
	// process restarts; never create an unbounded sequence of retries.
	for attempt := 0; attempt < 3; attempt++ {
		key := "deployment-" + model.Package.ContentHash + "-" + strconv.FormatInt(model.AvailabilityRevision, 10) + "-" + strconv.Itoa(attempt)
		result, err := b.admission.StartDeploymentActivation(ctx, model, key)
		if err != nil {
			return err
		}
		if result.Operation.State != aimodel.OperationFailed && result.Operation.State != aimodel.OperationCancelled {
			return nil
		}
	}
	return aimodel.ErrInferenceRuntimeUnavailable
}
