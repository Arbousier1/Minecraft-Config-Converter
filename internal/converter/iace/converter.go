package iace

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/yamlx"
	"gopkg.in/yaml.v3"
)

var namespacePattern = regexp.MustCompile(`^[0-9a-z_.-]+$`)

type Options struct {
	ExtractDir        string
	SessionUploadDir  string
	SessionOutputDir  string
	OutputDir         string
	OriginalFilename  string
	UserNamespace     string
	TargetFormat      string
}

type Result struct {
	DownloadURL string
	OutputFile  string
}

type Converter struct {
	namespace string
	config    ceConfig
}

type ceConfig struct {
	Items      map[string]map[string]any
	Equipments map[string]map[string]any
	Templates  map[string]any
	Categories map[string]map[string]any
	Recipes    map[string]map[string]any
}

type mergedData struct {
	Items           map[string]map[string]any
	Equipments      map[string]map[string]any
	ArmorsRendering map[string]map[string]any
	Templates       map[string]any
	Categories      map[string]map[string]any
	Recipes         map[string]map[string]map[string]any
	Info            map[string]any
}

func Run(opts Options) (*Result, error) {
	data, err := loadMergedData(opts.ExtractDir)
	if err != nil {
		return nil, err
	}

	namespace := "converted"
	if rawNS, ok := data.Info["namespace"].(string); ok && rawNS != "" {
		namespace = rawNS
	}
	if opts.UserNamespace != "" {
		if !namespacePattern.MatchString(opts.UserNamespace) {
			return nil, fmt.Errorf("namespace contains invalid characters")
		}
		namespace = opts.UserNamespace
	}

	converter := New(namespace)
	converter.Convert(data)

	configDir := filepath.Join(opts.SessionOutputDir, "CraftEngine", "resources", namespace, "configuration", "items", namespace)
	if err := converter.Save(configDir); err != nil {
		return nil, err
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
	}
}

func (c *Converter) Convert(data *mergedData) {
	for key, value := range data.Templates {
		c.config.Templates[key] = value
	}

	for key, item := range data.Items {
		c.convertItem(key, item)
	}

	for key, equipment := range data.Equipments {
		c.convertEquipment(key, equipment)
	}

	for key, armor := range data.ArmorsRendering {
		c.convertArmorRendering(key, armor)
	}

	for key, category := range data.Categories {
		c.convertCategory(key, category)
	}

	for groupKey, recipes := range data.Recipes {
		c.convertRecipeGroup(groupKey, recipes)
	}

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

	return nil
}

