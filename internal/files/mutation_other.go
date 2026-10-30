//go:build !linux

package files

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/deletion"
)

type mutationState struct{}

func newMutationState(*Root) (*mutationState, error) { return nil, deletion.ErrWriteUnavailable }
func validateMutationRequest(mutationRequest) error  { return deletion.ErrWriteUnavailable }
func (*mutationState) execute(context.Context, mutationRequest) (mutationReply, error) {
	return mutationReply{}, deletion.ErrWriteUnavailable
}
func (*mutationState) closeRemoval()     {}
func (*mutationState) close() error      { return nil }
func mutationOperationError(error) error { return deletion.ErrWriteUnavailable }
