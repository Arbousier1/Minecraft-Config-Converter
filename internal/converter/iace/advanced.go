package iace

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func (c *Converter) isArmor(material string, data map[string]any) bool {
	if isArmorMaterial(material) {
		return true
	}
	if specific, ok := asStringMap(data["specific_properties"]); ok {
		if armor, ok := asStringMap(specific["armor"]); ok && len(armor) > 0 {
			return true
		}
	}
	return data["equipment"] != nil
}

func (c *Converter) handleArmor(ceItem map[string]any, data map[string]any) {
	equipmentID := ""
	slot := "head"

	if equipment, ok := asStringMap(data["equipment"]); ok {
		equipmentID = stringValue(equipment["id"])
	}

	if equipmentID == "" {
		if specific, ok := asStringMap(data["specific_properties"]); ok {
			if armor, ok := asStringMap(specific["armor"]); ok {
				equipmentID = stringValue(armor["custom_armor"])
				if rawSlot := stringValue(armor["slot"]); rawSlot != "" {
					slot = rawSlot
				}
			}
		}
	}

	material := stringValue(ceItem["material"])
	switch {
	case strings.HasSuffix(material, "_CHESTPLATE"):
		slot = "chest"
	case strings.HasSuffix(material, "_LEGGINGS"):
		slot = "legs"
	case strings.HasSuffix(material, "_BOOTS"):
		slot = "feet"
	}

	if equipmentID != "" {
		if material == "STONE" {
			switch slot {
			case "head":
				ceItem["material"] = "DIAMOND_HELMET"
			case "chest":
				ceItem["material"] = "DIAMOND_CHESTPLATE"
			case "legs":
				ceItem["material"] = "DIAMOND_LEGGINGS"
			case "feet":
				ceItem["material"] = "DIAMOND_BOOTS"
			}
		}

		if parts := strings.SplitN(equipmentID, ":", 2); len(parts) == 2 {
			equipmentID = parts[1]
		}

		ceItem["settings"] = map[string]any{
			"equipment": map[string]any{
				"asset-id": c.namespaced(equipmentID),
				"slot":     slot,
			},
		}
	}

	resource, _ := asStringMap(data["resource"])
	c.handleGenericModel(ceItem, resource)
}

func (c *Converter) handleFurniture(ceItem map[string]any, data map[string]any, ceID string, furniture map[string]any) {
	behaviours, _ := asStringMap(data["behaviours"])
	sitData, _ := asStringMap(behaviours["furniture_sit"])
	entityType := stringValue(furniture["entity"])
	if entityType == "" {
		entityType = "armor_stand"
	}

	resource, _ := asStringMap(data["resource"])
	translationY := c.calculateModelYTranslation(stringValue(resource["model_path"]))

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
		},
	}

	placeableOn, _ := asStringMap(furniture["placeable_on"])
	if len(placeableOn) == 0 {
		placeableOn = map[string]any{"floor": true}
	}

	placement := map[string]any{}
	if boolDefaultTrue(placeableOn["floor"]) {
		placement["ground"] = c.createPlacementBlock(ceID, furniture, "ground", sitData, translationY)
	}
	if boolValue(placeableOn["walls"]) {
		placement["wall"] = c.createPlacementBlock(ceID, furniture, "wall", sitData, translationY)
	}
	if boolValue(placeableOn["ceiling"]) {
		placement["ceiling"] = c.createPlacementBlock(ceID, furniture, "ceiling", sitData, translationY)
	}

	behavior := ceItem["behavior"].(map[string]any)
	behavior["furniture"].(map[string]any)["placement"] = placement
	c.handleGenericModel(ceItem, resource)
}

