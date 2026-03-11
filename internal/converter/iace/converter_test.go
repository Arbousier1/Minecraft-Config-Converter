package iace

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestConvertItemPreservesPythonDisplayFormatting(t *testing.T) {
	converter := New("demo")

	converter.convertItem("sword", map[string]any{
		"display_name": "&aSword",
		"resource": map[string]any{
			"material": "STONE",
		},
	})

	item := converter.config.Items["demo:sword"]
	data := item["data"].(map[string]any)
	if got, want := data["item-name"], "<!i><white>§aSword"; got != want {
		t.Fatalf("item-name mismatch: got %v want %v", got, want)
	}
}

func TestConvertCategoryFallsBackToConfiguredIcon(t *testing.T) {
	converter := New("demo")
	converter.config.Items["demo:known"] = map[string]any{"material": "STONE"}

	converter.convertCategory("decor", map[string]any{
		"name":  "§6Decor",
		"icon":  "known",
		"items": []any{"missing"},
	})

	category := converter.config.Categories["demo:decor"]
	if got, want := category["icon"], "demo:known"; got != want {
		t.Fatalf("icon mismatch: got %v want %v", got, want)
	}
	if got, want := category["name"], "<!i>Decor"; got != want {
		t.Fatalf("name mismatch: got %v want %v", got, want)
	}
}

func TestHandleGenericModelNormalizesArmorTexturePath(t *testing.T) {
	converter := New("demo")
	ceItem := map[string]any{
		"material": "DIAMOND_HELMET",
	}

	converter.handleGenericModel(ceItem, map[string]any{
		"texture": "armor/test_helmet",
	})

	textures, ok := ceItem["textures"].([]string)
	if !ok {
		t.Fatalf("textures not generated: %#v", ceItem["textures"])
	}
	if got, want := textures, []string{"demo:item/armor/test_helmet"}; !slices.Equal(got, want) {
		t.Fatalf("textures mismatch: got %v want %v", got, want)
	}
}

func TestHasEquipmentSettingReadsTopLevelSettings(t *testing.T) {
	if !hasEquipmentSetting(map[string]any{
		"settings": map[string]any{
			"equipment": map[string]any{"slot": "head"},
		},
	}) {
		t.Fatal("expected equipment setting to be detected")
	}
}

func TestCalculateModelYTranslationCachesResults(t *testing.T) {
	root := t.TempDir()
	modelDir := filepath.Join(root, "assets", "demo", "models")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	modelPath := filepath.Join(modelDir, "chair.json")
	payload := []byte(`{"elements":[{"from":[0,-8,0],"to":[16,8,16]}]}`)
	if err := os.WriteFile(modelPath, payload, 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	converter := New("demo")
	converter.SetResourcePaths(root, "")

	if got, want := converter.calculateModelYTranslation("chair"), 1.5; got != want {
		t.Fatalf("first translation mismatch: got %v want %v", got, want)
	}

	if err := os.Remove(modelPath); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	if got, want := converter.calculateModelYTranslation("chair"), 1.5; got != want {
		t.Fatalf("cached translation mismatch: got %v want %v", got, want)
	}
}

func TestFindModelPathVariantCachesResults(t *testing.T) {
	root := t.TempDir()
	modelDir := filepath.Join(root, "assets", "demo", "models")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	variantPath := filepath.Join(modelDir, "bow_pulling_0.json")
	if err := os.WriteFile(variantPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	converter := New("demo")
	converter.SetResourcePaths(root, "")

	if got, want := converter.findModelPathVariant("bow", []string{"_pulling_0", "_0"}), "bow_pulling_0"; got != want {
		t.Fatalf("first variant mismatch: got %v want %v", got, want)
	}

	if err := os.Remove(variantPath); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	if got, want := converter.findModelPathVariant("bow", []string{"_pulling_0", "_0"}), "bow_pulling_0"; got != want {
		t.Fatalf("cached variant mismatch: got %v want %v", got, want)
	}
}
