package app

import (
	"testing"
)

func TestManagedProfileUsesValidatedDeploymentContract(t *testing.T) {
	config, err := deploymentFromEnvironment([]string{"FOLIOPATH_AI_PROFILE=standard-v1"})
	if err != nil {
		t.Fatal(err)
	}
	if config == nil || config.Identity.Dimension != 640 || config.Face == nil || config.Tag == nil || config.OCR == nil {
		t.Fatalf("incomplete managed deployment: %+v", config)
	}
	if config.Origin != "http://machine-learning:3003" {
		t.Fatal(config.Origin)
	}
	if _, err := loadConfiguration(Input{Environ: []string{"FOLIOPATH_AI_PROFILE=standard-v1"}}); err != nil {
		t.Fatal(err)
	}
}

func TestManagedProfileRejectsAmbiguousOrUnknownConfiguration(t *testing.T) {
	for _, env := range [][]string{
		{"FOLIOPATH_AI_PROFILE="},
		{"FOLIOPATH_AI_PROFILE=other"},
		{"FOLIOPATH_AI_PROFILE"},
		{"FOLIOPATH_AI_PROFILE=standard-v1", "FOLIOPATH_AI_PROFILE=standard-v1"},
		{"FOLIOPATH_AI_PROFILE=standard-v1", "FOLIOPATH_AI_DEPLOYMENT={}"},
	} {
		if _, err := deploymentFromEnvironment(env); err == nil {
			t.Fatalf("accepted %q", env)
		}
	}
	if config, err := deploymentFromEnvironment(nil); err != nil || config != nil {
		t.Fatal(config, err)
	}
}
