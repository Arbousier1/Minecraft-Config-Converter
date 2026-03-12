package analyzer

import "github.com/Arbousier1/Minecraft-Config-Converter/internal/packageindex"

const (
	contentTypeEquipment = "装备"
	contentTypeDecor     = "装饰"
	contentTypeTexture   = "贴图"
	contentTypeModel     = "模型"
)

type Completeness struct {
	ItemsConfig      bool `json:"items_config"`
	CategoriesConfig bool `json:"categories_config"`
	ResourceFiles    bool `json:"resource_files"`
}

type Details struct {
	ItemCount    int `json:"item_count"`
	TextureCount int `json:"texture_count"`
	ModelCount   int `json:"model_count"`
}

type Report struct {
	Formats      []string     `json:"formats"`
	ContentTypes []string     `json:"content_types"`
	Completeness Completeness `json:"completeness"`
	Details      Details      `json:"details"`
}

type Analyzer struct {
	extractPath string
	index       *packageindex.Index
	formats     map[string]struct{}
	content     map[string]struct{}
	report      Report
}

func New(extractPath string) *Analyzer {
	return &Analyzer{
		extractPath: extractPath,
		formats:     make(map[string]struct{}),
		content:     make(map[string]struct{}),
		report: Report{
			Formats:      []string{},
			ContentTypes: []string{},
			Completeness: Completeness{},
			Details:      Details{},
		},
	}
}

func NewFromIndex(index *packageindex.Index) *Analyzer {
	analyzer := New(index.ExtractDir)
	analyzer.index = index
	return analyzer
}

func (a *Analyzer) Analyze() (Report, error) {
	index := a.index
	if index == nil {
		var err error
		index, err = packageindex.Build(a.extractPath)
		if err != nil {
			return Report{}, err
		}
		a.index = index
	}

	a.resetReport()
	if index.HasItemsAdderRoot {
		a.addFormat("ItemsAdder")
	}
	if index.HasCraftEngineDir {
		a.addFormat("CraftEngine")
	}
	if index.HasNexoRoot {
		a.addFormat("Nexo")
	}
	if index.TextureCount > 0 {
		a.addContent(contentTypeTexture)
	}
	if index.ModelCount > 0 {
		a.addContent(contentTypeModel)
	}
	a.report.Details.TextureCount = index.TextureCount
	a.report.Details.ModelCount = index.ModelCount
	a.report.Completeness.ResourceFiles = index.HasResourceFiles

	for _, doc := range index.YAMLDocs {
		a.analyzeYAML(doc.Data)
	}

	a.report.Formats = setToSortedSlice(a.formats, []string{"ItemsAdder", "CraftEngine", "Nexo"})
	a.report.ContentTypes = setToSortedSlice(a.content, []string{
		contentTypeEquipment,
		contentTypeDecor,
		contentTypeTexture,
		contentTypeModel,
	})
	return a.report, nil
}

func (a *Analyzer) analyzeYAML(data map[string]any) {
	isIA := isIAConfig(data)
	isCE := isCEConfig(data)
	isNexo := isNexoConfig(data)

	if isIA {
		a.addFormat("ItemsAdder")
		if items, ok := asMap(data["items"]); ok {
			a.report.Completeness.ItemsConfig = true
			a.addContent(contentTypeEquipment)
			a.report.Details.ItemCount += len(items)
			for _, rawItem := range items {
				item, ok := asMap(rawItem)
				if !ok {
					continue
				}
				behaviours, ok := asMap(item["behaviours"])
				if !ok {
					continue
				}
				if _, hasFurniture := behaviours["furniture"]; hasFurniture {
					a.addContent(contentTypeDecor)
				}
			}
		}
		if _, ok := asMap(data["categories"]); ok {
			a.report.Completeness.CategoriesConfig = true
		}
	}

	if isCE {
		a.addFormat("CraftEngine")
	}

	if isNexo {
		a.addFormat("Nexo")
	}
}

func (a *Analyzer) resetReport() {
	a.formats = make(map[string]struct{})
	a.content = make(map[string]struct{})
	a.report = Report{
		Formats:      []string{},
		ContentTypes: []string{},
		Completeness: Completeness{},
		Details:      Details{},
	}
}

func isIAConfig(data map[string]any) bool {
	if info, ok := asMap(data["info"]); ok {
		if _, exists := info["namespace"]; exists {
			return true
		}
	}

	for _, key := range []string{"items", "categories", "equipments", "armors_rendering", "recipes", "loots", "info"} {
		raw, ok := data[key]
		if !ok {
			continue
		}

		if key != "items" {
			return true
		}

		items, ok := asMap(raw)
		if !ok {
			continue
		}
		for _, rawItem := range items {
			item, ok := asMap(rawItem)
			if !ok {
				continue
			}
			if _, hasResource := item["resource"]; hasResource {
				return true
			}
			if _, hasBehaviours := item["behaviours"]; hasBehaviours {
				return true
			}
		}
	}

	return false
}

func isCEConfig(data map[string]any) bool {
	items, ok := asMap(data["items"])
	if !ok {
		return false
	}

	for _, rawItem := range items {
		item, ok := asMap(rawItem)
		if !ok {
			continue
		}

		if behavior, ok := asMap(item["behavior"]); ok {
			if behaviorType, ok := behavior["type"].(string); ok && behaviorType == "furniture_item" {
				return true
			}
		}

		if _, hasModel := item["model"]; hasModel {
			return true
		}
	}

	return false
}

func isNexoConfig(data map[string]any) bool {
	for _, rawValue := range data {
		value, ok := asMap(rawValue)
		if !ok {
			continue
		}
		for _, key := range []string{"Mechanics", "Pack", "Components", "itemname"} {
			if _, exists := value[key]; exists {
				return true
			}
		}
	}

	return false
}

func asMap(value any) (map[string]any, bool) {
	m, ok := value.(map[string]any)
	if ok {
		return m, true
	}
	return nil, false
}

func (a *Analyzer) addFormat(name string) {
	a.formats[name] = struct{}{}
}

func (a *Analyzer) addContent(name string) {
	a.content[name] = struct{}{}
}

func setToSortedSlice(items map[string]struct{}, order []string) []string {
	result := make([]string, 0, len(items))
	for _, candidate := range order {
		if _, ok := items[candidate]; ok {
			result = append(result, candidate)
		}
	}
	return result
}
