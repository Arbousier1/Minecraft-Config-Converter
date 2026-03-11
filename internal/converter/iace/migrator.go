package iace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/fileutil"
)

type migrator struct {
	inputPath         string
	outputPath        string
	namespace         string
	armorHumanoidKeys map[string]struct{}
	armorLeggingsKeys map[string]struct{}
}

func newMigrator(inputPath, outputPath, namespace string, humanoidKeys, leggingsKeys map[string]struct{}) *migrator {
	return &migrator{
		inputPath:         inputPath,
		outputPath:        outputPath,
		namespace:         namespace,
		armorHumanoidKeys: humanoidKeys,
		armorLeggingsKeys: leggingsKeys,
	}
}

func (m *migrator) migrate() error {
	if err := m.migrateTextures(); err != nil {
		return err
	}
	if err := m.migrateModels(); err != nil {
		return err
	}
	return nil
}

func (m *migrator) getResourceDir(resourceType string) string {
	candidates := []string{
		filepath.Join(m.inputPath, "assets", m.namespace, resourceType),
		filepath.Join(m.inputPath, m.namespace, resourceType),
		filepath.Join(m.inputPath, resourceType),
	}
	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
	}
	return ""
}

func (m *migrator) migrateTextures() error {
	srcDir := m.getResourceDir("textures")
	if srcDir == "" {
		return nil
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".png" && ext != ".mcmeta" {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		if relPath == "." {
			relPath = ""
		}

		key := normalizeArmorKey(filepath.ToSlash(filepath.Join(relPath, info.Name())))
		var destRel string
		switch {
		case m.hasHumanoidKey(key):
			destRel = m.armorKeyToDestRel(key, false)
		case m.hasLeggingsKey(key):
			destRel = m.armorKeyToDestRel(key, true)
		case m.isArmorIconTexture(info.Name(), relPath):
			destRel = m.buildItemArmorDir(relPath)
		case m.isArmorTexture(info.Name(), relPath):
			destRel = m.buildArmorTextureDir(relPath, info.Name())
		default:
			parts := splitPath(relPath)
			if len(parts) > 0 && parts[0] == "item" {
				destRel = relPath
			} else if relPath == "" {
				destRel = "item"
			} else {
				destRel = filepath.Join("item", relPath)
			}
		}

		destDir := filepath.Join(m.outputPath, "assets", m.namespace, "textures", filepath.FromSlash(filepath.ToSlash(destRel)))
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		return fileutil.CopyFile(path, filepath.Join(destDir, info.Name()))
	})
}

func (m *migrator) migrateModels() error {
	srcDir := m.getResourceDir("models")
	if srcDir == "" {
		return nil
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.ToLower(filepath.Ext(info.Name())) != ".json" {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		if relPath == "." {
			relPath = ""
		}
		destRel := relPath
		parts := splitPath(relPath)
		if len(parts) == 0 {
			destRel = "item"
		} else if parts[0] != "item" {
			destRel = filepath.Join("item", relPath)
		}

		destDir := filepath.Join(m.outputPath, "assets", m.namespace, "models", filepath.FromSlash(filepath.ToSlash(destRel)))
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		return m.processModelFile(path, filepath.Join(destDir, info.Name()))
	})
}

func (m *migrator) processModelFile(srcFile, destFile string) error {
	raw, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	if parent := stringValue(data["parent"]); parent != "" && !strings.HasPrefix(parent, "minecraft:") {
		delete(data, "parent")
	}

	if textures, ok := asStringMap(data["textures"]); ok {
		fixed := map[string]any{}
		for key, value := range textures {
			text := stringValue(value)
			if text == "" {
				continue
			}
			if strings.HasPrefix(text, "#") {
				fixed[key] = text
				continue
			}
			if parts := strings.SplitN(text, ":", 2); len(parts) == 2 {
				if parts[0] == "minecraft" {
					fixed[key] = text
					continue
				}
				fixed[key] = m.namespace + ":" + m.normalizeTexturePath(parts[1])
			} else {
				fixed[key] = m.namespace + ":" + m.normalizeTexturePath(text)
			}
		}
		data["textures"] = fixed
	}

	if overrides, ok := data["overrides"].([]any); ok {
		for _, rawOverride := range overrides {
			override, ok := asStringMap(rawOverride)
			if !ok {
				continue
			}
			modelValue := stringValue(override["model"])
			if modelValue == "" {
				continue
			}
			if parts := strings.SplitN(modelValue, ":", 2); len(parts) == 2 && parts[0] != "minecraft" {
				pathPart := parts[1]
				if !strings.HasPrefix(pathPart, "item/") {
					pathPart = "item/" + pathPart
				}
				override["model"] = m.namespace + ":" + pathPart
			}
		}
	}

	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(destFile, output, 0o644)
}

