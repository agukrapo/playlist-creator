package main

import (
	"os"

	"fyne.io/fyne/v2"
	"github.com/agukrapo/playlist-creator/internal/env"
	"github.com/agukrapo/playlist-creator/internal/logs"
)

const appTitle = "playlist-creator"

func main() {
	cookie, err := env.Lookup[string]("DEEZER_ARL_COOKIE")
	if err != nil {
		fyne.LogError("env.Lookup", err)
	}

	logFile, err := logs.NewFile(appTitle)
	if err != nil {
		fyne.LogError("logs.NewFile", err)
		os.Exit(1)
	}
	defer logFile.Close()

	app := newApplication(cookie, logs.New(logFile))
	app.ShowAndRun()
}
