package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGoldenManifestIntegrity(t *testing.T) {
	manifestPath := filepath.Join("..", "testdata", "golden", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read golden manifest %s: %v", manifestPath, err)
	}

	var manifest struct {
		Models []string `json:"models"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to parse golden manifest.json: %v", err)
	}

	if len(manifest.Models) == 0 {
		t.Fatal("manifest should define at least one golden model")
	}

	for _, modelName := range manifest.Models {
		modelDir := filepath.Join("..", "testdata", "golden", "models", modelName)
		if _, err := os.Stat(modelDir); os.IsNotExist(err) {
			t.Errorf("model directory %s does not exist", modelDir)
		}
	}
}
