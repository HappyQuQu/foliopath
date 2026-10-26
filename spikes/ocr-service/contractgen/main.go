// Deterministic internal OCR contract generation; never application composition.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func main() {
	check := flag.Bool("check", false, "verify generated contract bytes")
	flag.Parse()
	for name, data := range map[string]string{"ocr-input.generated.json": ocr.InputContract, "ocr-output.generated.json": ocr.OutputContractJSON()} {
		path := filepath.Join("services/inference", name)
		if *check {
			old, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(old, []byte(data)) {
				fmt.Fprintln(os.Stderr, "OCR generated contract out of date:", path)
				os.Exit(1)
			}
		} else if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			panic(err)
		}
	}
}