func (c *Converter) calculateModelYTranslation(modelPath string) float64 {
	if modelPath == "" || c.resourcepackPath == "" {
		return 0.5
	}
	if cached, ok := c.modelYCache[modelPath]; ok {
		return cached
	}

	targetNamespace := c.namespace
	cleanPath := modelPath
	if parts := strings.SplitN(modelPath, ":", 2); len(parts) == 2 {
		targetNamespace = parts[0]
		cleanPath = parts[1]
	}

	candidates := []string{
		filepath.Join(c.resourcepackPath, "assets", targetNamespace, "models", cleanPath+".json"),
		filepath.Join(c.resourcepackPath, targetNamespace, "models", cleanPath+".json"),
	}
	for _, candidate := range candidates {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		elements, ok := payload["elements"].([]any)
		if !ok {
			c.modelYCache[modelPath] = 0.5
			return 0.5
		}
		for _, rawElement := range elements {
			element, ok := asStringMap(rawElement)
			if !ok {
				continue
			}
			if axisBelowThreshold(element["from"], -7.0) || axisBelowThreshold(element["to"], -7.0) {
				c.modelYCache[modelPath] = 1.5
				return 1.5
			}
		}
		c.modelYCache[modelPath] = 0.5
		return 0.5
	}

	c.modelYCache[modelPath] = 0.5
	return 0.5
}

