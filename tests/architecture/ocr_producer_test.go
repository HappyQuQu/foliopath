package architecture_test

import (
	"os/exec"
	"testing"
)

func TestOCRProducerDelegatesToFrozenSchedulerOwner(t *testing.T) {
	command := exec.Command("python3", "scripts/check_ocr_ownership.py")
	command.Dir = repositoryRoot(t)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("OCR owner fitness: %v\n%s", err, output)
	}
}
