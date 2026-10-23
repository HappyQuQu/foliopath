package semantic

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	MaxQueryRunes      = 512
	TextSequenceLength = 64
	SigLIPPadTokenID   = int64(1)
	SigLIPEOSTokenID   = int64(1)
)

var ErrInvalidQuery = errors.New("invalid semantic query")

const asciiPunctuation = `!"#$%&'()*+,-./:;<=>?@[\]^_` + "`" + `{|}~`

var reservedSigLIPControlTokens = [...]string{"</s>", "<unk>"}

// CanonicalizeQuery implements the text transform used by the reviewed
// SigLIP tokenizer before SentencePiece encoding: per-code-point Unicode
// lowercase, removal of ASCII punctuation, Unicode whitespace collapse, and
// outer trim. Transformers applies lowercase through a non-greedy regex one
// code point at a time, which intentionally avoids context-sensitive mappings
// such as Greek final sigma.
func CanonicalizeQuery(value string) (string, error) {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > MaxQueryRunes {
		return "", ErrInvalidQuery
	}
	lower := cases.Lower(language.Und)
	var lowered strings.Builder
	lowered.Grow(len(value))
	for _, source := range value {
		lowered.WriteString(lower.String(string(source)))
	}
	loweredValue := lowered.String()
	for _, token := range reservedSigLIPControlTokens {
		if strings.Contains(loweredValue, token) {
			return "", ErrInvalidQuery
		}
	}
	var result strings.Builder
	result.Grow(len(value))
	pendingSpace := false
	for _, current := range loweredValue {
		if current <= unicode.MaxASCII && strings.ContainsRune(asciiPunctuation, current) {
			continue
		}
		if unicode.IsSpace(current) {
			if result.Len() != 0 {
				pendingSpace = true
			}
			continue
		}
		if pendingSpace {
			result.WriteByte(' ')
			pendingSpace = false
		}
		result.WriteRune(current)
	}
	if result.Len() == 0 {
		return "", ErrInvalidQuery
	}
	return result.String(), nil
}