func (c *Converter) createPlacementBlock(ceID string, furniture map[string]any, placementType string, sitData map[string]any, customTranslationY float64) map[string]any {
	height, width, length := 1.0, 1.0, 1.0
	hitbox, hasHitbox := asStringMap(furniture["hitbox"])
	if hasHitbox {
		if value, ok := floatValue(hitbox["height"]); ok {
			height = value
		}
		if value, ok := floatValue(hitbox["width"]); ok {
			width = value
		}
		if value, ok := floatValue(hitbox["length"]); ok {
			length = value
		}
	}

	translationY := customTranslationY
	if translationY == 0 {
		translationY = height / 2.0
	}
	if placementType == "wall" || placementType == "ceiling" {
		translationY = 0
	}

	translationX, translationZ := 0.0, 0.0
	if height == 2 && width == 3 && length == 2 {
		translationZ = 0.5
	}

	displayTransformation, _ := asStringMap(furniture["display_transformation"])
	scaleX, scaleY, scaleZ := 1.0, 1.0, 1.0
	if scale, ok := asStringMap(displayTransformation["scale"]); ok {
		scaleX, _ = floatValue(scale["x"])
		scaleY, _ = floatValue(scale["y"])
		scaleZ, _ = floatValue(scale["z"])
		if scaleX == 0 {
			scaleX = 1
		}
		if scaleY == 0 {
			scaleY = 1
		}
		if scaleZ == 0 {
			scaleZ = 1
		}
	}

	elementEntry := map[string]any{
		"item":              ceID,
		"display-transform": "NONE",
		"billboard":         "FIXED",
		"translation":       fmt.Sprintf("%g,%g,%g", translationX, translationY, translationZ),
	}

	if scaleX != 1 || scaleY != 1 || scaleZ != 1 {
		elementEntry["scale"] = fmt.Sprintf("%g,%g,%g", scaleX, scaleY, scaleZ)
	}
	switch placementType {
	case "wall":
		elementEntry["position"] = "0,0,0.5"
	case "ceiling":
		elementEntry["position"] = "0,-1,0"
	}

	blockConfig := map[string]any{
		"loot-spawn-offset": "0,0.4,0",
		"rules": map[string]any{
			"rotation":  "eight",
			"alignment": "center",
		},
		"elements": []any{elementEntry},
	}

	wOffset, hOffset, lOffset := 0.0, 0.0, 0.0
	if hasHitbox {
		wOffset, _ = floatValue(hitbox["width_offset"])
		hOffset, _ = floatValue(hitbox["height_offset"])
		lOffset, _ = floatValue(hitbox["length_offset"])
	}
	if placementType == "ceiling" {
		hOffset -= height
	}

	hitboxes := []any{}
	isSolid := boolDefaultTrue(furniture["solid"])
	if placementType != "wall" {
		if len(sitData) > 0 {
			sitHeight, ok := floatValue(sitData["sit_height"])
			if !ok {
				sitHeight = 0.5
			}
			seatY := sitHeight - 0.85
			seats := []string{}
			wRange := maxInt(1, int(math.Round(width)))
			if wRange <= 1 {
				seats = append(seats, fmt.Sprintf("0,%g,0", seatY))
			} else {
				for i := 0; i < wRange; i++ {
					offsetX := float64(i) - float64(wRange-1)/2.0
					seats = append(seats, fmt.Sprintf("%g,%g,0", offsetX, seatY))
				}
			}
			hitboxes = append(hitboxes, map[string]any{
				"position":        fmt.Sprintf("%g,%g,%g", wOffset, hOffset, lOffset),
				"type":            "interaction",
				"blocks-building": isSolid,
				"width":           width,
				"height":          height,
				"interactive":     true,
				"seats":           seats,
			})
		} else if hasHitbox {
			if isSolid {
				wRange := maxInt(1, int(math.Round(width)))
				hRange := maxInt(1, int(math.Round(height)))
				lRange := maxInt(1, int(math.Round(length)))
				for y := 0; y < hRange; y++ {
					for x := 0; x < wRange; x++ {
						for z := 0; z < lRange; z++ {
							relX := float64(x) - float64(wRange-1)/2.0
							relY := float64(y)
							relZ := float64(z) - float64(lRange-1)/2.0
							finalX := math.Floor(relX + wOffset + 0.5)
							finalY := math.Floor(relY + hOffset + 0.5)
							finalZ := math.Floor(relZ + lOffset + 0.5)
							hitboxes = append(hitboxes, map[string]any{
								"position":        fmt.Sprintf("%d,%d,%d", int(finalX), int(finalY), int(finalZ)),
								"type":            "shulker",
								"blocks-building": true,
								"interactive":     true,
							})
						}
					}
				}
			} else {
				hitboxes = append(hitboxes, map[string]any{
					"position":        fmt.Sprintf("%g,%g,%g", wOffset, hOffset, lOffset),
					"type":            "interaction",
					"blocks-building": false,
					"width":           width,
					"height":          height,
					"interactive":     true,
				})
			}
		}
	}

	if placementType == "ceiling" && len(hitboxes) == 0 {
		hitboxes = append(hitboxes, map[string]any{
			"type":            "interaction",
			"position":        "0,-1,0",
			"width":           width,
			"height":          height,
			"interactive":     true,
			"blocks-building": false,
		})
	} else if placementType == "wall" && len(hitboxes) == 0 {
		hitboxes = append(hitboxes, map[string]any{
			"type":            "interaction",
			"position":        "0,-0.5,0",
			"width":           width,
			"height":          height,
			"interactive":     true,
			"blocks-building": false,
		})
	}

	if len(hitboxes) > 0 {
		blockConfig["hitboxes"] = hitboxes
	}

	return blockConfig
}

func isComplexItem(material string) bool {
	switch material {
	case "BOW", "CROSSBOW", "FISHING_ROD", "SHIELD":
		return true
	default:
		return false
	}
}

func (c *Converter) getModelRef(path string) string {
	if strings.HasPrefix(path, "item/") {
		return c.namespace + ":" + path
	}
	return c.namespace + ":item/" + path
}

