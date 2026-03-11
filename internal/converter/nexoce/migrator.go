package nexoce

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type migrator struct {
	inputPath         string
	outputPath        string
	namespace         string
	armorHumanoidKeys map[string]struct{}
	armorLeggingsKeys map[string]struct{}
	sourceNamespaces  map[string]struct{}
	rootTexturesDir   string
	rootModelsDir     string
}

func newMigrator(inputPath, outputPath, namespace string, humanoidKeys, leggingsKeys, sourceNamespaces map[string]struct{}) *migrator {
	m := &migrator{
		inputPath:         inputPath,
		outputPath:        outputPath,
		namespace:         namespace,
		armorHumanoidKeys: copySet(humanoidKeys),
		armorLeggingsKeys: copySet(leggingsKeys),
		sourceNamespaces:  copySet(sourceNamespaces),
	}
	m.sourceNamespaces[namespace] = struct{}{}
	m.scanNamespaces()
	return m
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

func (m *migrator) scanNamespaces() {
	assetsPath := filepath.Join(m.inputPath, "assets")
	if stat, err := os.Stat(assetsPath); err == nil && stat.IsDir() {
		entries, _ := os.ReadDir(assetsPath)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name == "minecraft" || name == ".mcassetsroot" || name == "realms" {
				continue
			}
			m.sourceNamespaces[name] = struct{}{}
		}

		scanNamespacesUnder(filepath.Join(assetsPath, "minecraft", "textures"), m.sourceNamespaces)
		scanNamespacesUnder(filepath.Join(assetsPath, "minecraft", "models"), m.sourceNamespaces)
	}

	entries, err := os.ReadDir(m.inputPath)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name == "assets" || name == ".mcassetsroot" || name == "realms" {
				continue
			}

			fullPath := filepath.Join(m.inputPath, name)
			if dirExists(filepath.Join(fullPath, "textures")) || dirExists(filepath.Join(fullPath, "models")) {
				m.sourceNamespaces[name] = struct{}{}
			}
		}
	}

	rootTextures := filepath.Join(m.inputPath, "textures")
	if dirExists(rootTextures) {
		m.rootTexturesDir = rootTextures
	}

	rootModels := filepath.Join(m.inputPath, "models")
	if dirExists(rootModels) {
		m.rootModelsDir = rootModels
	}
}