func (m *migrator) normalizeTexturePath(pathPart string) string {
	path := normalizePath(pathPart)
	path = strings.TrimPrefix(path, "textures/")
	key := normalizeArmorKey(path)
	switch {
	case m.hasHumanoidKey(key):
		return m.armorKeyToDestRel(key, false)
	case m.hasLeggingsKey(key):
		return m.armorKeyToDestRel(key, true)
	case m.isArmorIconTexture(filepath.Base(path), filepath.Dir(path)):
		if path == filepath.Base(path) {
			return "item/armor/" + strings.TrimSuffix(path, filepath.Ext(path))
		}
		return filepath.ToSlash(filepath.Join("item", "armor", filepath.Dir(path), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))))
	case m.isArmorTexture(filepath.Base(path), filepath.Dir(path)):
		dir := m.buildArmorTextureDir(filepath.Dir(path), filepath.Base(path))
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return filepath.ToSlash(filepath.Join(dir, base))
	case strings.HasPrefix(path, "item/"):
		return path
	default:
		return "item/" + path
	}
}

func (m *migrator) hasHumanoidKey(key string) bool {
	_, ok := m.armorHumanoidKeys[key]
	return ok
}

func (m *migrator) hasLeggingsKey(key string) bool {
	_, ok := m.armorLeggingsKeys[key]
	return ok
}

func (m *migrator) isLeggingsTexture(name, relPath string) bool {
	name = strings.ToLower(name)
	relPath = strings.ToLower(filepath.ToSlash(relPath))
	return strings.Contains(name, "layer_2") ||
		strings.Contains(name, "layer2") ||
		strings.Contains(name, "legging") ||
		strings.Contains(relPath, "layer_2") ||
		strings.Contains(relPath, "layer2") ||
		strings.Contains(relPath, "legging")
}

func (m *migrator) armorKeyToDestRel(key string, leggings bool) string {
	path := filepath.ToSlash(strings.TrimPrefix(key, "/"))
	if strings.HasPrefix(path, "entity/equipment/humanoid/") ||
		strings.HasPrefix(path, "entity/equipment/humanoid_legging/") ||
		strings.HasPrefix(path, "entity/equipment/humanoid_leggings/") {
		return path
	}

	parts := splitFiltered(path, map[string]struct{}{
		"textures":          {},
		"entity":            {},
		"equipment":         {},
		"humanoid":          {},
		"humanoid_legging":  {},
		"humanoid_leggings": {},
		"armor":             {},
		"armour":            {},
	})
	target := "humanoid"
	if leggings {
		target = "humanoid_legging"
	}
	if len(parts) == 0 {
		return filepath.ToSlash(filepath.Join("entity", "equipment", target))
	}
	return filepath.ToSlash(filepath.Join("entity", "equipment", target))
}

func (m *migrator) isArmorIconTexture(name, relPath string) bool {
	rel := filepath.ToSlash(strings.ToLower(relPath))
	if rel == "item" || strings.HasPrefix(rel, "item/") {
		return false
	}
	key := normalizeArmorKey(filepath.ToSlash(filepath.Join(relPath, name)))
	if m.hasHumanoidKey(key) || m.hasLeggingsKey(key) {
		return false
	}
	base := strings.TrimSuffix(strings.ToLower(name), filepath.Ext(name))
	if base == "helmet" || base == "chestplate" || base == "leggings" || base == "boots" {
		return true
	}
	if strings.HasSuffix(base, "_helmet") || strings.HasSuffix(base, "_chestplate") || strings.HasSuffix(base, "_leggings") || strings.HasSuffix(base, "_boots") {
		return true
	}
	return strings.Contains(base, "icon") || strings.Contains(rel, "icon")
}

func (m *migrator) isArmorTexture(name, relPath string) bool {
	rel := filepath.ToSlash(strings.ToLower(relPath))
	if rel == "item" || strings.HasPrefix(rel, "item/") {
		return false
	}
	if m.isArmorIconTexture(name, relPath) {
		return false
	}
	name = strings.ToLower(name)
	return strings.Contains(name, "layer_1") ||
		strings.Contains(name, "layer_2") ||
		strings.Contains(name, "armor") ||
		strings.Contains(name, "armour") ||
		strings.Contains(rel, "armor") ||
		strings.Contains(rel, "armour") ||
		strings.Contains(rel, "equipment") ||
		strings.Contains(rel, "humanoid")
}

func (m *migrator) buildItemArmorDir(relPath string) string {
	parts := splitFiltered(relPath, map[string]struct{}{
		"textures": {},
		"armor":    {},
		"armour":   {},
	})
	if len(parts) == 0 {
		return filepath.ToSlash(filepath.Join("item", "armor"))
	}
	return filepath.ToSlash(filepath.Join(append([]string{"item", "armor"}, parts...)...))
}

func (m *migrator) buildArmorTextureDir(relPath, name string) string {
	parts := splitFiltered(relPath, map[string]struct{}{
		"textures":          {},
		"entity":            {},
		"equipment":         {},
		"humanoid":          {},
		"humanoid_legging":  {},
		"humanoid_leggings": {},
		"armor":             {},
		"armour":            {},
	})
	target := "humanoid"
	if m.isLeggingsTexture(name, relPath) {
		target = "humanoid_legging"
	}
	if len(parts) == 0 {
		return filepath.ToSlash(filepath.Join("entity", "equipment", target))
	}
	return filepath.ToSlash(filepath.Join(append([]string{"entity", "equipment", target}, parts...)...))
}

func splitPath(path string) []string {
	path = filepath.ToSlash(path)
	if path == "" || path == "." {
		return nil
	}
	parts := strings.Split(path, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != "." {
			result = append(result, part)
		}
	}
	return result
}