func loadMergedData(extractDir string) (*mergedData, error) {
	scanRoot := extractDir
	_ = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if strings.EqualFold(info.Name(), "itemsadder") {
			scanRoot = path
			return filepath.SkipDir
		}
		return nil
	})

	merged := &mergedData{
		Items:           map[string]map[string]any{},
		Equipments:      map[string]map[string]any{},
		ArmorsRendering: map[string]map[string]any{},
		Templates:       map[string]any{},
		Categories:      map[string]map[string]any{},
		Recipes:         map[string]map[string]map[string]any{},
		Info:            map[string]any{},
	}

	foundItems := false

	err := filepath.Walk(scanRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".yml" && ext != ".yaml" {
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

		if infoMap, ok := asStringMap(data["info"]); ok && len(merged.Info) == 0 {
			for key, value := range infoMap {
				merged.Info[key] = value
			}
		}

		if items, ok := asNestedMap(data["items"]); ok {
			foundItems = true
			mergeNested(merged.Items, items)
		}
		if equipments, ok := asNestedMap(data["equipments"]); ok {
			foundItems = true
			mergeNested(merged.Equipments, equipments)
		}
		if armors, ok := asNestedMap(data["armors_rendering"]); ok {
			foundItems = true
			mergeNested(merged.ArmorsRendering, armors)
		}
		if templates, ok := asStringMap(data["templates"]); ok {
			for key, value := range templates {
				merged.Templates[key] = value
			}
		}
		if categories, ok := asNestedMap(data["categories"]); ok {
			mergeNested(merged.Categories, categories)
		}
		if recipes, ok := asRecipeGroups(data["recipes"]); ok {
			mergeRecipeGroups(merged.Recipes, recipes)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if !foundItems {
		return nil, fmt.Errorf("unable to find ItemsAdder item definitions")
	}

	return merged, nil
}

func (c *Converter) convertItem(key string, data map[string]any) {
	ceID := c.namespaced(key)
	resource, _ := asStringMap(data["resource"])
	material := "STONE"
	if value, ok := resource["material"].(string); ok && value != "" {
		material = value
	}
	displayName := key
	if value, ok := data["display_name"].(string); ok && value != "" {
		displayName = value
	}

	ceItem := map[string]any{
		"material": material,
		"data": map[string]any{
			"item-name": formatDisplayName(displayName),
		},
	}
	dataSection := ceItem["data"].(map[string]any)

	if lore := normalizeLore(data["lore"]); lore != nil {
		dataSection["lore"] = lore
	}

	if modelID, ok := numericValue(resource["model_id"]); ok {
		dataSection["custom-model-data"] = modelID
	}

	behaviours, _ := asStringMap(data["behaviours"])
	if boolValue(behaviours["hat"]) {
		dataSection["equippable"] = map[string]any{"slot": "head"}
	}

	if modelPath, ok := resource["model_path"].(string); ok && modelPath != "" {
		ceItem["model"] = map[string]any{
			"type": "minecraft:model",
			"path": c.modelRef(modelPath),
		}
	} else if textures := normalizeTextures(resource); len(textures) > 0 {
		namespaced := make([]string, 0, len(textures))
		for _, texture := range textures {
			namespaced = append(namespaced, c.namespace+":"+texture)
		}
		ceItem["textures"] = namespaced
	}

	if equipment, ok := asStringMap(data["equipment"]); ok && len(equipment) > 0 {
		id := stringValue(equipment["id"])
		if id == "" {
			id = key
		}
		dataSection["equipment"] = c.namespaced(id)
	}

	c.config.Items[ceID] = ceItem
}

func (c *Converter) convertEquipment(key string, data map[string]any) {
	entry := map[string]any{
		"type": "component",
	}
	if layer1 := stringValue(data["layer_1"]); layer1 != "" {
		entry["humanoid"] = c.normalizeEquipmentTexturePath(layer1, false)
	}
	if layer2 := stringValue(data["layer_2"]); layer2 != "" {
		entry["humanoid-leggings"] = c.normalizeEquipmentTexturePath(layer2, true)
	}
	c.config.Equipments[c.namespaced(key)] = entry
}

func (c *Converter) convertArmorRendering(key string, data map[string]any) {
	c.convertEquipment(key, data)
}

func (c *Converter) convertCategory(key string, data map[string]any) {
	ceItems := []string{}
	if items, ok := data["items"].([]any); ok {
		for _, rawItem := range items {
			item := stringValue(rawItem)
			if item == "" {
				continue
			}
			ceItems = append(ceItems, c.namespacedPath(item))
		}
	}

	icon := "minecraft:stone"
	if len(ceItems) > 0 {
		icon = ceItems[0]
	} else if rawIcon := stringValue(data["icon"]); rawIcon != "" {
		icon = c.namespacedPath(rawIcon)
	}

	name := stripMinecraftColorCodes(stringValue(data["name"]))
	if name == "" {
		name = key
	}

	entry := map[string]any{
		"name":     "<!i>" + name,
		"priority": 1,
		"icon":     icon,
		"list":     ceItems,
		"hidden":   !boolDefaultTrue(data["enabled"]),
	}
	c.config.Categories[c.namespaced(key)] = entry
}

func (c *Converter) convertRecipeGroup(groupKey string, recipes map[string]map[string]any) {
	for recipeKey, recipeData := range recipes {
		if boolValue(recipeData["enabled"]) == false && recipeData["enabled"] != nil {
			continue
		}

		recipeType := mapRecipeType(groupKey, recipeData)
		if recipeType == "" {
			continue
		}

		entry := map[string]any{"type": recipeType}
		switch recipeType {
		case "shaped":
			if pattern, ok := asStringSlice(recipeData["pattern"]); ok && len(pattern) > 0 {
				entry["pattern"] = normalizePattern(pattern, recipeData["ingredients"])
			}
			if ingredients, ok := asStringMap(recipeData["ingredients"]); ok && len(ingredients) > 0 {
				entry["ingredients"] = normalizeIngredientMap(ingredients)
			}
		case "shapeless":
			entry["ingredients"] = normalizeShapelessIngredients(recipeData["ingredients"])
		case "smelting", "blasting", "smoking", "campfire_cooking":
			if ingredient := firstNonNil(recipeData["ingredient"], recipeData["ingredients"]); ingredient != nil {
				entry["ingredient"] = normalizeRecipeItem(flattenSingleList(ingredient))
			}
			if exp, ok := numericValue(recipeData["experience"]); ok {
				entry["experience"] = exp
			}
			if timeValue, ok := numericValue(firstNonNil(recipeData["time"], recipeData["cookingTime"])); ok {
				entry["time"] = timeValue
			}
			if category := stringValue(recipeData["category"]); category != "" {
				entry["category"] = category
			}
			if group := stringValue(recipeData["group"]); group != "" {
				entry["group"] = group
			}
		case "stonecutting":
			if ingredient := normalizeRecipeItem(recipeData["ingredient"]); ingredient != nil {
				entry["ingredient"] = ingredient
			}
			if group := stringValue(recipeData["group"]); group != "" {
				entry["group"] = group
			}
		case "smithing_transform":
			if template := normalizeRecipeItem(firstNonNil(recipeData["template"], recipeData["template-type"])); template != nil {
				entry["template-type"] = template
			}
			if base := normalizeRecipeItem(recipeData["base"]); base != nil {
				entry["base"] = base
			}
			if addition := normalizeRecipeItem(recipeData["addition"]); addition != nil {
				entry["addition"] = addition
			}
			if mergeComponents := recipeData["merge-components"]; mergeComponents != nil {
				entry["merge-components"] = mergeComponents
			}
		case "brewing":
			if ingredient := normalizeRecipeItem(recipeData["ingredient"]); ingredient != nil {
				entry["ingredient"] = ingredient
			}
			if container := normalizeRecipeItem(recipeData["container"]); container != nil {
				entry["container"] = container
			}
		}

		if result := normalizeRecipeResult(recipeData["result"]); result != nil {
			entry["result"] = result
		}

		c.config.Recipes[c.namespaced(recipeKey)] = entry
	}
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

	c.config.Categories[c.namespaced("default")] = map[string]any{
		"name":     "<!i>" + strings.Title(c.namespace),
		"priority": 1,
		"icon":     itemIDs[0],
		"list":     itemIDs,
		"hidden":   false,
	}
}

func (c *Converter) namespaced(key string) string {
	if strings.Contains(key, ":") {
		return key
	}
	return c.namespace + ":" + key
}

func (c *Converter) namespacedPath(raw string) string {
	if raw == "" {
		return raw
	}
	if strings.Contains(raw, ":") {
		parts := strings.SplitN(raw, ":", 2)
		if parts[0] == "minecraft" {
			return "minecraft:" + strings.ToLower(parts[1])
		}
		return c.namespace + ":" + parts[1]
	}
	if strings.HasPrefix(raw, "#") {
		tag := strings.TrimPrefix(raw, "#")
		if strings.Contains(tag, ":") {
			parts := strings.SplitN(tag, ":", 2)
			if parts[0] == "minecraft" {
				return "#minecraft:" + strings.ToLower(parts[1])
			}
			return "#" + parts[0] + ":" + parts[1]
		}
		return "#minecraft:" + strings.ToLower(tag)
	}
	return c.namespace + ":" + raw
}

func (c *Converter) modelRef(modelPath string) string {
	value := normalizePath(modelPath)
	value = strings.TrimSuffix(value, ".json")
	if parts := strings.SplitN(value, ":", 2); len(parts) == 2 {
		value = parts[1]
	}
	if !strings.HasPrefix(value, "item/") {
		value = "item/" + strings.TrimPrefix(value, "/")
	}
	return c.namespace + ":" + value
}

func (c *Converter) normalizeEquipmentTexturePath(raw string, leggings bool) string {
	value := normalizePath(raw)
	value = strings.TrimSuffix(value, ".png")
	if parts := strings.SplitN(value, ":", 2); len(parts) == 2 {
		value = parts[1]
	}
	value = strings.TrimPrefix(value, "textures/")

	parts := splitFiltered(value, map[string]struct{}{
		"textures":           {},
		"entity":             {},
		"equipment":          {},
		"humanoid":           {},
		"humanoid_legging":   {},
		"humanoid_leggings":  {},
		"armor":              {},
		"armour":             {},
	})

	target := "humanoid"
	if leggings {
		target = "humanoid_legging"
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s:entity/equipment/%s/unknown", c.namespace, target)
	}
	return fmt.Sprintf("%s:entity/equipment/%s/%s", c.namespace, target, strings.Join(parts, "/"))
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

func formatDisplayName(name string) string {
	return "<!i><white>" + strings.ReplaceAll(name, "&", "§")
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

func normalizeTextures(resource map[string]any) []string {
	var textures []string
	switch value := resource["textures"].(type) {
	case []any:
		for _, item := range value {
			textures = append(textures, normalizeTexture(item))
		}
	case string:
		textures = append(textures, normalizeTexture(value))
	}
	if len(textures) == 0 {
		if texture := stringValue(resource["texture"]); texture != "" {
			textures = append(textures, normalizeTexture(texture))
		}
	}

	filtered := make([]string, 0, len(textures))
	for _, item := range textures {
		if item != "" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func normalizeTexture(raw any) string {
	value := normalizePath(stringValue(raw))
	value = strings.TrimSuffix(value, ".png")
	if parts := strings.SplitN(value, ":", 2); len(parts) == 2 {
		value = parts[1]
	}
	value = strings.TrimPrefix(value, "textures/")
	if !strings.HasPrefix(value, "item/") && !strings.HasPrefix(value, "block/") {
		value = "item/" + strings.TrimPrefix(value, "/")
	}
	return value
}

func mapRecipeType(groupKey string, recipe map[string]any) string {
	group := strings.ToLower(groupKey)
	if boolValue(recipe["shapeless"]) {
		return "shapeless"
	}

	mapping := map[string]string{
		"crafting_table":    "shaped",
		"shapeless":         "shapeless",
		"shapeless_crafting":"shapeless",
		"furnace":           "smelting",
		"smelting":          "smelting",
		"blast_furnace":     "blasting",
		"blasting":          "blasting",
		"smoker":            "smoking",
		"smoking":           "smoking",
		"campfire":          "campfire_cooking",
		"campfire_cooking":  "campfire_cooking",
		"stonecutting":      "stonecutting",
		"smithing":          "smithing_transform",
		"smithing_transform":"smithing_transform",
		"brewing":           "brewing",
	}
	if value, ok := mapping[group]; ok {
		return value
	}
	if _, ok := recipe["pattern"]; ok {
		return "shaped"
	}
	if _, ok := recipe["ingredients"].([]any); ok {
		return "shapeless"
	}
	return ""
}

func normalizeRecipeResult(raw any) map[string]any {
	if raw == nil {
		return nil
	}

	resultID := ""
	count := 1

	if data, ok := asStringMap(raw); ok {
		resultID = stringValue(firstNonNil(data["item"], data["id"]))
		if numeric, ok := numericValue(firstNonNil(data["amount"], data["count"])); ok {
			count = numeric
		}
	} else {
		resultID = stringValue(raw)
	}

	if resultID == "" {
		return nil
	}

	return map[string]any{
		"id":    normalizeRecipeItem(resultID),
		"count": count,
	}
}

func normalizeRecipeItem(raw any) any {
	switch value := raw.(type) {
	case nil:
		return nil
	case map[string]any:
		return normalizeRecipeItem(firstNonNil(value["item"], value["id"]))
	case string:
		item := strings.TrimSpace(value)
		if item == "" {
			return item
		}
		if strings.HasPrefix(item, "#") {
			tag := strings.TrimPrefix(item, "#")
			if strings.Contains(tag, ":") {
				parts := strings.SplitN(tag, ":", 2)
				if parts[0] == "minecraft" {
					return "#minecraft:" + strings.ToLower(parts[1])
				}
				return "#" + parts[0] + ":" + parts[1]
			}
			return "#minecraft:" + strings.ToLower(tag)
		}
		if strings.Contains(item, ":") {
			parts := strings.SplitN(item, ":", 2)
			if parts[0] == "minecraft" {
				return "minecraft:" + strings.ToLower(parts[1])
			}
			return parts[0] + ":" + parts[1]
		}
		return "minecraft:" + strings.ToLower(item)
	default:
		return value
	}
}

func normalizePattern(pattern []string, ingredients any) []string {
	keys := map[string]struct{}{}
	if data, ok := asStringMap(ingredients); ok {
		for key := range data {
			keys[key] = struct{}{}
		}
	}
	if len(keys) == 0 {
		return pattern
	}

	result := make([]string, 0, len(pattern))
	for _, row := range pattern {
		var builder strings.Builder
		for _, char := range row {
			if _, ok := keys[string(char)]; ok {
				builder.WriteRune(char)
			} else {
				builder.WriteByte(' ')
			}
		}
		result = append(result, builder.String())
	}
	return result
}

func normalizeIngredientMap(raw map[string]any) map[string]any {
	result := make(map[string]any, len(raw))
	for key, value := range raw {
		result[key] = normalizeRecipeItem(value)
	}
	return result
}

func normalizeShapelessIngredients(raw any) []any {
	switch value := raw.(type) {
	case []any:
		result := make([]any, 0, len(value))
		for _, item := range value {
			if nested, ok := item.([]any); ok {
				group := make([]any, 0, len(nested))
				for _, nestedItem := range nested {
					group = append(group, normalizeRecipeItem(nestedItem))
				}
				result = append(result, group)
			} else {
				result = append(result, normalizeRecipeItem(item))
			}
		}
		return result
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		result := make([]any, 0, len(keys))
		for _, key := range keys {
			result = append(result, normalizeRecipeItem(value[key]))
		}
		return result
	default:
		return nil
	}
}

func isArmorMaterial(material string) bool {
	for _, suffix := range []string{"_HELMET", "_CHESTPLATE", "_LEGGINGS", "_BOOTS"} {
		if strings.HasSuffix(material, suffix) {
			return true
		}
	}
	return false
}

func hasEquipmentSetting(item map[string]any) bool {
	settings, ok := asStringMap(item["data"])
	if !ok {
		return false
	}
	_, exists := settings["equipment"]
	return exists
}

func stripMinecraftColorCodes(value string) string {
	replacer := strings.NewReplacer(
		"&0", "", "&1", "", "&2", "", "&3", "", "&4", "", "&5", "", "&6", "", "&7", "", "&8", "", "&9", "",
		"&a", "", "&b", "", "&c", "", "&d", "", "&e", "", "&f", "", "&k", "", "&l", "", "&m", "", "&n", "", "&o", "", "&r", "",
		"§0", "", "§1", "", "§2", "", "§3", "", "§4", "", "§5", "", "§6", "", "§7", "", "§8", "", "§9", "",
		"§a", "", "§b", "", "§c", "", "§d", "", "§e", "", "§f", "", "§k", "", "§l", "", "§m", "", "§n", "", "§o", "", "§r", "",
	)
	return replacer.Replace(value)
}

func mergeNested(dst map[string]map[string]any, src map[string]map[string]any) {
	for key, value := range src {
		dst[key] = value
	}
}

func mergeRecipeGroups(dst map[string]map[string]map[string]any, src map[string]map[string]map[string]any) {
	for group, recipes := range src {
		if _, ok := dst[group]; !ok {
			dst[group] = map[string]map[string]any{}
		}
		for key, value := range recipes {
			dst[group][key] = value
		}
	}
}

func asNestedMap(raw any) (map[string]map[string]any, bool) {
	input, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	result := make(map[string]map[string]any, len(input))
	for key, value := range input {
		if item, ok := asStringMap(value); ok {
			result[key] = item
		}
	}
	return result, true
}

func asRecipeGroups(raw any) (map[string]map[string]map[string]any, bool) {
	input, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	result := map[string]map[string]map[string]any{}
	for groupKey, groupValue := range input {
		groupMap, ok := groupValue.(map[string]any)
		if !ok {
			continue
		}
		result[groupKey] = map[string]map[string]any{}
		for recipeKey, recipeValue := range groupMap {
			if recipeMap, ok := asStringMap(recipeValue); ok {
				result[groupKey][recipeKey] = recipeMap
			}
		}
	}
	return result, true
}

func asStringMap(raw any) (map[string]any, bool) {
	value, ok := raw.(map[string]any)
	return value, ok
}

func asStringSlice(raw any) ([]string, bool) {
	value, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(value))
	for _, item := range value {
		result = append(result, stringValue(item))
	}
	return result, true
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
	case uint64:
		return int(value), true
	default:
		return 0, false
	}
}

func boolValue(raw any) bool {
	switch value := raw.(type) {
	case bool:
		return value
	default:
		return false
	}
}

func boolDefaultTrue(raw any) bool {
	if raw == nil {
		return true
	}
	if value, ok := raw.(bool); ok {
		return value
	}
	return true
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func flattenSingleList(raw any) any {
	list, ok := raw.([]any)
	if !ok {
		return raw
	}
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

func normalizePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimPrefix(value, "/")
	return value
}

func splitFiltered(value string, excluded map[string]struct{}) []string {
	parts := strings.Split(value, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, skip := excluded[strings.ToLower(part)]; skip {
			continue
		}
		result = append(result, part)
	}
	return result
}

func (c *Converter) MarshalConfig() ([]byte, error) {
	return yaml.Marshal(c.config)
}

func (c *Converter) DebugString() string {
	raw, err := c.MarshalConfig()
	if err != nil {
		return err.Error()
	}
	return bytes.NewBuffer(raw).String()
}
