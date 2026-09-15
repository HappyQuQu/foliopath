package semantic

import (
	"context"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

// DeploymentActivationCommit binds a verified service space and activates it in
// the existing activation transaction. It does not establish deployment trust.
type DeploymentActivationCommit struct {
	Activation aimodel.ActivationCommit
	Identity   SpaceIdentity
}

type DeploymentActivationRepository interface {
	CommitDeploymentActivation(context.Context, DeploymentActivationCommit) (aimodel.Operation, error)
}
