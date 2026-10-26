package face

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
)

var reviewIdempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,200}$`)
var reviewDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ReviewIdentity converts the opaque HTTP idempotency key and typed request
// into non-reversible identifiers safe for persistence.
func ReviewIdentity(idempotencyKey string, request any) (string, string, error) {
	if !reviewIdempotencyKeyPattern.MatchString(idempotencyKey) || request == nil {
		return "", "", ErrInvalidReview
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return "", "", err
	}
	keyDigest := sha256.Sum256([]byte("foliopath:face-review-key:v1\x00" + idempotencyKey))
	requestDigest := sha256.Sum256(append([]byte("foliopath:face-review-request:v1\x00"), encoded...))
	return "freview_" + hex.EncodeToString(keyDigest[:20]), hex.EncodeToString(requestDigest[:]), nil
}

func ValidReviewDigest(value string) bool { return reviewDigestPattern.MatchString(value) }
