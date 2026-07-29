package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeUsernameUsesNFKCAndFullCaseFolding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input       string
		wantDisplay string
		wantKey     string
	}{
		{input: "Administrator", wantDisplay: "Administrator", wantKey: "administrator"},
		{input: "Ａdmin", wantDisplay: "Admin", wantKey: "admin"},
		{input: "Straße", wantDisplay: "Straße", wantKey: "strasse"},
		{input: "管理员", wantDisplay: "管理员", wantKey: "管理员"},
	}
	for _, test := range tests {
		display, key, err := NormalizeUsername(test.input)
		if err != nil {
			t.Errorf("NormalizeUsername(%q) error = %v", test.input, err)
			continue
		}
		if display != test.wantDisplay || key != test.wantKey {
			t.Errorf(
				"NormalizeUsername(%q) = (%q, %q), want (%q, %q)",
				test.input,
				display,
				key,
				test.wantDisplay,
				test.wantKey,
			)
		}
	}
}

func TestIdentityValidationRejectsUnsafeOrOutOfContractValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"admin user",
		"admin\nuser",
		string([]byte{0xff}),
		strings.Repeat("a", maxUsernameRunes+1),
	} {
		if _, _, err := NormalizeUsername(value); !errors.Is(err, ErrInvalidUsername) {
			t.Errorf("NormalizeUsername(%q) error = %v, want ErrInvalidUsername", value, err)
		}
	}

	for _, value := range []string{
		"",
		"\n",
		"Administrator\x00",
		string([]byte{0xff}),
		strings.Repeat("a", maxDisplayRunes+1),
	} {
		if _, err := NormalizeDisplayName(value); !errors.Is(err, ErrInvalidDisplayName) {
			t.Errorf("NormalizeDisplayName(%q) error = %v, want ErrInvalidDisplayName", value, err)
		}
	}

	for _, value := range []string{
		"short7",
		"valid length\nbut control",
		string([]byte{0xff}),
		strings.Repeat("p", maxPasswordRunes+1),
	} {
		if err := ValidatePassword(value); !errors.Is(err, ErrInvalidPassword) {
			t.Errorf("ValidatePassword(%q) error = %v, want ErrInvalidPassword", value, err)
		}
	}

	if got, err := NormalizeDisplayName("  管理员  "); err != nil || got != "管理员" {
		t.Fatalf("NormalizeDisplayName() = %q, %v; want trimmed Unicode name", got, err)
	}
	for _, value := range []string{"12345678", "密码测试甲乙丙丁"} {
		if err := ValidatePassword(value); err != nil {
			t.Fatalf("ValidatePassword(%q) error = %v for valid password", value, err)
		}
	}
}
