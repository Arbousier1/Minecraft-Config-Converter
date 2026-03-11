package analyzer

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestAnalyzeUsesReadableChineseContentTypes(t *testing.T) {
	root := t.TempDir()
	itemsAdderDir := filepath.Join(root, "ItemsAdder")
	if err := os.MkdirAll(filepath.Join(itemsAdderDir, "resourcepack", "textures"), 0o755); err != nil {
		t.Fatalf("mkdir textures failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(itemsAdderDir, "resourcepack", "models"), 0o755); err != nil {
		t.Fatalf("mkdir models failed: %v", err)
	}

	yamlPath := filepath.Join(itemsAdderDir, "items.yml")
	yamlContent := []byte("items:\n  chair:\n    resource:\n      material: STONE\n    behaviours:\n      furniture: {}\n")
	if err := os.WriteFile(yamlPath, yamlContent, 0o644); err != nil {
		t.Fatalf("write yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(itemsAdderDir, "resourcepack", "textures", "chair.png"), []byte{}, 0o644); err != nil {
		t.Fatalf("write texture failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(itemsAdderDir, "resourcepack", "models", "chair.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write model failed: %v", err)
	}

	report, err := New(root).Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	if !slices.Contains(report.Formats, "ItemsAdder") {
		t.Fatalf("expected ItemsAdder format, got %v", report.Formats)
	}
	wantContent := []string{"装备", "装饰", "贴图", "模型"}
	if !slices.Equal(report.ContentTypes, wantContent) {
		t.Fatalf("content types mismatch: got %v want %v", report.ContentTypes, wantContent)
	}
}
