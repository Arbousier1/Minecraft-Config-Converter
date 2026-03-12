//go:build windows

package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/server"
	webview2 "github.com/jchv/go-webview2"
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

	serveErrCh := make(chan error, 1)
	go func() {
		err := httpSrv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErrCh <- err
			return
		}
		serveErrCh <- nil
	}()

	if err := waitForServerReady(appURL, serveErrCh, 5*time.Second); err != nil {
		log.Fatalf("start server: %v", err)
	}

	w := webview2.New(false)
	if w == nil {
		log.Fatal("init webview failed")
	}
	defer w.Destroy()

	if err := w.Bind("quitApp", func() {
		go app.Shutdown()
		w.Terminate()
	}); err != nil {
		log.Fatalf("bind quitApp: %v", err)
	}

	w.SetTitle("MCC")
	w.SetSize(1280, 900, webview2.HintNone)
	w.Navigate(appURL)
	log.Printf("starting MCC desktop shell on %s", appURL)
	w.Run()

	app.Shutdown()
	if err := <-serveErrCh; err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