func (c *Converter) findModelPathVariant(basePath string, variants []string) string {
	if c.resourcepackPath == "" {
		return basePath + variants[0]
	}
	cacheKey := basePath + "\x00" + strings.Join(variants, "\x00")
	if cached, ok := c.modelVariantCache[cacheKey]; ok {
		return cached
	}

	ns := c.namespace
	relPath := basePath
	if parts := strings.SplitN(basePath, ":", 2); len(parts) == 2 {
		ns = parts[0]
		relPath = parts[1]
	}
	relPath = strings.TrimSuffix(relPath, ".json")

	for _, suffix := range variants {
		candidate := relPath + suffix
		fullPaths := []string{
			filepath.Join(c.resourcepackPath, "assets", ns, "models", candidate+".json"),
			filepath.Join(c.resourcepackPath, ns, "models", candidate+".json"),
		}
		for _, fullPath := range fullPaths {
			if c.modelFileExists(fullPath) {
				if strings.Contains(basePath, ":") {
					c.modelVariantCache[cacheKey] = ns + ":" + candidate
					return c.modelVariantCache[cacheKey]
				}
				c.modelVariantCache[cacheKey] = candidate
				return c.modelVariantCache[cacheKey]
			}
		}
	}

	baseClean := strings.TrimSuffix(basePath, ".json")
	c.modelVariantCache[cacheKey] = baseClean + variants[0]
	return c.modelVariantCache[cacheKey]
}

func (c *Converter) handleComplexItem(ceItem map[string]any, key string, data map[string]any, material string) {
	resource, _ := asStringMap(data["resource"])
	baseModelPath := stringValue(resource["model_path"])
	textures := rawTextures(resource)

	if (material == "BOW" || material == "CROSSBOW") && len(textures) > 0 {
		expanded := expandBowTextures(textures)
		ceItem["textures"] = c.normalizeTexturesForItem(expanded, ceItem)
		return
	}

	templateID := fmt.Sprintf("models:%s_%s_model", c.namespace, key)
	templateDef := map[string]any{}
	args := map[string]any{}

	switch material {
	case "BOW":
		templateDef = map[string]any{
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
		args["bow_model"] = c.getModelRef(baseModelPath)
		args["bow_pulling_0_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_pulling_0", "_0"}))
		args["bow_pulling_1_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_pulling_1", "_1"}))
		args["bow_pulling_2_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_pulling_2", "_2"}))
	case "CROSSBOW":
		templateDef = map[string]any{
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
		args["model"] = c.getModelRef(baseModelPath)
		args["arrow_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_charged", "_arrow"}))
		args["firework_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_firework", "_rocket"}))
		args["pulling_0_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_pulling_0", "_0"}))
		args["pulling_1_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_pulling_1", "_1"}))
		args["pulling_2_model"] = c.getModelRef(c.findModelPathVariant(baseModelPath, []string{"_pulling_2", "_2"}))
	case "SHIELD":
		templateDef = map[string]any{
			"type":     "minecraft:condition",
			"property": "minecraft:using_item",
			"on-false": map[string]any{"type": "minecraft:model", "path": "${shield_model}"},
			"on-true":  map[string]any{"type": "minecraft:model", "path": "${shield_blocking_model}"},
		}
		args["shield_model"] = c.getModelRef(baseModelPath)
		args["shield_blocking_model"] = c.getModelRef(baseModelPath + "_blocking")
	case "FISHING_ROD":
		templateDef = map[string]any{
			"type":     "minecraft:condition",
			"property": "minecraft:fishing_rod/cast",
			"on-false": map[string]any{"type": "minecraft:model", "path": "${path}"},
			"on-true":  map[string]any{"type": "minecraft:model", "path": "${cast_path}"},
		}
		args["path"] = c.getModelRef(baseModelPath)
		args["cast_path"] = c.getModelRef(baseModelPath + "_cast")
	}

	c.config.Templates[templateID] = templateDef
	ceItem["model"] = map[string]any{
		"template":  templateID,
		"arguments": args,
	}
}

