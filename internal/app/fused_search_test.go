package app

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/search"
)

type fusedEncoderStub struct{ err error }

func (s fusedEncoderStub) EncodeSemanticText(context.Context, string, string) ([]float32, error) {
	return nil, s.err
}
func TestFusedEncoderClassifiesOutageButPreservesIdentityAndCallerCancellation(t *testing.T) {
	for _, err := range []error{remoteml.ErrUnavailable, remoteml.ErrBusy, context.DeadlineExceeded} {
		_, got := (fusedTextEncoder{fusedEncoderStub{err}}).EncodeSemanticText(context.Background(), "generation", "car")
		if !errors.Is(got, search.ErrUnavailable) {
			t.Fatalf("temporary outage not classified: %v", got)
		}
	}
	for _, err := range []error{remoteml.ErrIdentityMismatch, remoteml.ErrInvalidResponse} {
		_, got := (fusedTextEncoder{fusedEncoderStub{err}}).EncodeSemanticText(context.Background(), "generation", "car")
		if errors.Is(got, search.ErrUnavailable) || !errors.Is(got, err) {
			t.Fatalf("invalid identity/output hidden: %v", got)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, got := (fusedTextEncoder{fusedEncoderStub{context.Canceled}}).EncodeSemanticText(ctx, "generation", "car")
	if !errors.Is(got, context.Canceled) || errors.Is(got, search.ErrUnavailable) {
		t.Fatalf("cancellation hidden: %v", got)
	}
}