func (m *migrator) migrateTextures() error {
	for _, ns := range sortedKeys(m.sourceNamespaces) {
		for _, srcDir := range m.textureSourceDirs(ns) {
			err := filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if info.IsDir() {
					return nil
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

				key := m.normalizeKey(filepath.ToSlash(filepath.Join(relPath, info.Name())))
				destRel := ""
				switch {
				case m.hasHumanoidKey(key):
					destRel = filepath.ToSlash(filepath.Join("entity", "equipment", "humanoid", key+ext))
				case m.hasLeggingsKey(key):
					destRel = filepath.ToSlash(filepath.Join("entity", "equipment", "humanoid_legging", key+ext))
				case m.isArmorIconTexture(info.Name(), relPath):
					destRel = m.armorIconDestRel(relPath, info.Name(), true)
				default:
					baseRel := filepath.ToSlash(relPath)
					if baseRel == "" {
						baseRel = "item"
					} else if !strings.HasPrefix(baseRel, "item") && !strings.HasPrefix(baseRel, "items") && !strings.HasPrefix(baseRel, "block") {
						baseRel = filepath.ToSlash(filepath.Join("item", baseRel))
					}
					destRel = filepath.ToSlash(filepath.Join(baseRel, info.Name()))
				}

				if ns != m.namespace {
					destRel = m.injectNamespacePath(destRel, ns)
				}

				destFile := filepath.Join(m.outputPath, "assets", m.namespace, "textures", filepath.FromSlash(destRel))
				if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
					return err
				}
				return copyFile(path, destFile)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *migrator) migrateModels() error {
	for _, ns := range sortedKeys(m.sourceNamespaces) {
		for _, srcDir := range m.modelSourceDirs(ns) {
			err := filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if info.IsDir() {
					return nil
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

				baseRel := filepath.ToSlash(relPath)
				if baseRel == "" {
					baseRel = "item"
				} else if !strings.HasPrefix(baseRel, "item") && !strings.HasPrefix(baseRel, "items") && !strings.HasPrefix(baseRel, "block") {
					baseRel = filepath.ToSlash(filepath.Join("item", baseRel))
				}
				destRel := filepath.ToSlash(filepath.Join(baseRel, info.Name()))
				if ns != m.namespace {
					destRel = m.injectNamespacePath(destRel, ns)
				}

				destFile := filepath.Join(m.outputPath, "assets", m.namespace, "models", filepath.FromSlash(destRel))
				if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
					return err
				}
				return m.processModelFile(path, destFile, ns)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *migrator) processModelFile(srcFile, destFile, sourceNS string) error {
	raw, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	if textures, ok := asMap(data["textures"]); ok {
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
				ns := parts[0]
				if ns == m.namespace || m.hasSourceNamespace(ns) {
					newPath := m.adjustTexturePath(parts[1])
					if ns != m.namespace {
						newPath = m.injectNamespacePath(newPath, ns)
					}
					fixed[key] = m.namespace + ":" + filepath.ToSlash(newPath)
					continue
				}
				fixed[key] = text
				continue
			}

			newPath := m.adjustTexturePath(text)
			if sourceNS != "" && sourceNS != m.namespace {
				newPath = m.injectNamespacePath(newPath, sourceNS)
			}
			fixed[key] = m.namespace + ":" + filepath.ToSlash(newPath)
		}
		data["textures"] = fixed
	}

	if overrides, ok := data["overrides"].([]any); ok {
		for _, rawOverride := range overrides {
			override, ok := asMap(rawOverride)
			if !ok {
				continue
			}
			modelValue := stringValue(override["model"])
			if modelValue == "" {
				continue
			}

			if parts := strings.SplitN(modelValue, ":", 2); len(parts) == 2 {
				ns := parts[0]
				if ns == m.namespace || m.hasSourceNamespace(ns) {
					newPath := m.adjustModelPath(parts[1])
					if ns != m.namespace {
						newPath = m.injectNamespacePath(newPath, ns)
					}
					override["model"] = m.namespace + ":" + filepath.ToSlash(newPath)
				}
				continue
			}

			newPath := m.adjustModelPath(modelValue)
			if sourceNS != "" && sourceNS != m.namespace {
				newPath = m.injectNamespacePath(newPath, sourceNS)
			}
			override["model"] = m.namespace + ":" + filepath.ToSlash(newPath)
		}
	}

	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(destFile, output, 0o644)
}

func (m *migrator) adjustTexturePath(path string) string {
	key := m.normalizeKey(path)
	switch {
	case m.hasHumanoidKey(key):
		return filepath.ToSlash(filepath.Join("entity", "equipment", "humanoid", key))
	case m.hasLeggingsKey(key):
		return filepath.ToSlash(filepath.Join("entity", "equipment", "humanoid_legging", key))
	}

	pathNorm := m.normalizeKey(path)
	name := filepath.Base(pathNorm)
	relPath := filepath.Dir(pathNorm)
	if relPath == "." {
		relPath = ""
	}
	if m.isArmorIconTexture(name, relPath) {
		return m.armorIconDestRel(relPath, name, false)
	}

	pathNorm = normalizePath(path)
	if !strings.HasPrefix(pathNorm, "item/") && !strings.HasPrefix(pathNorm, "block/") {
		return "item/" + strings.TrimPrefix(pathNorm, "/")
	}
	return pathNorm
}

func (m *migrator) adjustModelPath(path string) string {
	pathNorm := normalizePath(path)
	if !strings.HasPrefix(pathNorm, "item/") && !strings.HasPrefix(pathNorm, "block/") {
		return "item/" + strings.TrimPrefix(pathNorm, "/")
	}
	return pathNorm
}

func (m *migrator) normalizeKey(path string) string {
	path = normalizePath(path)
	if parts := strings.SplitN(path, ":", 2); len(parts) == 2 {
		path = parts[1]
	}
	path = strings.TrimPrefix(path, "textures/")
	path = strings.TrimSuffix(path, ".png")
	return path
}

func (m *migrator) hasHumanoidKey(key string) bool {
	_, ok := m.armorHumanoidKeys[key]
	return ok
}

func (m *migrator) hasLeggingsKey(key string) bool {
	_, ok := m.armorLeggingsKeys[key]
	return ok
}

func (m *migrator) hasSourceNamespace(namespace string) bool {
	_, ok := m.sourceNamespaces[namespace]
	return ok
}

func (m *migrator) isArmorIconTexture(name, relPath string) bool {
	rel := strings.ToLower(filepath.ToSlash(relPath))
	if rel == "item" || strings.HasPrefix(rel, "item/") {
		return false
	}

	key := m.normalizeKey(filepath.ToSlash(filepath.Join(relPath, name)))
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

func (m *migrator) armorIconDestRel(relPath, name string, includeExt bool) string {
	parts := splitFiltered(relPath, map[string]struct{}{
		"textures": {},
		"armor":    {},
		"armour":   {},
	})
	base := strings.TrimSuffix(name, filepath.Ext(name))
	segments := append([]string{"item", "armor"}, parts...)
	if base != "" {
		segments = append(segments, base)
	}
	path := filepath.ToSlash(filepath.Join(segments...))
	if includeExt {
		return path + filepath.Ext(name)
	}
	return path
}

func (m *migrator) injectNamespacePath(path, namespace string) string {
	path = filepath.ToSlash(strings.TrimPrefix(path, "/"))
	switch {
	case strings.HasPrefix(path, "item/"):
		rest := strings.TrimPrefix(path, "item/")
		if strings.HasPrefix(rest, namespace+"/"+namespace+"/") {
			return "item/" + strings.TrimPrefix(rest, namespace+"/")
		}
		if strings.HasPrefix(rest, namespace+"/") {
			return "item/" + rest
		}
		return "item/" + namespace + "/" + rest
	case strings.HasPrefix(path, "block/"):
		rest := strings.TrimPrefix(path, "block/")
		if strings.HasPrefix(rest, namespace+"/"+namespace+"/") {
			return "block/" + strings.TrimPrefix(rest, namespace+"/")
		}
		if strings.HasPrefix(rest, namespace+"/") {
			return "block/" + rest
		}
		return "block/" + namespace + "/" + rest
	case strings.HasPrefix(path, namespace+"/"+namespace+"/"):
		return strings.TrimPrefix(path, namespace+"/")
	case strings.HasPrefix(path, namespace+"/"):
		return path
	default:
		return namespace + "/" + path
	}
}

func (m *migrator) textureSourceDirs(namespace string) []string {
	dirs := []string{}
	appendDir := func(path string) {
		if dirExists(path) {
			dirs = append(dirs, path)
		}
	}

	appendDir(filepath.Join(m.inputPath, "assets", namespace, "textures"))
	appendDir(filepath.Join(m.inputPath, "assets", "minecraft", "textures", namespace))
	appendDir(filepath.Join(m.inputPath, namespace, "textures"))
	if namespace == m.namespace && m.rootTexturesDir != "" {
		appendDir(m.rootTexturesDir)
	}
	return uniqueStrings(dirs)
}

func (m *migrator) modelSourceDirs(namespace string) []string {
	dirs := []string{}
	appendDir := func(path string) {
		if dirExists(path) {
			dirs = append(dirs, path)
		}
	}

	appendDir(filepath.Join(m.inputPath, "assets", namespace, "models"))
	appendDir(filepath.Join(m.inputPath, "assets", "minecraft", "models", namespace))
	appendDir(filepath.Join(m.inputPath, namespace, "models"))
	if namespace == m.namespace && m.rootModelsDir != "" {
		appendDir(m.rootModelsDir)
	}
	return uniqueStrings(dirs)
}

func scanNamespacesUnder(root string, dst map[string]struct{}) {
	if !dirExists(root) {
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			dst[entry.Name()] = struct{}{}
		}
	}
}

func dirExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func copySet(src map[string]struct{}) map[string]struct{} {
	dst := map[string]struct{}{}
	for key := range src {
		dst[key] = struct{}{}
	}
	return dst
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func copyFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}

func splitFiltered(value string, excluded map[string]struct{}) []string {
	parts := strings.Split(filepath.ToSlash(value), "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if _, skip := excluded[strings.ToLower(part)]; skip {
			continue
		}
		result = append(result, part)
	}
	return result
}
