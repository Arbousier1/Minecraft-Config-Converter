package nexoce

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/yamlx"
	"gopkg.in/yaml.v3"
)

var namespacePattern = regexp.MustCompile(`^[0-9a-z_.-]+$`)

type Options struct {
	ExtractDir       string
	SessionUploadDir string
	SessionOutputDir string
	OutputDir        string
	OriginalFilename string
	UserNamespace    string
	TargetFormat     string
}

type Result struct {
	DownloadURL string
	OutputFile  string
}

type configDoc struct {
	path string
	data map[string]any
}

type scanResult struct {
	configs          []configDoc
	resourcepackPath string
}

type ceConfig struct {
	Items      map[string]map[string]any
	Equipments map[string]map[string]any
	Templates  map[string]any
	Categories map[string]map[string]any
	Recipes    map[string]map[string]any
}

type Converter struct {
	namespace          string
	config             ceConfig
	resourcepackPath   string
	outputResourceRoot string
	generatedModels    map[string]map[string]any
	armorHumanoidKeys  map[string]struct{}
	armorLeggingsKeys  map[string]struct{}
	sourceNamespaces   map[string]struct{}
}

func Run(opts Options) (*Result, error) {
	scan, err := loadConfigs(opts.ExtractDir)
	if err != nil {
		return nil, err
	}

	convertedCount := 0
	if opts.UserNamespace != "" {
		if !namespacePattern.MatchString(opts.UserNamespace) {
			return nil, fmt.Errorf("namespace contains invalid characters")
		}

		merged := map[string]any{}
		for _, doc := range scan.configs {
			for key, value := range doc.data {
				merged[key] = value
			}
		}

		if err := convertSingle(scan.resourcepackPath, opts.SessionOutputDir, opts.UserNamespace, merged); err != nil {
			return nil, err
		}
		convertedCount++
	} else {
		for _, doc := range scan.configs {
			namespace := sanitizeNamespace(strings.TrimSuffix(filepath.Base(doc.path), filepath.Ext(doc.path)))
			if namespace == "" {
				namespace = "converted"
			}
			if err := convertSingle(scan.resourcepackPath, opts.SessionOutputDir, namespace, doc.data); err != nil {
				return nil, err
			}
			convertedCount++
		}
	}

	if convertedCount == 0 {
		return nil, fmt.Errorf("unable to find Nexo item definitions")
	}

	outputName := buildArchiveName(opts.OriginalFilename, opts.TargetFormat)
	outputPath := filepath.Join(opts.OutputDir, outputName)
	if err := zipDirectory(filepath.Join(opts.SessionOutputDir, "CraftEngine"), outputPath); err != nil {
		return nil, err
	}

	return &Result{
		DownloadURL: "/api/download/" + outputName,
		OutputFile:  outputPath,
	}, nil
}

func convertSingle(resourcepackPath, sessionOutputDir, namespace string, data map[string]any) error {
	converter := New(namespace)
	converter.SetResourcePaths(
		resourcepackPath,
		filepath.Join(sessionOutputDir, "CraftEngine", "resources", namespace, "resourcepack"),
	)
	converter.Convert(data)

	configDir := filepath.Join(sessionOutputDir, "CraftEngine", "resources", namespace, "configuration", "items", namespace)
	return converter.Save(configDir)
}

func New(namespace string) *Converter {
	return &Converter{
		namespace: namespace,
		config: ceConfig{
			Items:      map[string]map[string]any{},
			Equipments: map[string]map[string]any{},
			Templates:  map[string]any{},
			Categories: map[string]map[string]any{},
			Recipes:    map[string]map[string]any{},
		},
		generatedModels:   map[string]map[string]any{},
		armorHumanoidKeys: map[string]struct{}{},
		armorLeggingsKeys: map[string]struct{}{},
		sourceNamespaces:  map[string]struct{}{namespace: {}},
	}
}

func (c *Converter) SetResourcePaths(resourcepackPath, outputResourceRoot string) {
	c.resourcepackPath = resourcepackPath
	c.outputResourceRoot = outputResourceRoot
}

