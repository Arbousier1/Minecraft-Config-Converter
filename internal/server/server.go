package server

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mccassets "github.com/Arbousier1/Minecraft-Config-Converter"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/analyzer"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/converter/iace"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/converter/nexoce"
)

const maxUploadSize = 500 << 20
const heartbeatTimeout = 15 * time.Second

type Plugin struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type reportPayload struct {
	Formats          []string              `json:"formats"`
	ContentTypes     []string              `json:"content_types"`
	Completeness     analyzer.Completeness `json:"completeness"`
	Details          analyzer.Details      `json:"details"`
	SourceFormats    []string              `json:"source_formats"`
	AvailableTargets []string              `json:"available_targets"`
	Warnings         []string              `json:"warnings"`
	Filename         string                `json:"filename"`
	SupportedPlugins []Plugin              `json:"supported_plugins"`
}

type Server struct {
	baseDir        string
	uploadDir      string
	outputDir      string
	mux            *http.ServeMux
	httpSrv        *http.Server
	lastHeartbeat  atomic.Int64
	shuttingDown   atomic.Bool
	monitorStart   sync.Once
	shutdownSignal sync.Once
}

func New(baseDir string) (*Server, error) {
	s := &Server{
		baseDir:   baseDir,
		uploadDir: filepath.Join(baseDir, "temp_uploads"),
		outputDir: filepath.Join(baseDir, "temp_output"),
		mux:       http.NewServeMux(),
	}

	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.outputDir, 0o755); err != nil {
		return nil, err
	}

	s.lastHeartbeat.Store(time.Now().UnixNano())
	s.registerRoutes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) SetHTTPServer(httpSrv *http.Server) {
	s.httpSrv = httpSrv
	s.monitorStart.Do(func() {
		go s.monitorHeartbeat()
	})
}

func (s *Server) Shutdown() {
	s.shutdown(false)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/analyze", s.handleAnalyze)
	s.mux.HandleFunc("/api/convert", s.handleConvert)
	s.mux.HandleFunc("/api/download/", s.handleDownload)
	s.mux.HandleFunc("/api/heartbeat", s.handleHeartbeat)
	s.mux.HandleFunc("/api/shutdown", s.handleShutdown)

	assetsFS, err := fs.Sub(mccassets.Files, "web/dist/assets")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	imageFS, err := fs.Sub(mccassets.Files, "web/static/images")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("/static/images/", http.StripPrefix("/static/images/", http.FileServer(http.FS(imageFS))))

	s.mux.HandleFunc("/", s.handleFrontend)
}

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	filePath := "web/dist" + path
	content, err := mccassets.Files.ReadFile(filePath)
	if err != nil {
		content, err = mccassets.Files.ReadFile("web/dist/index.html")
		if err != nil {
			http.Error(w, "failed to load frontend", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(content)
		return
	}

	if contentType := mime.TypeByExtension(filepath.Ext(filePath)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = w.Write(content)
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse upload"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	if header.Filename == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty filename"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "please upload a .zip file"})
		return
	}

	sessionID := newSessionID()
	sessionUploadDir := filepath.Join(s.uploadDir, sessionID)
	if err := os.MkdirAll(sessionUploadDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	zipPath := filepath.Join(sessionUploadDir, filepath.Base(header.Filename))
	if err := saveUpload(file, zipPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	extractDir := filepath.Join(sessionUploadDir, "extracted")
	if err := extractZip(zipPath, extractDir); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	report, err := analyzer.New(extractDir).Analyze()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	payload := buildReportPayload(report, header.Filename)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "success",
		"report":     payload,
		"session_id": sessionID,
	})
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse request"})
		return
	}

	targetFormat := r.FormValue("target_format")
	if targetFormat == "" {
		targetFormat = "CraftEngine"
	}
	sourceFormat := r.FormValue("source_format")
	namespace := r.FormValue("namespace")

	sessionID := r.FormValue("session_id")
	var (
		sessionUploadDir string
		sessionOutputDir string
		extractDir       string
		originalFilename string
	)

	if sessionID != "" {
		sessionUploadDir = filepath.Join(s.uploadDir, sessionID)
		sessionOutputDir = filepath.Join(s.outputDir, sessionID)
		extractDir = filepath.Join(sessionUploadDir, "extracted")
		if _, err := os.Stat(extractDir); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session expired or missing"})
			return
		}
		if err := os.MkdirAll(sessionOutputDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create output session"})
			return
		}
		originalFilename = detectOriginalFilename(sessionUploadDir)
	} else {
		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
			return
		}
		defer file.Close()

		if header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "please upload a .zip file"})
			return
		}

		sessionID = newSessionID()
		sessionUploadDir = filepath.Join(s.uploadDir, sessionID)
		sessionOutputDir = filepath.Join(s.outputDir, sessionID)
		if err := os.MkdirAll(sessionUploadDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create upload session"})
			return
		}
		if err := os.MkdirAll(sessionOutputDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create output session"})
			return
		}

		zipPath := filepath.Join(sessionUploadDir, filepath.Base(header.Filename))
		if err := saveUpload(file, zipPath); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		extractDir = filepath.Join(sessionUploadDir, "extracted")
		if err := extractZip(zipPath, extractDir); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		originalFilename = header.Filename
	}

	if targetFormat != "CraftEngine" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported target format: " + targetFormat})
		return
	}

	var (
		downloadURL string
		err         error
	)

	switch sourceFormat {
	case "Nexo":
		result, runErr := nexoce.Run(nexoce.Options{
			ExtractDir:       extractDir,
			SessionUploadDir: sessionUploadDir,
			SessionOutputDir: sessionOutputDir,
			OutputDir:        s.outputDir,
			OriginalFilename: originalFilename,
			UserNamespace:    namespace,
			TargetFormat:     targetFormat,
		})
		if result != nil {
			downloadURL = result.DownloadURL
		}
		err = runErr
	default:
		result, runErr := iace.Run(iace.Options{
			ExtractDir:       extractDir,
			SessionUploadDir: sessionUploadDir,
			SessionOutputDir: sessionOutputDir,
			OutputDir:        s.outputDir,
			OriginalFilename: originalFilename,
			UserNamespace:    namespace,
			TargetFormat:     targetFormat,
		})
		if result != nil {
			downloadURL = result.DownloadURL
		}
		err = runErr
	}
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case iace.IsValidationError(err):
			status = http.StatusBadRequest
		case nexoce.IsValidationError(err):
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "success",
		"download_url": downloadURL,
	})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/download/")
	name = filepath.Base(name)
	if name == "." || name == "" {
		http.NotFound(w, r)
		return
	}

	target := filepath.Join(s.outputDir, name)
	if _, err := os.Stat(target); err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, target)
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	s.lastHeartbeat.Store(time.Now().UnixNano())
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "server shutting down..."})
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.shutdown(false)
	}()
}

