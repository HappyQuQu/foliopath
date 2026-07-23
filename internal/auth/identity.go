package auth

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	minPasswordRunes = 12
	maxPasswordRunes = 128
	maxUsernameRunes = 64
	maxDisplayRunes  = 128
)

func NormalizeUsername(value string) (string, string, error) {
	if !validUTF8Length(value, 1, maxUsernameRunes) {
		return "", "", ErrInvalidUsername
	}

	normalized := norm.NFKC.String(value)
	if !validUTF8Length(normalized, 1, maxUsernameRunes) {
		return "", "", ErrInvalidUsername
	}
	for _, character := range normalized {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return "", "", ErrInvalidUsername
		}
	}

	key := cases.Fold().String(normalized)
	if key == "" {
		return "", "", ErrInvalidUsername
	}
	return normalized, key, nil
}

func NormalizeDisplayName(value string) (string, error) {
	normalized := norm.NFC.String(strings.TrimSpace(value))
	if !validUTF8Length(normalized, 1, maxDisplayRunes) {
		return "", ErrInvalidDisplayName
	}
	for _, character := range normalized {
		if unicode.IsControl(character) {
			return "", ErrInvalidDisplayName
		}
	}
	return normalized, nil
}

func ValidatePassword(value string) error {
	if !validUTF8Length(value, minPasswordRunes, maxPasswordRunes) {
		return ErrInvalidPassword
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return ErrInvalidPassword
		}
	}
	return nil
}

func validUTF8Length(value string, minimum, maximum int) bool {
	if !utf8.ValidString(value) {
		return false
	}
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}
