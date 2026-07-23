package files

import (
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
)

// Normalize validates a slash-separated path relative to a media root.
//
// The empty string is the canonical representation of the root itself. The
// returned path preserves the caller's bytes; percent-decoded forms are checked
// only to reject inputs that another layer could reinterpret as traversal.
func Normalize(relative string) (string, error) {
	normalized, err := pathpolicy.Normalize(relative)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidPath, err)
	}
	return normalized, nil
}
