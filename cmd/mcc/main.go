package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/server"
)

func main() {
	baseDir, err := executableDir()
	if err != nil {
		log.Fatalf("resolve base dir: %v", err)
	}

	app, err := server.New(baseDir)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	httpSrv := &http.Server{
		Addr:    ":5000",
		Handler: app.Handler(),
	}
	app.SetHTTPServer(httpSrv)

	go openBrowser("http://127.0.0.1:5000/")
	log.Printf("starting MCC Go rewrite on http://127.0.0.1:5000")
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func openBrowser(url string) {
	time.Sleep(1500 * time.Millisecond)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func executableDir() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		return cwd, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}
