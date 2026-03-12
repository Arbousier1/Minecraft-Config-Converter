package workflow

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/analyzer"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/converter/iace"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/converter/nexoce"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/packageindex"
)

type Service struct {
	workRoot string
}

type Session struct {
	ID                    string
	OriginalFilename      string
	SourceZipPath         string
	SessionDir            string
	SessionUploadDir      string
	SessionOutputDir      string
	SessionArchiveDir     string
	ExtractDir            string
	Index                 *packageindex.Index
	Report                analyzer.Report
	AvailableSourceFormat []string
	AvailableTargets      []string
	ItemsAdderNamespace   string
}

func New(baseDir string) (*Service, error) {
	workRoot := filepath.Join(baseDir, "temp_sessions")
	if err := os.MkdirAll(workRoot, 0o755); err != nil {
		return nil, err
	}
	return &Service{workRoot: workRoot}, nil
}

func (s *Service) AnalyzeZip(zipPath string) (*Session, error) {
	if zipPath == "" {
		return nil, errors.New("please choose a zip file")
	}
	if !strings.EqualFold(filepath.Ext(zipPath), ".zip") {
		return nil, errors.New("please choose a .zip file")
	}

	sessionID := newSessionID()
	sessionDir := filepath.Join(s.workRoot, sessionID)
	uploadDir := filepath.Join(sessionDir, "upload")
	outputDir := filepath.Join(sessionDir, "output")
	archiveDir := filepath.Join(sessionDir, "artifacts")
	extractDir := filepath.Join(sessionDir, "extracted")

	for _, dir := range []string{uploadDir, outputDir, archiveDir, extractDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	originalFilename := filepath.Base(zipPath)
	storedZipPath := filepath.Join(uploadDir, originalFilename)
	if err := copyFile(zipPath, storedZipPath); err != nil {
		return nil, err
	}

	if err := extractZip(storedZipPath, extractDir); err != nil {
		_ = os.RemoveAll(sessionDir)
		return nil, err
	}

	index, err := packageindex.Build(extractDir)
	if err != nil {
		_ = os.RemoveAll(sessionDir)
		return nil, err
	}

	report, err := analyzer.NewFromIndex(index).Analyze()
	if err != nil {
		_ = os.RemoveAll(sessionDir)
		return nil, err
	}

	return &Session{
		ID:                    sessionID,
		OriginalFilename:      originalFilename,
		SourceZipPath:         zipPath,
		SessionDir:            sessionDir,
		SessionUploadDir:      uploadDir,
		SessionOutputDir:      outputDir,
		SessionArchiveDir:     archiveDir,
		ExtractDir:            extractDir,
		Index:                 index,
		Report:                report,
		AvailableSourceFormat: sourceFormats(report),
		AvailableTargets:      targetFormats(report),
		ItemsAdderNamespace:   detectItemsAdderNamespace(index),
	}, nil
}

func (s *Service) Convert(session *Session, sourceFormat, targetFormat, namespace, savePath string) (string, error) {
	if session == nil {
		return "", errors.New("please analyze a package first")
	}
	if sourceFormat == "" {
		return "", errors.New("please choose a source format")
	}
	if targetFormat == "" {
		return "", errors.New("please choose a target format")
	}
	if targetFormat != "CraftEngine" {
		return "", fmt.Errorf("unsupported target format: %s", targetFormat)
	}

	savePath = normalizeSavePath(savePath)
	if savePath == "" {
		return "", errors.New("please choose where to save the converted archive")
	}

	if err := os.RemoveAll(session.SessionOutputDir); err != nil {
		return "", err
	}
	if err := os.RemoveAll(session.SessionArchiveDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(session.SessionOutputDir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(session.SessionArchiveDir, 0o755); err != nil {
		return "", err
	}

	if sourceFormat == "ItemsAdder" && strings.TrimSpace(namespace) == "" {
		namespace = session.ItemsAdderNamespace
	}

	var (
		outputFile string
		err        error
	)

	switch sourceFormat {
	case "ItemsAdder":
		var result *iace.Result
		result, err = iace.Run(iace.Options{
			ExtractDir:       session.ExtractDir,
			Index:            session.Index,
			SessionUploadDir: session.SessionUploadDir,
			SessionOutputDir: session.SessionOutputDir,
			OutputDir:        session.SessionArchiveDir,
			OriginalFilename: session.OriginalFilename,
			UserNamespace:    namespace,
			TargetFormat:     targetFormat,
		})
		if result != nil {
			outputFile = result.OutputFile
		}
	case "Nexo":
		var result *nexoce.Result
		result, err = nexoce.Run(nexoce.Options{
			ExtractDir:       session.ExtractDir,
			Index:            session.Index,
			SessionUploadDir: session.SessionUploadDir,
			SessionOutputDir: session.SessionOutputDir,
			OutputDir:        session.SessionArchiveDir,
			OriginalFilename: session.OriginalFilename,
			UserNamespace:    namespace,
			TargetFormat:     targetFormat,
		})
		if result != nil {
			outputFile = result.OutputFile
		}
	default:
		return "", fmt.Errorf("unsupported source format: %s", sourceFormat)
	}
	if err != nil {
		return "", err
	}

	if err := copyFile(outputFile, savePath); err != nil {
		return "", err
	}
	return savePath, nil
}

func SuggestedArchiveName(originalFilename, targetFormat string) string {
	base := strings.TrimSuffix(originalFilename, filepath.Ext(originalFilename))
	if base == "" {
		base = "converted"
	}

	name := fmt.Sprintf("%s [%s by MCC].zip", base, targetFormat)
	replacer := strings.NewReplacer("\\", "", "/", "", ":", "", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
	return replacer.Replace(name)
}

func sourceFormats(report analyzer.Report) []string {
	result := make([]string, 0, 2)
	for _, candidate := range []string{"ItemsAdder", "Nexo"} {
		if contains(report.Formats, candidate) {
			result = append(result, candidate)
		}
	}
	return result
}

func targetFormats(report analyzer.Report) []string {
	if contains(report.Formats, "ItemsAdder") || contains(report.Formats, "Nexo") {
		return []string{"CraftEngine"}
	}
	return nil
}

func detectItemsAdderNamespace(index *packageindex.Index) string {
	for _, doc := range index.ItemsAdderDocs() {
		info, ok := doc.Data["info"].(map[string]any)
		if !ok {
			continue
		}
		namespace, ok := info["namespace"].(string)
		if ok && namespace != "" {
			return namespace
		}
	}
	return ""
}

func normalizeSavePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.EqualFold(filepath.Ext(path), ".zip") {
		path += ".zip"
	}
	return path
}

func copyFile(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func extractZip(zipPath, destination string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	cleanDestination := filepath.Clean(destination)
	for _, file := range reader.File {
		targetPath := filepath.Join(cleanDestination, file.Name)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDestination+string(os.PathSeparator)) && cleanTarget != cleanDestination {
			return fmt.Errorf("zip contains invalid path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.Create(cleanTarget)
		if err != nil {
			src.Close()
			return err
		}

		_, copyErr := io.Copy(dst, src)
		closeErr := errors.Join(dst.Close(), src.Close())
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}

	return nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func newSessionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
