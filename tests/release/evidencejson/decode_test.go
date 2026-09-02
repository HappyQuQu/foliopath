package evidencejson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testEvidence struct {
	Result string `json:"result"`
}

func TestDecodeRejectsAmbiguousEvidenceJSON(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "unknown", input: `{"result":"passed","extra":true}`, want: "unknown field"},
		{name: "duplicate", input: `{"result":"passed","result":"failed"}`, want: "duplicate JSON key"},
		{name: "trailing", input: `{"result":"passed"} {}`, want: "trailing JSON value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var value testEvidence
			if err := Decode([]byte(test.input), &value); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v want=%q", err, test.want)
			}
		})
	}
}

func TestDecodeAcceptsSingleStrictObject(t *testing.T) {
	var value testEvidence
	if err := Decode([]byte(`{"result":"passed"}`), &value); err != nil || value.Result != "passed" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
}

func TestReadRegularFileRejectsSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.json")
	if err := os.WriteFile(target, []byte(`{"result":"passed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "evidence.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegularFile(link); err == nil || !strings.Contains(err.Error(), "non-symlink regular file") {
		t.Fatalf("error=%v", err)
	}
}
