package main

import (
	"log"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/desktopui"
)

func main() {
	baseDir, err := executableDir()
	if err != nil {
		log.Fatalf("resolve base dir: %v", err)
	}

	if err := desktopui.Run(baseDir); err != nil {
		log.Fatalf("run desktop app: %v", err)
	}
}
