package benchmarks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/analyzer"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/converter/iace"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/converter/nexoce"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/packageindex"
)

func BenchmarkBuildPackageIndexItemsAdder(b *testing.B) {
	root := b.TempDir()
	extractDir := createItemsAdderFixture(b, root, 120)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := packageindex.Build(extractDir); err != nil {
			b.Fatalf("build package index: %v", err)
		}
	}
}

func BenchmarkAnalyzeFromPackageIndexItemsAdder(b *testing.B) {
	root := b.TempDir()
	extractDir := createItemsAdderFixture(b, root, 120)
	index, err := packageindex.Build(extractDir)
	if err != nil {
		b.Fatalf("build package index: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := analyzer.NewFromIndex(index).Analyze(); err != nil {
			b.Fatalf("analyze: %v", err)
		}
	}
}

func BenchmarkIACERunFromPackageIndex(b *testing.B) {
	root := b.TempDir()
	extractDir := createItemsAdderFixture(b, root, 80)
	index, err := packageindex.Build(extractDir)
	if err != nil {
		b.Fatalf("build package index: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sessionRoot := filepath.Join(root, "ia-bench", fmt.Sprintf("run-%d", i))
		if err := os.MkdirAll(sessionRoot, 0o755); err != nil {
			b.Fatalf("mkdir session root: %v", err)
		}

		_, err := iace.Run(iace.Options{
			ExtractDir:       extractDir,
			Index:            index,
			SessionUploadDir: filepath.Join(sessionRoot, "upload"),
			SessionOutputDir: filepath.Join(sessionRoot, "output"),
			OutputDir:        filepath.Join(sessionRoot, "archives"),
			OriginalFilename: "itemsadder-bench.zip",
			UserNamespace:    "bench",
			TargetFormat:     "CraftEngine",
		})
		if err != nil {
			b.Fatalf("iace run: %v", err)
		}
	}
}

func BenchmarkNexoRunFromPackageIndex(b *testing.B) {
	root := b.TempDir()
	extractDir := createNexoFixture(b, root, 80)
	index, err := packageindex.Build(extractDir)
	if err != nil {
		b.Fatalf("build package index: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sessionRoot := filepath.Join(root, "nexo-bench", fmt.Sprintf("run-%d", i))
		if err := os.MkdirAll(sessionRoot, 0o755); err != nil {
			b.Fatalf("mkdir session root: %v", err)
		}

		_, err := nexoce.Run(nexoce.Options{
			ExtractDir:       extractDir,
			Index:            index,
			SessionUploadDir: filepath.Join(sessionRoot, "upload"),
			SessionOutputDir: filepath.Join(sessionRoot, "output"),
			OutputDir:        filepath.Join(sessionRoot, "archives"),
			OriginalFilename: "nexo-bench.zip",
			UserNamespace:    "bench",
			TargetFormat:     "CraftEngine",
		})
		if err != nil {
			b.Fatalf("nexo run: %v", err)
		}
	}
}

func createItemsAdderFixture(tb testing.TB, root string, itemCount int) string {
	tb.Helper()

	extractDir := filepath.Join(root, "itemsadder-extract")
	resourcepackDir := filepath.Join(extractDir, "ItemsAdder", "resourcepack")
	textureDir := filepath.Join(resourcepackDir, "textures", "item")
	modelDir := filepath.Join(resourcepackDir, "models", "item")

	mustMkdirAll(tb, textureDir)
	mustMkdirAll(tb, modelDir)

	var itemsBuilder strings.Builder
	itemsBuilder.WriteString("info:\n  namespace: demo\nitems:\n")
	for i := 0; i < itemCount; i++ {
		key := fmt.Sprintf("item_%03d", i)
		itemsBuilder.WriteString(fmt.Sprintf("  %s:\n", key))
		itemsBuilder.WriteString("    display_name: \"&aBench Item\"\n")
		itemsBuilder.WriteString("    resource:\n")
		itemsBuilder.WriteString("      material: STONE\n")
		itemsBuilder.WriteString(fmt.Sprintf("      model_id: %d\n", 1000+i))
		itemsBuilder.WriteString(fmt.Sprintf("      model_path: item/%s\n", key))
		itemsBuilder.WriteString(fmt.Sprintf("      texture: item/%s\n", key))
		if i%5 == 0 {
			itemsBuilder.WriteString("    behaviours:\n")
			itemsBuilder.WriteString("      furniture: {}\n")
		}

		mustWriteFile(tb, filepath.Join(textureDir, key+".png"), []byte("png"))
		mustWriteFile(tb, filepath.Join(modelDir, key+".json"), []byte(`{"textures":{"layer0":"demo:item/`+key+`"}}`))
	}

	itemsBuilder.WriteString("categories:\n  default:\n    enabled: true\n    icon: item_000\n    items:\n")
	for i := 0; i < min(itemCount, 10); i++ {
		itemsBuilder.WriteString(fmt.Sprintf("      - item_%03d\n", i))
	}

	mustWriteFile(tb, filepath.Join(extractDir, "ItemsAdder", "items.yml"), []byte(itemsBuilder.String()))
	return extractDir
}

func createNexoFixture(tb testing.TB, root string, itemCount int) string {
	tb.Helper()

	extractDir := filepath.Join(root, "nexo-extract")
	configDir := filepath.Join(extractDir, "Nexo")
	resourcepackDir := filepath.Join(configDir, "pack", "assets", "bench")
	textureDir := filepath.Join(resourcepackDir, "textures", "item")
	modelDir := filepath.Join(resourcepackDir, "models", "item")

	mustMkdirAll(tb, textureDir)
	mustMkdirAll(tb, modelDir)

	var configBuilder strings.Builder
	for i := 0; i < itemCount; i++ {
		key := fmt.Sprintf("item_%03d", i)
		configBuilder.WriteString(fmt.Sprintf("%s:\n", key))
		configBuilder.WriteString("  material: STONE\n")
		configBuilder.WriteString("  itemname: \"Bench Item\"\n")
		configBuilder.WriteString(fmt.Sprintf("  model: %d\n", 2000+i))
		configBuilder.WriteString("  Pack:\n")
		configBuilder.WriteString(fmt.Sprintf("    model: item/%s\n", key))
		if i%4 == 0 {
			configBuilder.WriteString("  Mechanics:\n")
			configBuilder.WriteString("    furniture: {}\n")
		}

		mustWriteFile(tb, filepath.Join(textureDir, key+".png"), []byte("png"))
		mustWriteFile(tb, filepath.Join(modelDir, key+".json"), []byte(`{"textures":{"layer0":"bench:item/`+key+`"}}`))
	}

	mustWriteFile(tb, filepath.Join(configDir, "bench.yml"), []byte(configBuilder.String()))
	return extractDir
}

func mustMkdirAll(tb testing.TB, path string) {
	tb.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		tb.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(tb testing.TB, path string, payload []byte) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		tb.Fatalf("write %s: %v", path, err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
