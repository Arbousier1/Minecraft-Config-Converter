package packageindex

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/yamlx"
)

type YAMLDoc struct {
	Path string
	Data map[string]any
}

type Index struct {
	ExtractDir        string
	Directories       []string
	YAMLDocs          []YAMLDoc
	TextureCount      int
	ModelCount        int
	HasResourceFiles  bool
	HasItemsAdderRoot bool
	HasCraftEngineDir bool
	HasNexoRoot       bool

	dirSet map[string]struct{}
}

func Build(extractDir string) (*Index, error) {
	index := &Index{
		ExtractDir:  filepath.Clean(extractDir),
		Directories: make([]string, 0, 64),
		YAMLDocs:    make([]YAMLDoc, 0, 32),
		dirSet:      make(map[string]struct{}, 64),
	}

	err := filepath.WalkDir(index.ExtractDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		cleanPath := filepath.Clean(path)
		nameLower := strings.ToLower(entry.Name())
		pathLower := strings.ToLower(cleanPath)

		if entry.IsDir() {
			index.Directories = append(index.Directories, cleanPath)
			index.dirSet[cleanPath] = struct{}{}

			switch nameLower {
			case "itemsadder":
				index.HasItemsAdderRoot = true
			case "craftengine":
				index.HasCraftEngineDir = true
			case "nexo":
				index.HasNexoRoot = true
			case "resourcepack":
				index.HasResourceFiles = true
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		switch {
		case ext == ".png" && strings.Contains(pathLower, "textures"):
			index.TextureCount++
			index.HasResourceFiles = true
		case ext == ".json" && strings.Contains(pathLower, "models"):
			index.ModelCount++
			index.HasResourceFiles = true
		case ext == ".yml" || ext == ".yaml":
			raw, err := os.ReadFile(cleanPath)
			if err != nil {
				return nil
			}
			data, err := yamlx.LoadMap(raw)
			if err != nil || len(data) == 0 {
				return nil
			}
			index.YAMLDocs = append(index.YAMLDocs, YAMLDoc{
				Path: cleanPath,
				Data: data,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return index, nil
}

func (i *Index) ItemsAdderScanRoot() string {
	for _, dir := range i.Directories {
		if strings.EqualFold(filepath.Base(dir), "itemsadder") {
			return dir
		}
	}
	return i.ExtractDir
}

func (i *Index) NexoScanRoot() string {
	for _, dir := range i.Directories {
		if strings.EqualFold(filepath.Base(dir), "nexo") {
			return dir
		}
	}
	return i.ExtractDir
}

func (i *Index) ItemsAdderResourcepackPath() string {
	return i.findResourceRoot(i.ItemsAdderScanRoot(), "resourcepack", true)
}

func (i *Index) NexoResourcepackPath() string {
	return i.findResourceRoot(i.NexoScanRoot(), "pack", false)
}

func (i *Index) ItemsAdderDocs() []YAMLDoc {
	return i.docsUnder(i.ItemsAdderScanRoot(), nil)
}

func (i *Index) NexoDocs() []YAMLDoc {
	return i.docsUnder(i.NexoScanRoot(), map[string]struct{}{
		"config.yml":  {},
		"config.yaml": {},
	})
}

func (i *Index) findResourceRoot(scanRoot, preferredName string, acceptModelTexturePair bool) string {
	for _, dir := range i.Directories {
		if !isWithin(scanRoot, dir) {
			continue
		}

		if strings.EqualFold(filepath.Base(dir), preferredName) {
			return dir
		}

		if i.hasDir(filepath.Join(dir, "assets")) {
			return dir
		}
		if acceptModelTexturePair && i.hasDir(filepath.Join(dir, "models")) && i.hasDir(filepath.Join(dir, "textures")) {
			return dir
		}
	}
	return scanRoot
}

func (i *Index) docsUnder(root string, excludedNames map[string]struct{}) []YAMLDoc {
	docs := make([]YAMLDoc, 0, len(i.YAMLDocs))
	for _, doc := range i.YAMLDocs {
		if !isWithin(root, doc.Path) {
			continue
		}
		if excludedNames != nil {
			if _, skip := excludedNames[strings.ToLower(filepath.Base(doc.Path))]; skip {
				continue
			}
		}
		docs = append(docs, doc)
	}
	return docs
}

func (i *Index) hasDir(path string) bool {
	_, ok := i.dirSet[filepath.Clean(path)]
	return ok
}

func isWithin(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == path {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")
}
