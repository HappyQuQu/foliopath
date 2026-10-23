package semantic

import (
	"errors"
	"strings"
	"testing"
)

func TestCanonicalizeQueryMatchesPinnedSigLIPReference(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "COSER in RED-GOLD armor!!!", want: "coser in redgold armor"},
		{input: "  blue\t hair\nportrait  ", want: "blue hair portrait"},
		{input: "İSTANBUL ΣΟΣ ẞ", want: "i̇stanbul σοσ ß"},
		{input: "Ａ／Ｂ，Ｃ！", want: "ａ／ｂ，ｃ！"},
		{input: `a/b\c_d.e,f;g:h?i!`, want: "abcdefghi"},
		{input: "x\u00a0\u2003y", want: "x y"},
	} {
		got, err := CanonicalizeQuery(test.input)
		if err != nil || got != test.want {
			t.Errorf("CanonicalizeQuery(%q) = %q, %v; want %q", test.input, got, err, test.want)
		}
	}
}

func TestCanonicalizeQueryRejectsEmptyInvalidAndOversized(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"", " \t\n!!! ", string([]byte{0xff}), strings.Repeat("人", MaxQueryRunes+1),
		"</s>", "portrait </S> blue", "<unk>", "blue<UNK>hair",
	} {
		if _, err := CanonicalizeQuery(value); !errors.Is(err, ErrInvalidQuery) {
			t.Errorf("CanonicalizeQuery(%q) error = %v", value, err)
		}
	}
}
