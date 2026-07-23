// Package pathpolicy owns the lexical policy shared by media-library
// configuration, scanning, and filesystem access. It validates only relative
// path text; opening and identity checks remain in the files adapter.
package pathpolicy

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

var ErrInvalid = errors.New("unsafe relative path")

// Normalize validates a slash-separated relative path while preserving its
// original bytes. Percent-decoded views are inspected only for ambiguous path
// controls and are never returned for filesystem access.
func Normalize(relative string) (string, error) {
	if relative == "" {
		return "", nil
	}
	if !utf8.ValidString(relative) {
		return "", invalid("path is not valid UTF-8")
	}
	if strings.IndexByte(relative, 0) >= 0 {
		return "", invalid("path contains NUL")
	}
	if strings.Contains(relative, `\`) {
		return "", invalid("backslash is not a path separator")
	}
	if path.IsAbs(relative) {
		return "", invalid("absolute paths are not allowed")
	}

	for _, component := range strings.Split(relative, "/") {
		if component == "" {
			return "", invalid("empty path component")
		}
		if component == "." || component == ".." {
			return "", invalid("dot path component")
		}
		if err := validatePercentViews(component); err != nil {
			return "", err
		}
	}
	if path.Clean(relative) != relative {
		return "", invalid("path is not canonical")
	}
	return relative, nil
}

func validatePercentViews(component string) error {
	view := component
	for {
		decoded, changed := decodePercentOnce(view)
		if !changed {
			return nil
		}
		if !utf8.ValidString(decoded) {
			return invalid("percent decoding produces invalid UTF-8")
		}
		if strings.IndexByte(decoded, 0) >= 0 {
			return invalid("percent decoding produces NUL")
		}
		if strings.ContainsAny(decoded, `/\`) {
			return invalid("percent decoding produces a path separator")
		}
		if decoded == "." || decoded == ".." {
			return invalid("percent decoding produces a dot component")
		}
		view = decoded
	}
}

func decodePercentOnce(value string) (string, bool) {
	first := -1
	for index := 0; index+2 < len(value); index++ {
		if value[index] == '%' && isHex(value[index+1]) && isHex(value[index+2]) {
			first = index
			break
		}
	}
	if first < 0 {
		return value, false
	}

	var decoded strings.Builder
	decoded.Grow(len(value))
	decoded.WriteString(value[:first])
	for index := first; index < len(value); {
		if index+2 < len(value) && value[index] == '%' && isHex(value[index+1]) && isHex(value[index+2]) {
			decoded.WriteByte(fromHex(value[index+1])<<4 | fromHex(value[index+2]))
			index += 3
			continue
		}
		decoded.WriteByte(value[index])
		index++
	}
	return decoded.String(), true
}

func isHex(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}

func fromHex(value byte) byte {
	switch {
	case value >= '0' && value <= '9':
		return value - '0'
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10
	default:
		return value - 'A' + 10
	}
}

func invalid(reason string) error {
	return fmt.Errorf("%w: %s", ErrInvalid, reason)
}