func (c *Converter) Convert(data map[string]any) {
	var recurse func(map[string]any)
	recurse = func(node map[string]any) {
		for key, value := range node {
			child, ok := asMap(value)
			if !ok {
				continue
			}

			if isNexoItem(child) {
				c.convertItem(key, child)
				continue
			}
			recurse(child)
		}
	}

	recurse(data)

	if len(c.config.Categories) == 0 && len(c.config.Items) > 0 {
		c.generateDefaultCategory()
	}
}

func (c *Converter) Save(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	armorItems := map[string]map[string]any{}
	otherItems := map[string]map[string]any{}

	for key, item := range c.config.Items {
		if isArmorMaterial(stringValue(item["material"])) || hasEquipmentSetting(item) {
			armorItems[key] = item
			continue
		}
		otherItems[key] = item
	}

	if len(c.config.Templates) > 0 || len(otherItems) > 0 {
		payload := map[string]any{}
		if len(c.config.Templates) > 0 {
			payload["templates"] = c.config.Templates
		}
		if len(otherItems) > 0 {
			payload["items"] = otherItems
		}
		if err := writeYAML(filepath.Join(outputDir, "items.yml"), payload); err != nil {
			return err
		}
	}

	if len(armorItems) > 0 || len(c.config.Equipments) > 0 {
		payload := map[string]any{}
		if len(armorItems) > 0 {
			payload["items"] = armorItems
		}
		if len(c.config.Equipments) > 0 {
			payload["equipments"] = c.config.Equipments
		}
		if err := writeYAML(filepath.Join(outputDir, "armor.yml"), payload); err != nil {
			return err
		}
	}

	if len(c.config.Categories) > 0 {
		if err := writeYAML(filepath.Join(outputDir, "categories.yml"), map[string]any{
			"categories": c.config.Categories,
		}); err != nil {
			return err
		}
	}

	if len(c.config.Recipes) > 0 {
		if err := writeYAML(filepath.Join(outputDir, "recipe.yml"), map[string]any{
			"recipes": c.config.Recipes,
		}); err != nil {
			return err
		}
	}

	if c.resourcepackPath != "" && c.outputResourceRoot != "" {
		migrator := newMigrator(c.resourcepackPath, c.outputResourceRoot, c.namespace, c.armorHumanoidKeys, c.armorLeggingsKeys, c.sourceNamespaces)
		if err := migrator.migrate(); err != nil {
			return err
		}
	}

	if c.outputResourceRoot != "" && len(c.generatedModels) > 0 {
		modelsRoot := filepath.Join(c.outputResourceRoot, "assets", c.namespace, "models")
		for relPath, content := range c.generatedModels {
			fullPath := filepath.Join(modelsRoot, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return err
			}
			raw, err := json.MarshalIndent(content, "", "    ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(fullPath, raw, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

func loadConfigs(extractDir string) (*scanResult, error) {
	scanRoot := extractDir
	_ = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if strings.EqualFold(info.Name(), "nexo") {
			scanRoot = path
			return filepath.SkipDir
		}
		return nil
	})

	result := &scanResult{
		configs:          []configDoc{},
		resourcepackPath: "",
	}

	err := filepath.Walk(scanRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			if result.resourcepackPath == "" {
				switch {
				case strings.EqualFold(info.Name(), "pack"):
					result.resourcepackPath = path
				default:
					entries, err := os.ReadDir(path)
					if err == nil {
						for _, entry := range entries {
							if entry.IsDir() && strings.EqualFold(entry.Name(), "assets") {
								result.resourcepackPath = path
								break
							}
						}
					}
				}
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		if strings.EqualFold(info.Name(), "config.yml") || strings.EqualFold(info.Name(), "config.yaml") {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		data, err := yamlx.LoadMap(raw)
		if err != nil || len(data) == 0 {
			return nil
		}

		result.configs = append(result.configs, configDoc{
			path: path,
			data: data,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(result.configs) == 0 {
		return nil, fmt.Errorf("unable to find Nexo item definitions")
	}

	return result, nil
}

func (c *Converter) convertItem(key string, data map[string]any) {
	ceID := c.namespace + ":" + key
	material := stringValue(data["material"])
	if material == "" {
		material = "STONE"
	}

	itemName := stringValue(data["itemname"])
	if itemName == "" {
		itemName = key
	}

	ceItem := map[string]any{
		"material": material,
		"data": map[string]any{
			"item-name": itemName,
		},
	}
	dataSection := ceItem["data"].(map[string]any)

	if lore := normalizeLore(data["lore"]); lore != nil {
		dataSection["lore"] = lore
	}

	if modelID, ok := numericValue(data["model"]); ok {
		dataSection["custom-model-data"] = modelID
	}

	pack, _ := asMap(data["Pack"])
	mechanics, _ := asMap(data["Mechanics"])

	switch {
	case isArmorMaterial(material):
		c.handleArmor(ceItem, data)
	case hasMap(mechanics, "furniture"):
		c.handleFurniture(ceItem, data, ceID)
	case isComplexItem(material):
		c.handleComplexItem(ceItem, key, data, material)
	default:
		c.handleGenericModel(ceItem, pack)
	}

	c.config.Items[ceID] = ceItem
}

func (c *Converter) handleArmor(ceItem map[string]any, data map[string]any) {
	pack, _ := asMap(data["Pack"])
	customArmor, _ := asMap(pack["CustomArmor"])

	slot := armorSlot(stringValue(ceItem["material"]))
	if len(customArmor) > 0 {
		layer1 := stringValue(customArmor["layer1"])
		layer2 := stringValue(customArmor["layer2"])
		if layer1 != "" {
			c.registerEquipmentTexture(layer1, false)
		}
		if layer2 != "" {
			c.registerEquipmentTexture(layer2, true)
		}

		texturePath := stringValue(customArmor["texture"])
		if texturePath == "" {
			texturePath = layer1
		}
		assetID := c.normalizeEquipmentKey(texturePath)
		if assetID == "" {
			assetID = "armor_" + slot
		}

		ceItem["settings"] = map[string]any{
			"equipment": map[string]any{
				"asset-id": c.namespace + ":" + assetID,
				"slot":     slot,
			},
		}
	}

	c.handleGenericModel(ceItem, pack)
}

func (c *Converter) handleFurniture(ceItem map[string]any, data map[string]any, ceID string) {
	mechanics, _ := asMap(data["Mechanics"])
	furniture, _ := asMap(mechanics["furniture"])
	properties, _ := asMap(furniture["properties"])
	hitboxConfig, _ := asMap(furniture["hitbox"])
	limitedPlacing, _ := asMap(furniture["limited_placing"])

	tx, ty, tz := parseTriple(stringValue(properties["translation"]), 0, 0, 0)
	sx, sy, sz := parseTriple(stringValue(properties["scale"]), 1, 1, 1)

	element := map[string]any{
		"item":              ceID,
		"display-transform": "NONE",
		"billboard":         "FIXED",
		"translation":       fmt.Sprintf("%g,%g,%g", tx, ty, tz),
	}
	if sx != 1 || sy != 1 || sz != 1 {
		element["scale"] = fmt.Sprintf("%g,%g,%g", sx, sy, sz)
	}

	hitboxes := []any{}
	if barriers, ok := asSlice(hitboxConfig["barriers"]); ok {
		for _, rawBarrier := range barriers {
			bx, by, bz, ok := parseTripleValue(rawBarrier)
			if !ok {
				continue
			}
			hitboxes = append(hitboxes, map[string]any{
				"position":        fmt.Sprintf("%d,%d,%d", int(bx), int(by), int(bz)),
				"type":            "shulker",
				"blocks-building": true,
				"interactive":     true,
			})
		}
	}

	if seats, ok := asSlice(furniture["seats"]); ok && len(seats) > 0 {
		ceSeats := make([]string, 0, len(seats))
		for _, rawSeat := range seats {
			sx, sy, sz, ok := parseTripleValue(rawSeat)
			if !ok {
				continue
			}
			ceSeats = append(ceSeats, fmt.Sprintf("%g,%g,%g", sx, sy, sz))
		}
		if len(ceSeats) > 0 {
			hitboxes = append(hitboxes, map[string]any{
				"position":        "0,0,0",
				"type":            "interaction",
				"blocks-building": true,
				"width":           1,
				"height":          1,
				"interactive":     true,
				"seats":           ceSeats,
			})
		}
	} else if len(hitboxes) == 0 {
		hitboxes = append(hitboxes, map[string]any{
			"position":        "0,0,0",
			"type":            "interaction",
			"blocks-building": true,
			"interactive":     true,
		})
	}

	placementConfig := map[string]any{
		"loot-spawn-offset": "0,0.4,0",
		"rules": map[string]any{
			"rotation":  "ANY",
			"alignment": "ANY",
		},
		"elements": []any{element},
	}
	if len(hitboxes) > 0 {
		placementConfig["hitboxes"] = hitboxes
	}

	placement := map[string]any{}
	if boolValue(limitedPlacing["floor"]) {
		placement["ground"] = cloneMap(placementConfig)
	}
	if boolValue(limitedPlacing["wall"]) {
		placement["wall"] = cloneMap(placementConfig)
	}
	if boolValue(limitedPlacing["roof"]) {
		placement["ceiling"] = cloneMap(placementConfig)
	}
	if len(placement) == 0 {
		placement["ground"] = placementConfig
	}

	ceItem["behavior"] = map[string]any{
		"type": "furniture_item",
		"furniture": map[string]any{
			"settings": map[string]any{
				"item": ceID,
				"sounds": map[string]any{
					"break": "minecraft:block.stone.break",
					"place": "minecraft:block.stone.place",
				},
			},
			"loot": map[string]any{
				"template": "default:loot_table/furniture",
				"arguments": map[string]any{
					"item": ceID,
				},
			},
			"placement": placement,
		},
	}

	pack, _ := asMap(data["Pack"])
	c.handleGenericModel(ceItem, pack)
}

func (c *Converter) handleComplexItem(ceItem map[string]any, key string, data map[string]any, material string) {
	pack, _ := asMap(data["Pack"])
	templateID := fmt.Sprintf("models:%s_%s_model", c.namespace, key)
	args := map[string]any{}

	baseModel := stringValue(pack["model"])
	baseRef := c.getModelRef(baseModel)
	switch material {
	case "BOW":
		args["bow_model"] = firstNonEmpty(baseRef)
		models := sliceStrings(pack["pulling_models"])
		args["bow_pulling_0_model"] = fallbackRef(c.getModelRef(indexOr(models, 0)), baseRef)
		args["bow_pulling_1_model"] = fallbackRef(c.getModelRef(indexOr(models, 1)), baseRef)
		args["bow_pulling_2_model"] = fallbackRef(c.getModelRef(indexOr(models, 2)), baseRef)
	case "CROSSBOW":
		args["model"] = firstNonEmpty(baseRef)
		models := sliceStrings(pack["pulling_models"])
		args["pulling_0_model"] = fallbackRef(c.getModelRef(indexOr(models, 0)), baseRef)
		args["pulling_1_model"] = fallbackRef(c.getModelRef(indexOr(models, 1)), baseRef)
		args["pulling_2_model"] = fallbackRef(c.getModelRef(indexOr(models, 2)), baseRef)
		args["arrow_model"] = fallbackRef(c.getModelRef(stringValue(pack["charged_model"])), baseRef)
		args["firework_model"] = fallbackRef(c.getModelRef(stringValue(pack["firework_model"])), baseRef)
	case "SHIELD":
		args["shield_model"] = firstNonEmpty(baseRef)
		args["shield_blocking_model"] = fallbackRef(c.getModelRef(stringValue(pack["blocking_model"])), baseRef)
	case "FISHING_ROD":
		args["path"] = firstNonEmpty(baseRef)
		args["cast_path"] = fallbackRef(c.getModelRef(stringValue(pack["cast_model"])), baseRef)
	}

	c.generateTemplateDefinition(templateID, material)
	ceItem["model"] = map[string]any{
		"template":  templateID,
		"arguments": args,
	}
}

func (c *Converter) generateTemplateDefinition(templateID, material string) {
	template := map[string]any{}
	switch material {
	case "BOW":
		template = map[string]any{
			"type":     "minecraft:condition",
			"property": "minecraft:using_item",
			"on-false": map[string]any{"type": "minecraft:model", "path": "${bow_model}"},
			"on-true": map[string]any{
				"type":     "minecraft:range_dispatch",
				"property": "minecraft:use_duration",
				"scale":    0.05,
				"entries": []any{
					map[string]any{"threshold": 0.65, "model": map[string]any{"type": "minecraft:model", "path": "${bow_pulling_1_model}"}},
					map[string]any{"threshold": 0.9, "model": map[string]any{"type": "minecraft:model", "path": "${bow_pulling_2_model}"}},
				},
				"fallback": map[string]any{"type": "minecraft:model", "path": "${bow_pulling_0_model}"},
			},
		}
	case "CROSSBOW":
		template = map[string]any{
			"type":     "minecraft:condition",
			"property": "minecraft:using_item",
			"on-false": map[string]any{
				"type":     "minecraft:select",
				"property": "minecraft:charge_type",
				"cases": []any{
					map[string]any{"when": "arrow", "model": map[string]any{"type": "minecraft:model", "path": "${arrow_model}"}},
					map[string]any{"when": "rocket", "model": map[string]any{"type": "minecraft:model", "path": "${firework_model}"}},
				},
				"fallback": map[string]any{"type": "minecraft:model", "path": "${model}"},
			},
			"on-true": map[string]any{
				"type":     "minecraft:range_dispatch",
				"property": "minecraft:crossbow/pull",
				"entries": []any{
					map[string]any{"threshold": 0.58, "model": map[string]any{"type": "minecraft:model", "path": "${pulling_1_model}"}},
					map[string]any{"threshold": 1.0, "model": map[string]any{"type": "minecraft:model", "path": "${pulling_2_model}"}},
				},
				"fallback": map[string]any{"type": "minecraft:model", "path": "${pulling_0_model}"},
			},
		}
	case "SHIELD":
		template = map[string]any{
			"type":     "minecraft:condition",
			"property": "minecraft:using_item",
			"on-false": map[string]any{"type": "minecraft:model", "path": "${shield_model}"},
			"on-true":  map[string]any{"type": "minecraft:model", "path": "${shield_blocking_model}"},
		}
	case "FISHING_ROD":
		template = map[string]any{
			"type":     "minecraft:condition",
			"property": "minecraft:fishing_rod/cast",
			"on-false": map[string]any{"type": "minecraft:model", "path": "${path}"},
			"on-true":  map[string]any{"type": "minecraft:model", "path": "${cast_path}"},
		}
	}
	if len(template) > 0 {
		c.config.Templates[templateID] = template
	}
}

func (c *Converter) handleGenericModel(ceItem map[string]any, pack map[string]any) {
	modelPath := stringValue(pack["model"])
	if modelPath == "" {
		return
	}
	ceItem["model"] = map[string]any{
		"type": "minecraft:model",
		"path": c.getModelRef(modelPath),
	}
}

func (c *Converter) getModelRef(path string) string {
	if path == "" {
		return ""
	}

	p := normalizePath(path)
	ns := c.namespace
	if parts := strings.SplitN(p, ":", 2); len(parts) == 2 {
		ns = parts[0]
		p = parts[1]
		c.sourceNamespaces[ns] = struct{}{}
		if ns != c.namespace {
			p = filepath.ToSlash(filepath.Join(ns, p))
		}
	} else if strings.HasPrefix(p, c.namespace+"/") {
		p = strings.TrimPrefix(p, c.namespace+"/")
	} else {
		parts := splitPath(p)
		if len(parts) > 1 {
			ns = parts[0]
			c.sourceNamespaces[ns] = struct{}{}
		}
	}

	if !strings.HasPrefix(p, "item/") && !strings.HasPrefix(p, "block/") {
		p = "item/" + strings.TrimPrefix(p, "/")
	}
	return c.namespace + ":" + filepath.ToSlash(p)
}

func (c *Converter) generateDefaultCategory() {
	itemIDs := make([]string, 0, len(c.config.Items))
	for key := range c.config.Items {
		itemIDs = append(itemIDs, key)
	}
	sort.Strings(itemIDs)
	if len(itemIDs) == 0 {
		return
	}

	c.config.Categories[c.namespace+":default"] = map[string]any{
		"name":     "<!i>" + strings.Title(c.namespace),
		"priority": 1,
		"icon":     itemIDs[0],
		"list":     itemIDs,
		"hidden":   false,
	}
}

func (c *Converter) registerEquipmentTexture(raw string, leggings bool) {
	key := c.normalizeEquipmentKey(raw)
	if key == "" {
		return
	}
	if leggings {
		c.armorLeggingsKeys[key] = struct{}{}
		return
	}
	c.armorHumanoidKeys[key] = struct{}{}
}

func (c *Converter) normalizeEquipmentKey(raw string) string {
	value := normalizePath(raw)
	if parts := strings.SplitN(value, ":", 2); len(parts) == 2 {
		value = parts[1]
	}
	value = strings.TrimSuffix(value, ".png")
	value = strings.TrimPrefix(value, "textures/")
	return value
}

func writeYAML(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	raw, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	return os.WriteFile(path, raw, 0o644)
}

func zipDirectory(sourceDir, destination string) error {
	dstFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	zipWriter := zip.NewWriter(dstFile)
	defer zipWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = writer.Write(content)
		return err
	})
}

func buildArchiveName(original, target string) string {
	base := strings.TrimSuffix(original, filepath.Ext(original))
	if base == "" {
		base = "converted"
	}
	name := fmt.Sprintf("%s [%s by MCC].zip", base, target)
	replacer := strings.NewReplacer("\\", "", "/", "", ":", "", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
	return replacer.Replace(name)
}

func sanitizeNamespace(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '.' || char == '_' || char == '-':
			builder.WriteRune(char)
		default:
			builder.WriteByte('_')
		}
	}
	return builder.String()
}

func isNexoItem(data map[string]any) bool {
	_, hasMaterial := data["material"]
	_, hasItemName := data["itemname"]
	return hasMaterial || hasItemName
}

func hasEquipmentSetting(item map[string]any) bool {
	settings, ok := asMap(item["settings"])
	if !ok {
		return false
	}
	_, exists := settings["equipment"]
	return exists
}

func isArmorMaterial(material string) bool {
	for _, suffix := range []string{"_HELMET", "_CHESTPLATE", "_LEGGINGS", "_BOOTS"} {
		if strings.HasSuffix(material, suffix) {
			return true
		}
	}
	return false
}

func armorSlot(material string) string {
	switch {
	case strings.HasSuffix(material, "_CHESTPLATE"):
		return "chest"
	case strings.HasSuffix(material, "_LEGGINGS"):
		return "legs"
	case strings.HasSuffix(material, "_BOOTS"):
		return "feet"
	default:
		return "head"
	}
}

func isComplexItem(material string) bool {
	switch material {
	case "BOW", "CROSSBOW", "FISHING_ROD", "SHIELD":
		return true
	default:
		return false
	}
}

func normalizeLore(raw any) []string {
	switch value := raw.(type) {
	case nil:
		return nil
	case string:
		return []string{value}
	case []any:
		result := make([]string, 0, len(value))
		for _, line := range value {
			result = append(result, fmt.Sprint(line))
		}
		return result
	default:
		return nil
	}
}

func parseTriple(raw string, fallbackX, fallbackY, fallbackZ float64) (float64, float64, float64) {
	if raw == "" {
		return fallbackX, fallbackY, fallbackZ
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 3 {
		return fallbackX, fallbackY, fallbackZ
	}
	x, errX := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	y, errY := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	z, errZ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if errX != nil || errY != nil || errZ != nil {
		return fallbackX, fallbackY, fallbackZ
	}
	return x, y, z
}

func parseTripleValue(raw any) (float64, float64, float64, bool) {
	if stringValue(raw) == "" {
		return 0, 0, 0, false
	}
	x, y, z := parseTriple(stringValue(raw), 0, 0, 0)
	return x, y, z, true
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func fallbackRef(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func indexOr(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

func sliceStrings(raw any) []string {
	values, ok := asSlice(raw)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text := stringValue(value)
		if text != "" {
			result = append(result, text)
		}
	}
	return result
}

func asMap(raw any) (map[string]any, bool) {
	value, ok := raw.(map[string]any)
	return value, ok
}

func asSlice(raw any) ([]any, bool) {
	value, ok := raw.([]any)
	return value, ok
}

func hasMap(raw map[string]any, key string) bool {
	value, ok := asMap(raw[key])
	return ok && len(value) > 0
}

func stringValue(raw any) string {
	switch value := raw.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

func numericValue(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case float32:
		return int(value), true
	case string:
		number, err := strconv.Atoi(strings.TrimSpace(value))
		return number, err == nil
	default:
		return 0, false
	}
}

func boolValue(raw any) bool {
	value, ok := raw.(bool)
	return ok && value
}

func normalizePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimPrefix(value, "/")
	return value
}

func splitPath(path string) []string {
	path = normalizePath(path)
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