func expandBowTextures(textures []string) []string {
	if len(textures) == 0 {
		return textures
	}
	cleaned := make([]string, 0, len(textures))
	for _, texture := range textures {
		value := normalizePath(texture)
		value = strings.TrimSuffix(value, ".png")
		if parts := strings.SplitN(value, ":", 2); len(parts) == 2 {
			value = parts[1]
		}
		value = strings.TrimPrefix(value, "textures/")
		cleaned = append(cleaned, value)
	}

	base := cleaned[0]
	style := "numeric"
	for _, texture := range cleaned {
		if strings.HasSuffix(texture, "_pulling_0") {
			base = strings.TrimSuffix(texture, "_pulling_0")
			style = "pulling"
			break
		}
	}
	if style == "numeric" {
		for _, texture := range cleaned {
			if strings.HasSuffix(texture, "_0") {
				base = strings.TrimSuffix(texture, "_0")
				break
			}
		}
	}

	variants := []string{}
	if style == "pulling" {
		variants = []string{base + "_pulling_0", base + "_pulling_1", base + "_pulling_2"}
	} else {
		variants = []string{base + "_0", base + "_1", base + "_2"}
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(cleaned)+len(variants)+1)
	for _, item := range append(append([]string{base}, variants...), cleaned...) {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func (c *Converter) normalizeTexturesForItem(textures []string, ceItem map[string]any) []string {
	result := make([]string, 0, len(textures))
	isArmor := isArmorMaterial(stringValue(ceItem["material"]))
	for _, texture := range textures {
		value := normalizePath(texture)
		value = strings.TrimSuffix(value, ".png")
		if parts := strings.SplitN(value, ":", 2); len(parts) == 2 {
			value = parts[1]
		}
		value = strings.TrimPrefix(value, "textures/")

		finalPath := value
		if isArmor {
			if strings.HasPrefix(finalPath, "item/armor/") {
			} else {
				finalPath = strings.TrimPrefix(finalPath, "item/")
				finalPath = strings.TrimPrefix(finalPath, "armor/")
				finalPath = "item/armor/" + finalPath
			}
		} else if !strings.HasPrefix(finalPath, "item/") && !strings.HasPrefix(finalPath, "block/") {
			finalPath = "item/" + finalPath
		}
		result = append(result, c.namespace+":"+finalPath)
	}
	return result
}

func (c *Converter) handleGenericModel(ceItem map[string]any, resource map[string]any) {
	modelPath := stringValue(resource["model_path"])
	if modelPath != "" {
		ceItem["model"] = map[string]any{
			"type": "minecraft:model",
			"path": c.modelRef(modelPath),
		}
		return
	}

	textures := rawTextures(resource)
	if len(textures) == 0 {
		return
	}
	ceItem["textures"] = c.normalizeTexturesForItem(textures, ceItem)
}

func (c *Converter) modelFileExists(path string) bool {
	if exists, ok := c.modelFileCache[path]; ok {
		return exists
	}
	_, err := os.Stat(path)
	exists := err == nil
	c.modelFileCache[path] = exists
	return exists
}

func (c *Converter) registerArmorTexture(raw string, leggings bool) {
	key := normalizeArmorKey(raw)
	if key == "" {
		return
	}
	if leggings {
		c.armorLeggingsKeys[key] = struct{}{}
	} else {
		c.armorHumanoidKeys[key] = struct{}{}
	}
}

func normalizeArmorKey(raw string) string {
	value := normalizePath(raw)
	value = strings.TrimSuffix(value, ".png")
	if parts := strings.SplitN(value, ":", 2); len(parts) == 2 {
		value = parts[1]
	}
	return strings.TrimPrefix(value, "textures/")
}

func axisBelowThreshold(raw any, threshold float64) bool {
	values, ok := raw.([]any)
	if !ok || len(values) < 2 {
		return false
	}
	value, ok := floatValue(values[1])
	return ok && value < threshold
}

func floatValue(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	default:
		return 0, false
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func rawTextures(resource map[string]any) []string {
	switch value := resource["textures"].(type) {
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			result = append(result, stringValue(item))
		}
		return result
	case string:
		return []string{value}
	default:
		if texture := stringValue(resource["texture"]); texture != "" {
			return []string{texture}
		}
		return nil
	}
}

func hasStringKey(raw any, key string) bool {
	data, ok := asStringMap(raw)
	if !ok {
		return false
	}
	_, exists := data[key]
	return exists
}
