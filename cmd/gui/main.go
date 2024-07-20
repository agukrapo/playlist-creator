package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	a := app.New()
	w := a.NewWindow("Playlist creator")
	w.Resize(fyne.NewSize(1000, 500))

	w.SetContent(widget.NewLabel("Hello World!"))
	w.ShowAndRun()

	return nil
}