func (s *Server) monitorHeartbeat() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if s.shuttingDown.Load() {
			return
		}

		last := s.lastHeartbeat.Load()
		if last == 0 {
			continue
		}
		if time.Since(time.Unix(0, last)) > heartbeatTimeout {
			s.shutdown(true)
			return
		}
	}
}

func (s *Server) shutdown(forceExit bool) {
	s.shutdownSignal.Do(func() {
		s.shuttingDown.Store(true)
		if s.httpSrv != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.httpSrv.Shutdown(ctx)
		}
		if forceExit {
			os.Exit(0)
		}
	})
}

func buildReportPayload(report analyzer.Report, filename string) reportPayload {
	availableTargets := make([]string, 0, 1)
	warnings := make([]string, 0, 2)

	hasItemsAdder := contains(report.Formats, "ItemsAdder")
	hasCraftEngine := contains(report.Formats, "CraftEngine")
	hasNexo := contains(report.Formats, "Nexo")

	if hasItemsAdder || hasNexo {
		availableTargets = append(availableTargets, "CraftEngine")
	}
	if hasCraftEngine && (hasItemsAdder || hasNexo) {
		warnings = append(warnings, "Detected existing CraftEngine content. Conversion may overwrite or conflict with current files.")
	}

	return reportPayload{
		Formats:          report.Formats,
		ContentTypes:     report.ContentTypes,
		Completeness:     report.Completeness,
		Details:          report.Details,
		SourceFormats:    append([]string(nil), report.Formats...),
		AvailableTargets: availableTargets,
		Warnings:         warnings,
		Filename:         filename,
		SupportedPlugins: supportedPlugins(),
	}
}

func supportedPlugins() []Plugin {
	return []Plugin{
		{ID: "ItemsAdder", Name: "ItemsAdder", Icon: "/static/images/itemsadder.webp"},
		{ID: "Nexo", Name: "Nexo", Icon: "/static/images/nexo.webp"},
		{ID: "Oraxen", Name: "Oraxen", Icon: "/static/images/oraxen.webp"},
		{ID: "CraftEngine", Name: "CraftEngine", Icon: "/static/images/craftengine.webp"},
		{ID: "MythicCrucible", Name: "MythicCrucible", Icon: "/static/images/mythiccrucible.webp"},
	}
}

func saveUpload(file io.Reader, target string) error {
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	return err
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

func newSessionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func detectOriginalFilename(sessionUploadDir string) string {
	entries, err := os.ReadDir(sessionUploadDir)
	if err != nil {
		return "converted.zip"
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
			return entry.Name()
		}
	}
	return "converted.zip"
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
