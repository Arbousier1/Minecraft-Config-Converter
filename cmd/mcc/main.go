//go:build !windows

package main

import (
	"log"
	"net/http"
	"os/exec"
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
		Addr:    appAddr,
		Handler: app.Handler(),
	}
	app.SetHTTPServer(httpSrv)

	go openBrowser(appURL)
	log.Printf("starting MCC on %s", appURL)
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
